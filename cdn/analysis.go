package cdn

import (
	"context"
	"encoding/json"
	"fmt"
	"go/build"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

var DefaultAnalyserParams = &AnalyserParams{
	Tests: false,
}

type Analyser struct {
	*AnalyserParams
}

type AnalyserParams struct {
	Tests bool
}

type Analysis struct {
	modules  map[string]*Module
	packages map[string]*Package
}

func NewAnalyser(params *AnalyserParams) *Analyser {
	if params == nil {
		params = DefaultAnalyserParams
	}
	return &Analyser{
		AnalyserParams: params,
	}
}

func (a *Analyser) Process(ctx context.Context, mod, version string) (*Analysis, *Module, error) {
	fmt.Printf("Processing %s at %s\n", mod, version)

	// Download the Go module (does nothing if already in $GOMODCACHE)
	if err := a.download(mod, version); err != nil {
		return nil, nil, err
	}

	// Run the analysis
	ana := &Analysis{
		modules:  map[string]*Module{},
		packages: map[string]*Package{},
	}
	m, err := ana.parseModule(mod, version, a.Tests)
	if err != nil {
		return nil, nil, err
	}

	// Remove duplicates to avoid duplicating edges
	ana.removeEmpty()

	return ana, m, nil
}

func (ana *Analyser) download(mod, version string) error {
	if err := module.CheckPath(mod); err != nil {
		return err
	}
	out, err := exec.Command("go", "mod", "download", mod+"@"+version).CombinedOutput()
	if err != nil {
		fmt.Printf("out: %v\n", out)
	}
	return err
}

func (ana *Analysis) parseModule(mod, version string, tests bool) (*Module, error) {
	dir := filepath.Join(gomodcache(), fmt.Sprintf("%s@%s", mod, version))

	module := &Module{
		Name:     mod,
		Version:  version,
		Packages: []*Package{},
	}
	ana.modules[mod] = module

	// Get modules
	mr, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return nil, err
	}
	mc, err := modfile.Parse("", mr, nil)
	if err != nil {
		return nil, err
	}
	for _, use := range mc.Require {
		ana.modules[use.Mod.Path] = &Module{
			Name:     use.Mod.Path,
			Version:  use.Mod.Version,
			Packages: []*Package{},
		}
	}

	// Walk in filesystem to determine which are packages we may use
	packages := []*Package{}
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		// Compute relative path to identify the package by the filesystem structure
		rel := strings.TrimPrefix(path, dir)
		pkg := &Package{
			Name:      filepath.Join(mod, rel),
			Functions: map[string]*Function{},
		}

		_, _, graph, err := Analyze(path, tests)
		if err != nil {
			return err
		}

		register := false
		for f, v := range graph.Nodes {
			// If not in the package we are analysing, don't track it specifically
			if f == nil || f.Pkg == nil || f.Pkg.Pkg == nil ||
				(f.Pkg != nil && !strings.HasPrefix(fmt.Sprint(f.Pkg.Pkg.Path()), pkg.Name)) {
				continue
			}
			register = true
			ana.register(v)
		}
		if register {
			packages = append(packages, pkg)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	module.Packages = packages

	return module, nil
}

func (ana *Analysis) register(v *callgraph.Node) {
	// Register this function
	// Get the node's package corresponding resource in the analysis
	pkgStr := v.Func.Pkg.Pkg.Path()
	pkg, ok := ana.packages[pkgStr]
	if !ok {
		pkg = &Package{
			Name:      pkgStr,
			Functions: map[string]*Function{},
		}
		ana.packages[pkgStr] = pkg

		modName := ""
		for name := range ana.modules {
			if len(name) > len(modName) && strings.HasPrefix(pkgStr, name) {
				modName = name
			}
		}
		mod, ok := ana.modules[modName]
		if ok {
			mod.Packages = append(mod.Packages, pkg)
		}
	}

	fStr := v.Func.String()
	f := &Function{
		Identity:     fStr,
		Package:      pkg,
		Dependencies: map[string]map[string]struct{}{},
	}
	pkg.Functions[fStr] = f

	// First recurse on dependencies until we don't have any thus
	// the dependency graph could be satisfied (bottom up)
	for _, edge := range v.Out {
		if edge.Callee.Func.Pkg == nil || edge.Callee.Func.Pkg.Pkg == nil {
			continue
		}

		pkgStr := edge.Callee.Func.Pkg.Pkg.Path()
		if _, ok := ana.packages[pkgStr]; !ok {
			ana.register(edge.Callee)
		}
		fStr := edge.Callee.Func.String()
		if _, ok := ana.packages[pkgStr].Functions[fStr]; !ok {
			ana.register(edge.Callee)
		}

		if _, ok := f.Dependencies[pkgStr]; !ok {
			f.Dependencies[pkgStr] = map[string]struct{}{}
		}
		f.Dependencies[pkgStr][fStr] = struct{}{}
	}
}

func Analyze(dir string, tests bool) (*ssa.Program, []*ssa.Package, *callgraph.Graph, error) {
	cfg := &packages.Config{
		Mode:       packages.LoadAllSyntax,
		Tests:      tests,
		Dir:        dir,
		BuildFlags: buildFlags(),
		Env:        append(os.Environ(), "GOWORK=off"), // disable go workspaces
	}

	initial, err := packages.Load(cfg)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "loading module at %s", dir)
	}

	// Create and build SSA-form program representation.
	mode := ssa.InstantiateGenerics
	prog, pkgs := ssautil.AllPackages(initial, mode)
	prog.Build()

	graph := cha.CallGraph(prog)

	return prog, pkgs, graph, nil
}

func buildFlags() []string {
	buildFlagTags := buildFlagTags(build.Default.BuildTags)
	if len(buildFlagTags) == 0 {
		return nil
	}

	return []string{buildFlagTags}
}

func buildFlagTags(buildTags []string) string {
	if len(buildTags) > 0 {
		return "-tags=" + strings.Join(buildTags, ",")
	}

	return ""
}

func (ana *Analysis) removeEmpty() {
	for _, m := range ana.modules {
		if len(m.Packages) == 0 {
			delete(ana.modules, m.Name)
		}
	}
}

var _ json.Marshaler = (*Analysis)(nil)

func (ana Analysis) MarshalJSON() ([]byte, error) {
	funcs := Functions{}
	for _, pkg := range ana.packages {
		merge(funcs, pkg.Functions)
	}
	return json.Marshal(funcs)
}

func merge[K comparable, V any](a, b map[K]V) map[K]V {
	for k, v := range b {
		a[k] = v
	}
	return a
}
