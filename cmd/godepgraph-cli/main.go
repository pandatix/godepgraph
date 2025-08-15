package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	apiv1alg4 "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/api/v1/alg4"
	apiv1cdn "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/api/v1/cdn"
	apiv1rdg "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/api/v1/rdg"
	apiv1sig "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/api/v1/sig"
	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/global"
)

type (
	cliCdnKey  struct{}
	cliRdgKey  struct{}
	cliSigKey  struct{}
	cliAlg4Key struct{}
)

func main() {
	app := &cli.Command{
		Name:  "godepgraph-cli",
		Usage: "CLI for GoDepGraph, for test purposes.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "url",
				Usage:    "The URL to reach out GoDepGraph.",
				Required: true,
			},
		},
		Commands: []*cli.Command{
			{
				Name: "cdn",
				Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
					conn, err := grpc.NewClient(cmd.String("url"),
						grpc.WithTransportCredentials(insecure.NewCredentials()),
					)
					if err != nil {
						return ctx, err
					}
					cliCdn := apiv1cdn.NewCDNClient(conn)

					return context.WithValue(ctx, cliCdnKey{}, cliCdn), nil
				},
				Commands: []*cli.Command{
					{
						Name: "create",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "version",
								Required: true,
							},
							&cli.BoolFlag{
								Name: "test",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							cliCdn := ctx.Value(cliCdnKey{}).(apiv1cdn.CDNClient)

							fmt.Println("    Creating library...")
							name, ver := cmd.String("name"), cmd.String("version")
							before := time.Now()
							_, err := cliCdn.CreateLibrary(ctx, &apiv1cdn.CreateLibraryRequest{
								Name:    name,
								Version: ver,
								Test:    ptr(cmd.Bool("test")),
							})
							dur := time.Since(before)
							if err != nil {
								return err
							}
							fmt.Printf("[+] Library %s@%s created in %v\n", name, ver, dur)

							return nil
						},
					}, {
						Name:  "reset",
						Usage: "Reset the global knowledge of a codebase.",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							cliCdn := ctx.Value(cliCdnKey{}).(apiv1cdn.CDNClient)

							fmt.Println("    Reseting CDN...")
							if _, err := cliCdn.Reset(ctx, nil); err != nil {
								return err
							}

							fmt.Println("[-] Reseted CDN")
							return nil
						},
					},
				},
			}, {
				Name: "rdg",
				Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
					conn, err := grpc.NewClient(cmd.String("url"),
						grpc.WithTransportCredentials(insecure.NewCredentials()),
					)
					if err != nil {
						return ctx, err
					}
					cliRdg := apiv1rdg.NewRDGClient(conn)

					return context.WithValue(ctx, cliRdgKey{}, cliRdg), nil
				},
				Commands: []*cli.Command{
					{
						Name:  "create",
						Usage: "Create a RDG from a Pulumi state file.",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "uri",
								Usage:    "The URI from where to load the file. Could be https://webserver.tld/file.json or file:///tmp/state.json",
								Required: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							cliRdg := ctx.Value(cliRdgKey{}).(apiv1rdg.RDGClient)

							fmt.Println("    Creating stack...")

							before := time.Now()
							_, err := cliRdg.CreateStack(ctx, &apiv1rdg.CreateStackRequest{
								Uri: cmd.String("uri"),
							})
							dur := time.Since(before)
							if err != nil {
								return err
							}

							fmt.Printf("[+] Stack created in %v\n", dur)

							return nil
						},
					}, {
						Name:  "reset",
						Usage: "Resets existing RDG(s).",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							cliRdg := ctx.Value(cliRdgKey{}).(apiv1rdg.RDGClient)

							fmt.Println("    Reseting RDG...")
							if _, err := cliRdg.Reset(ctx, nil); err != nil {
								return err
							}

							fmt.Println("[-] Reseted RDG")
							return nil
						},
					},
				},
			}, {
				Name: "sig",
				Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
					conn, err := grpc.NewClient(cmd.String("url"),
						grpc.WithTransportCredentials(insecure.NewCredentials()),
					)
					if err != nil {
						return ctx, err
					}
					cliSig := apiv1sig.NewSIGClient(conn)

					return context.WithValue(ctx, cliSigKey{}, cliSig), nil
				},
				Commands: []*cli.Command{
					{
						Name: "create",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "file",
								Required: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							cliSig := ctx.Value(cliSigKey{}).(apiv1sig.SIGClient)

							fmt.Println("    Parsing OTEL dump...")
							f, err := os.Open(cmd.String("file"))
							if err != nil {
								return err
							}
							defer func() {
								_ = f.Close()
							}()

							ana, err := newSigAnalysis(f)
							if err != nil {
								return err
							}

							fmt.Println("    Creating observed architecture...")
							before := time.Now()
							for _, comp := range ana.components {
								if _, err := cliSig.CreateComponent(ctx, &apiv1sig.CreateComponentRequest{
									Name:    comp.Name,
									Version: comp.Version,
								}); err != nil {
									return err
								}
							}

							for _, comp := range ana.components {
								for _, it := range comp.Interactions {
									if _, err := cliSig.CreateNetworkDependency(ctx, &apiv1sig.CreateNetworkDependencyRequest{
										Caller: &apiv1sig.CreateNetworkDependencyEndpointRequest{
											Name: it.From,
											Exposes: &apiv1sig.CreateNetworkDependencyEndpointComponentRequest{
												Name:    comp.Name,
												Version: comp.Version,
											},
										},
										Callees: []*apiv1sig.CreateNetworkDependencyEndpointRequest{
											{
												Name: it.Name,
												Exposes: &apiv1sig.CreateNetworkDependencyEndpointComponentRequest{
													Name:    it.To,
													Version: ana.components[it.To].Version,
												},
											},
										},
									}); err != nil {
										return err
									}
								}
							}
							dur := time.Since(before)

							fmt.Printf("[+] Observed architecture created in %v\n", dur)

							return nil
						},
					}, {
						Name:  "reset",
						Usage: "Reset the global knowledge of the system under observation.",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							cliSig := ctx.Value(cliSigKey{}).(apiv1sig.SIGClient)

							fmt.Println("    Reseting SIG...")
							if _, err := cliSig.Reset(ctx, nil); err != nil {
								return err
							}

							fmt.Println("[-] Reseted SIG")
							return nil
						},
					},
				},
			}, {
				Name: "alg4",
				Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
					conn, err := grpc.NewClient(cmd.String("url"),
						grpc.WithTransportCredentials(insecure.NewCredentials()),
					)
					if err != nil {
						return ctx, err
					}
					cliAlg4 := apiv1alg4.NewAlg4Client(conn)

					return context.WithValue(ctx, cliAlg4Key{}, cliAlg4), nil
				},
				Commands: []*cli.Command{
					{
						Name: "binding",
						Commands: []*cli.Command{
							{
								Name: "create",
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name: "library.name",
									},
									&cli.StringFlag{
										Name: "library.version",
									},
									&cli.StringFlag{
										Name: "component.name",
									},
									&cli.StringFlag{
										Name: "component.version",
									},
									&cli.StringFlag{
										Name: "asset.name",
									},
									&cli.StringFlag{
										Name: "asset.version",
									},
								},
								Action: func(ctx context.Context, cmd *cli.Command) error {
									cliAlg4 := ctx.Value(cliAlg4Key{}).(apiv1alg4.Alg4Client)

									req := &apiv1alg4.CreateBindingRequest{}

									if cmd.IsSet("library.name") || cmd.IsSet("library.version") {
										req.Library = &apiv1alg4.LibraryOrRefinement{
											Name:    cmd.String("library.name"),
											Version: cmd.String("library.version"),
										}
									}
									if cmd.IsSet("component.name") || cmd.IsSet("component.version") {
										req.Component = &apiv1alg4.LibraryOrRefinement{
											Name:    cmd.String("component.name"),
											Version: cmd.String("component.version"),
										}
									}
									if cmd.IsSet("asset.name") || cmd.IsSet("asset.version") {
										req.Asset = &apiv1alg4.LibraryOrRefinement{
											Name:    cmd.String("asset.name"),
											Version: cmd.String("asset.version"),
										}
									}

									if _, err := cliAlg4.CreateBinding(ctx, req); err != nil {
										return err
									}

									return nil
								},
							},
						},
					}, {
						Name: "serves",
						Commands: []*cli.Command{
							{
								Name: "create",
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "symbol.identity",
										Required: true,
									},
									&cli.StringFlag{
										Name:     "endpoint.name",
										Required: true,
									},
									&cli.StringFlag{
										Name:     "component.name",
										Required: true,
									},
									&cli.StringFlag{
										Name:     "component.version",
										Required: true,
									},
								},
								Action: func(ctx context.Context, cmd *cli.Command) error {
									cliAlg4 := ctx.Value(cliAlg4Key{}).(apiv1alg4.Alg4Client)

									if _, err := cliAlg4.CreateServes(ctx, &apiv1alg4.CreateServesRequest{
										Symbol: &apiv1alg4.CreateServesSymbolRequest{
											Identity: cmd.String("symbol.identity"),
										},
										Endpoint: &apiv1alg4.CreateServesEndpointRequest{
											Name: cmd.String("endpoint.name"),
											Exposes: &apiv1alg4.CreateServesComponentRequest{
												Name:    cmd.String("component.name"),
												Version: cmd.String("component.version"),
											},
										},
									}); err != nil {
										return err
									}

									return nil
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	if err := app.Run(ctx, os.Args); err != nil {
		global.Log().Error(ctx, "fatal error",
			zap.Error(err),
		)
		os.Exit(1)
	}
}

func ptr[T any](t T) *T {
	return &t
}

type (
	Component struct {
		Name         string
		Version      string
		Interactions []*Interaction // 1..*
	}

	Interaction struct {
		Timestamp time.Time
		To        string
		Name      string
		From      string
	}
)

type sigAnalysis struct {
	components map[string]*Component
	notFound   map[string]map[string]struct{}
}

func newSigAnalysis(f io.Reader) (*sigAnalysis, error) {
	// Get all spans per traces and service names
	traces, err := parseTraces(f)
	if err != nil {
		return nil, err
	}

	ana := &sigAnalysis{
		components: map[string]*Component{},
		notFound:   map[string]map[string]struct{}{},
	}
	for _, trace := range traces {
		for _, span := range trace.Spans {
			// Register this component if not already
			if _, ok := ana.components[span.SVCName]; !ok {
				ana.components[span.SVCName] = &Component{
					Name:         span.SVCName,
					Version:      span.SVCVersion,
					Interactions: []*Interaction{},
				}
			}

			// Get parent span
			pid := span.Sub.ParentSpanID()
			if pid.IsEmpty() {
				// Incomplete traces, accept to lose this interaction
				continue
			}
			parent, ok := trace.Spans[hex.EncodeToString(pid[:])]
			if !ok {
				if _, ok := ana.notFound[trace.ID]; !ok {
					ana.notFound[trace.ID] = map[string]struct{}{}
				}
				ana.notFound[trace.ID][fmt.Sprintf("%x", pid)] = struct{}{}

				// Incomplete traces, accept to lose this interaction
				continue
			}

			// A parent is interesting if it comes from another service
			if parent.SVCName == span.SVCName {
				continue
			}

			// For every upstream interaction, register if not already
			pcomp, ok := ana.components[parent.SVCName]
			if !ok {
				pcomp = &Component{
					Name:         parent.SVCName,
					Version:      parent.SVCVersion,
					Interactions: []*Interaction{},
				}
				ana.components[parent.SVCName] = pcomp
			}

			// Then register the interaction from parent to current one
			from, found := getFrom(trace, parent)
			switch found {
			case STATUS_PARENT_NOT_FOUND:
				if _, ok := ana.notFound[trace.ID]; !ok {
					ana.notFound[trace.ID] = map[string]struct{}{}
				}
				ana.notFound[trace.ID][fmt.Sprintf("%x", pid)] = struct{}{}

				// TODO if case STATUS_NO_CALLER_FOUND then we might create another endpoint rather than assigning an empty endpoint that aggregates every known interactions from outside our scope
			}
			pcomp.Interactions = append(pcomp.Interactions, &Interaction{
				Timestamp: span.Sub.StartTimestamp().AsTime(),
				To:        span.SVCName,
				Name: func() string {
					// Thanks to RPC instrumentation the parent has a good name
					// as it emitted the call, so we prefer it.
					_, hasRPCSystem := parent.Sub.Attributes().Get("rpc.system")
					if hasRPCSystem {
						return parent.Sub.Name()
					}
					return span.Sub.Name()
				}(),
				From: from,
			})
		}
	}

	if len(ana.notFound) > 0 {
		fmt.Println("The following traces are incomplete.")
		for trace, spans := range ana.notFound {
			fmt.Printf("- Trace %s:\n", trace)
			for span := range spans {
				fmt.Printf("  Span %s not found\n", span)
			}
		}
	}

	return ana, nil
}

type Status int

const (
	STATUS_FOUND Status = iota
	STATUS_PARENT_NOT_FOUND
	STATUS_NO_CALLER_FOUND
)

func getFrom(trace Trace, span Span) (from string, status Status) {
	// Get the parent span from this one
	pid := span.Sub.ParentSpanID()
	if pid.IsEmpty() {
		// Incomplete traces, accept to lose this interaction
		return "unknown", STATUS_NO_CALLER_FOUND
	}
	parent, ok := trace.Spans[hex.EncodeToString(pid[:])]
	if !ok {
		// Incomplete traces, accept to lose this information
		return "unknown", STATUS_PARENT_NOT_FOUND
	}

	// If the parent span reaches another network interaction, seems like the
	// spans involved in the trace did not use the caller.function attribute...
	if _, hasRPCSystem := parent.Sub.Attributes().Get("rpc.system"); hasRPCSystem {
		return "unknown", STATUS_NO_CALLER_FOUND
	}

	// If this span contains the information we are look for, stop here !
	if cf, hasCallerFunc := parent.Sub.Attributes().Get("caller.function"); hasCallerFunc {
		return cf.AsString(), STATUS_FOUND
	}

	// Else keep going...
	return getFrom(trace, parent)
}

// Traces per ID, then Span per ID (for quick search)
type Traces map[string]Trace

type Trace struct {
	ID    string
	Spans map[string]Span
}

type Span struct {
	Sub        ptrace.Span
	SVCName    string
	SVCVersion string
}

// parseTraces reads the input io.Reader until completly consumed.
// It returns a map of traces identified by their ID, then a map
// of spanId in each trace.
func parseTraces(f io.Reader) (Traces, error) {
	tracesUnmarshaler := &ptrace.JSONUnmarshaler{}
	out := Traces{}

	scan := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scan.Buffer(buf, 1024*1024) // increase buffer to make sure large traces can be processed
	for scan.Scan() {
		t, err := tracesUnmarshaler.UnmarshalTraces(scan.Bytes())
		if err != nil {
			return nil, err
		}

		if t.ResourceSpans().Len() != 1 {
			return nil, fmt.Errorf("multiple resource spans %v", t)
		}

		rs := t.ResourceSpans().At(0)
		attrs := rs.Resource().Attributes().AsRaw()
		svcName, hasSvcName := attrs["service.name"]
		if !hasSvcName {
			// resource span should have a service name
			return nil, fmt.Errorf("resource span has no service name")
		}
		svcVersion, hasSvcVersion := attrs["service.version"]
		if !hasSvcVersion {
			svcVersion = "unknown"
		}
		ss := rs.ScopeSpans()
		for i := 0; i < ss.Len(); i++ {
			s := ss.At(i)
			spans := s.Spans()
			for j := 0; j < spans.Len(); j++ {
				span := spans.At(j)

				ttid := span.TraceID()
				tid := hex.EncodeToString(ttid[:])
				ssid := span.SpanID()
				sid := hex.EncodeToString(ssid[:])

				if _, ok := out[tid]; !ok {
					out[tid] = Trace{
						ID:    tid,
						Spans: map[string]Span{},
					}
				}
				out[tid].Spans[sid] = Span{
					Sub:        span,
					SVCName:    svcName.(string),
					SVCVersion: svcVersion.(string),
				}
			}
		}
	}
	return out, scan.Err()
}
