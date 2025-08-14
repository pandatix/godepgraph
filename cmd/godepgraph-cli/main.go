package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	apiv1alg4 "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/api/v1/alg4"
	apiv1cdn "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/api/v1/cdn"
	apiv1rdg "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/api/v1/rdg"
	apiv1sig "git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/api/v1/sig"
)

type (
	cliCdnKey  struct{}
	cliRdgKey  struct{}
	cliSigKey  struct{}
	cliAlg4Key struct{}
)

func main() {
	app := &cli.App{
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
				Before: func(ctx *cli.Context) error {
					conn, err := grpc.NewClient(ctx.String("url"),
						grpc.WithTransportCredentials(insecure.NewCredentials()),
					)
					if err != nil {
						return err
					}
					cliCdn := apiv1cdn.NewCDNClient(conn)

					ctx.Context = context.WithValue(ctx.Context, cliCdnKey{}, cliCdn)
					return nil
				},
				Subcommands: []*cli.Command{
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
						Action: func(ctx *cli.Context) error {
							cliCdn := ctx.Context.Value(cliCdnKey{}).(apiv1cdn.CDNClient)

							fmt.Println("    Creating library...")
							name, ver := ctx.String("name"), ctx.String("version")
							before := time.Now()
							_, err := cliCdn.CreateLibrary(ctx.Context, &apiv1cdn.CreateLibraryRequest{
								Name:    name,
								Version: ver,
								Test:    ptr(ctx.Bool("test")),
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
						Action: func(ctx *cli.Context) error {
							cliCdn := ctx.Context.Value(cliCdnKey{}).(apiv1cdn.CDNClient)

							fmt.Println("    Reseting CDN...")
							if _, err := cliCdn.Reset(ctx.Context, nil); err != nil {
								return err
							}

							fmt.Println("[-] Reseted CDN")
							return nil
						},
					},
				},
			}, {
				Name: "rdg",
				Before: func(ctx *cli.Context) error {
					conn, err := grpc.NewClient(ctx.String("url"),
						grpc.WithTransportCredentials(insecure.NewCredentials()),
					)
					if err != nil {
						return err
					}
					cliRdg := apiv1rdg.NewRDGClient(conn)

					ctx.Context = context.WithValue(ctx.Context, cliRdgKey{}, cliRdg)
					return nil
				},
				Subcommands: []*cli.Command{
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
						Action: func(ctx *cli.Context) error {
							cliRdg := ctx.Context.Value(cliRdgKey{}).(apiv1rdg.RDGClient)

							fmt.Println("    Creating stack...")

							before := time.Now()
							_, err := cliRdg.CreateStack(ctx.Context, &apiv1rdg.CreateStackRequest{
								Uri: ctx.String("uri"),
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
						Action: func(ctx *cli.Context) error {
							cliRdg := ctx.Context.Value(cliRdgKey{}).(apiv1rdg.RDGClient)

							fmt.Println("    Reseting RDG...")
							if _, err := cliRdg.Reset(ctx.Context, nil); err != nil {
								return err
							}

							fmt.Println("[-] Reseted RDG")
							return nil
						},
					},
				},
			}, {
				Name: "sig",
				Before: func(ctx *cli.Context) error {
					conn, err := grpc.NewClient(ctx.String("url"),
						grpc.WithTransportCredentials(insecure.NewCredentials()),
					)
					if err != nil {
						return err
					}
					cliSig := apiv1sig.NewSIGClient(conn)

					ctx.Context = context.WithValue(ctx.Context, cliSigKey{}, cliSig)
					return nil
				},
				Subcommands: []*cli.Command{
					{
						Name: "create",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "file",
								Required: true,
							},
						},
						Action: func(ctx *cli.Context) error {
							cliSig := ctx.Context.Value(cliSigKey{}).(apiv1sig.SIGClient)

							fmt.Println("    Parsing OTEL dump...")
							f, err := os.Open(ctx.String("file"))
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
								if _, err := cliSig.CreateComponent(ctx.Context, &apiv1sig.CreateComponentRequest{
									Name:    comp.Name,
									Version: comp.Version,
								}); err != nil {
									return err
								}
							}

							for _, comp := range ana.components {
								for _, it := range comp.Interactions {
									if _, err := cliSig.CreateNetworkDependency(ctx.Context, &apiv1sig.CreateNetworkDependencyRequest{
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
						Action: func(ctx *cli.Context) error {
							cliSig := ctx.Context.Value(cliSigKey{}).(apiv1sig.SIGClient)

							fmt.Println("    Reseting SIG...")
							if _, err := cliSig.Reset(ctx.Context, nil); err != nil {
								return err
							}

							fmt.Println("[-] Reseted SIG")
							return nil
						},
					},
				},
			}, {
				Name: "alg4",
				Before: func(ctx *cli.Context) error {
					conn, err := grpc.NewClient(ctx.String("url"),
						grpc.WithTransportCredentials(insecure.NewCredentials()),
					)
					if err != nil {
						return err
					}
					cliAlg4 := apiv1alg4.NewAlg4Client(conn)

					ctx.Context = context.WithValue(ctx.Context, cliAlg4Key{}, cliAlg4)
					return nil
				},
				Subcommands: []*cli.Command{
					{
						Name: "binding",
						Subcommands: []*cli.Command{
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
								Action: func(ctx *cli.Context) error {
									cliAlg4 := ctx.Context.Value(cliAlg4Key{}).(apiv1alg4.Alg4Client)

									req := &apiv1alg4.CreateBindingRequest{}

									if ctx.IsSet("library.name") || ctx.IsSet("library.version") {
										req.Library = &apiv1alg4.LibraryOrRefinement{
											Name:    ctx.String("library.name"),
											Version: ctx.String("library.version"),
										}
									}
									if ctx.IsSet("component.name") || ctx.IsSet("component.version") {
										req.Component = &apiv1alg4.LibraryOrRefinement{
											Name:    ctx.String("component.name"),
											Version: ctx.String("component.version"),
										}
									}
									if ctx.IsSet("asset.name") || ctx.IsSet("asset.version") {
										req.Asset = &apiv1alg4.LibraryOrRefinement{
											Name:    ctx.String("asset.name"),
											Version: ctx.String("asset.version"),
										}
									}

									if _, err := cliAlg4.CreateBinding(ctx.Context, req); err != nil {
										return err
									}

									return nil
								},
							},
						},
					}, {
						Name: "serves",
						Subcommands: []*cli.Command{
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
								Action: func(ctx *cli.Context) error {
									cliAlg4 := ctx.Context.Value(cliAlg4Key{}).(apiv1alg4.Alg4Client)

									if _, err := cliAlg4.CreateServes(ctx.Context, &apiv1alg4.CreateServesRequest{
										Symbol: &apiv1alg4.CreateServesSymbolRequest{
											Identity: ctx.String("symbol.identity"),
										},
										Endpoint: &apiv1alg4.CreateServesEndpointRequest{
											Name: ctx.String("endpoint.name"),
											Exposes: &apiv1alg4.CreateServesComponentRequest{
												Name:    ctx.String("component.name"),
												Version: ctx.String("component.version"),
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

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
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
}

func newSigAnalysis(f io.Reader) (*sigAnalysis, error) {
	// Get all spans per traces and service names
	traces, err := parseTraces(f)
	if err != nil {
		return nil, err
	}

	ana := &sigAnalysis{
		components: map[string]*Component{},
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
				fmt.Printf("Span %x in trace %s not found\n", pid, trace.ID)
				// Incomplete traces, accept to lose this interaction
				continue
			}

			// A parent is interesting if it is has a non-empty attribute rpc.system or http.scheme
			// -> it was issued from one service to another, this one is the targetted system
			_, hasRPCSystem := parent.Sub.Attributes().Get("rpc.system")
			_, hasHTTPURL := parent.Sub.Attributes().Get("http.url")
			if !(hasRPCSystem || hasHTTPURL) {
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
				fmt.Printf("Span %x in trace %s not found\n", pid, trace.ID)

				// TODO if case STATUS_NO_CALLER_FOUND then we might create another endpoint rather than assigning an empty endpoint that aggregates every known interactions from outside our scope
			}
			pcomp.Interactions = append(pcomp.Interactions, &Interaction{
				Timestamp: span.Sub.StartTimestamp().AsTime(),
				To:        span.SVCName,
				Name: func() string {
					if hasRPCSystem {
						return parent.Sub.Name()
					}
					return fmt.Sprintf("%s %s", parent.Sub.Name(), span.Sub.Attributes().AsRaw()["url.path"]) // XXX should be in CM span
				}(), // It is the parent who emitted the RPC
				From: from,
			})
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
		fmt.Printf("Span %x in trace %s not found\n", pid, trace.ID)

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
