package services

import (
	"strconv"
	"strings"

	"github.com/pandatix/godepgraph/deploy/services/parts"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	netwv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type (
	GoDepGraph struct {
		pulumi.ResourceState

		n4j      *parts.Neo4J
		n4jSvc   *corev1.Service
		n4jNtp   *netwv1.NetworkPolicy
		n4jUiNtp *netwv1.NetworkPolicy

		gdg        *parts.GoDepGraph
		gdgSvc     *corev1.Service
		gdgNtp     *netwv1.NetworkPolicy
		gdgOtlpNtp *netwv1.NetworkPolicy
		gdgExpNtp  *netwv1.NetworkPolicy

		Endpoint       pulumi.StringOutput
		GoDepGraphPort pulumi.IntPtrOutput
		Neo4JUIPort    pulumi.IntPtrOutput
		Neo4JAPIPort   pulumi.IntPtrOutput
		Neo4JUser      pulumi.StringOutput
		Neo4JPass      pulumi.StringOutput
		Neo4JDBName    pulumi.StringOutput
	}

	GoDepGraphArgs struct {
		Namespace pulumi.StringInput
		Registry  pulumi.StringPtrInput

		GoDepGraphArgs GoDepGraphGoDepGraphArgs

		ExposeGoDepGraph bool
		ExposeNeo4J      bool
	}

	GoDepGraphGoDepGraphArgs struct {
		Tag      pulumi.StringPtrInput
		LogLevel pulumi.StringInput
		Replicas pulumi.IntPtrInput
		Requests pulumi.StringMapInput
		Limits   pulumi.StringMapInput
		Swagger  bool
	}
)

func NewGoDepGraph(ctx *pulumi.Context, name string, args *GoDepGraphArgs, opts ...pulumi.ResourceOption) (*GoDepGraph, error) {
	gdg := &GoDepGraph{}

	args = gdg.defaults(args)
	if err := ctx.RegisterComponentResource("pandatix:godepgraph", name, gdg, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(gdg))
	if err := gdg.provision(ctx, args, opts...); err != nil {
		return nil, err
	}
	if err := gdg.outputs(ctx, args); err != nil {
		return nil, err
	}

	return gdg, nil
}

func (gdg *GoDepGraph) defaults(args *GoDepGraphArgs) *GoDepGraphArgs {
	if args == nil {
		args = &GoDepGraphArgs{}
	}

	return args
}

func (gdg *GoDepGraph) provision(ctx *pulumi.Context, args *GoDepGraphArgs, opts ...pulumi.ResourceOption) (err error) {
	gdg.n4j, err = parts.NewNeo4J(ctx, "neo4j-db", &parts.Neo4JArgs{
		Namespace: args.Namespace,
		Registry:  args.Registry,
	}, opts...)
	if err != nil {
		return
	}

	gdg.gdg, err = parts.NewGoDepGraph(ctx, "godepgraph", &parts.GoDepGraphArgs{
		Tag:       args.GoDepGraphArgs.Tag,
		Registry:  args.Registry,
		LogLevel:  args.GoDepGraphArgs.LogLevel,
		Namespace: args.Namespace,
		Replicas:  args.GoDepGraphArgs.Replicas,
		Requests:  args.GoDepGraphArgs.Requests,
		Limits:    args.GoDepGraphArgs.Limits,
		Swagger:   args.GoDepGraphArgs.Swagger,
		Neo4J: parts.GoDepGraphNeo4JArgs{
			URI:      gdg.n4j.Endpoint,
			Username: gdg.n4j.Username,
			Password: gdg.n4j.Password,
		},
	}, opts...)
	if err != nil {
		return
	}

	// Create exposing services
	if args.ExposeNeo4J {
		gdg.n4jSvc, err = corev1.NewService(ctx, "neo4j-exposed", &corev1.ServiceArgs{
			Metadata: metav1.ObjectMetaArgs{
				Labels:    gdg.n4j.PodLabels,
				Namespace: args.Namespace,
			},
			Spec: corev1.ServiceSpecArgs{
				Type:     pulumi.String("NodePort"),
				Selector: gdg.n4j.PodLabels,
				Ports: corev1.ServicePortArray{
					corev1.ServicePortArgs{
						Name: pulumi.String("api"),
						Port: pulumi.Int(7687),
					},
					corev1.ServicePortArgs{
						Name: pulumi.String("ui"),
						Port: pulumi.Int(7474),
					},
				},
			},
		}, opts...)
		if err != nil {
			return
		}

		gdg.n4jUiNtp, err = netwv1.NewNetworkPolicy(ctx, "neo4j-ui-netpol", &netwv1.NetworkPolicyArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: args.Namespace,
				Labels:    gdg.n4j.PodLabels,
			},
			Spec: netwv1.NetworkPolicySpecArgs{
				PodSelector: metav1.LabelSelectorArgs{
					MatchLabels: gdg.n4j.PodLabels,
				},
				PolicyTypes: pulumi.ToStringArray([]string{
					"Ingress",
				}),
				Ingress: netwv1.NetworkPolicyIngressRuleArray{
					netwv1.NetworkPolicyIngressRuleArgs{
						From: netwv1.NetworkPolicyPeerArray{
							netwv1.NetworkPolicyPeerArgs{
								IpBlock: &netwv1.IPBlockArgs{
									Cidr: pulumi.String("0.0.0.0/0"),
								},
							},
						},
						Ports: netwv1.NetworkPolicyPortArray{
							netwv1.NetworkPolicyPortArgs{
								Port: gdg.n4jSvc.Spec.Ports().Index(pulumi.Int(0)).Port(),
							},
							netwv1.NetworkPolicyPortArgs{
								Port: gdg.n4jSvc.Spec.Ports().Index(pulumi.Int(1)).Port(),
							},
						},
					},
				},
			},
		}, opts...)
		if err != nil {
			return
		}
	}
	if args.ExposeGoDepGraph {
		gdg.gdgSvc, err = corev1.NewService(ctx, "godepgraph-exposed", &corev1.ServiceArgs{
			Metadata: metav1.ObjectMetaArgs{
				Labels:    gdg.gdg.PodLabels,
				Namespace: args.Namespace,
			},
			Spec: corev1.ServiceSpecArgs{
				Type:     pulumi.String("NodePort"),
				Selector: gdg.gdg.PodLabels,
				Ports: corev1.ServicePortArray{
					corev1.ServicePortArgs{
						Name: pulumi.String("api"),
						Port: gdg.gdg.Endpoint.ApplyT(func(edp string) int {
							_, port, _ := strings.Cut(edp, ":")
							iport, _ := strconv.Atoi(port)
							return iport
						}).(pulumi.IntOutput),
					},
				},
			},
		}, opts...)
		if err != nil {
			return
		}

		gdg.gdgExpNtp, err = netwv1.NewNetworkPolicy(ctx, "godepgraph-api-netpol", &netwv1.NetworkPolicyArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: args.Namespace,
				Labels:    gdg.gdg.PodLabels,
			},
			Spec: netwv1.NetworkPolicySpecArgs{
				PodSelector: metav1.LabelSelectorArgs{
					MatchLabels: gdg.gdg.PodLabels,
				},
				PolicyTypes: pulumi.ToStringArray([]string{
					"Ingress",
				}),
				Ingress: netwv1.NetworkPolicyIngressRuleArray{
					netwv1.NetworkPolicyIngressRuleArgs{
						From: netwv1.NetworkPolicyPeerArray{
							netwv1.NetworkPolicyPeerArgs{
								IpBlock: &netwv1.IPBlockArgs{
									Cidr: pulumi.String("0.0.0.0/0"),
								},
							},
						},
						Ports: netwv1.NetworkPolicyPortArray{
							netwv1.NetworkPolicyPortArgs{
								Port: gdg.gdgSvc.Spec.Ports().Index(pulumi.Int(0)).Port(),
							},
						},
					},
				},
			},
		}, opts...)
		if err != nil {
			return
		}
	}

	// NetworkPolicies
	gdg.n4jNtp, err = netwv1.NewNetworkPolicy(ctx, "neo4j-netpol", &netwv1.NetworkPolicyArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("neo4j"),
				"app.kubernetes.io/part-of":   pulumi.String("godepgraph"),
				"pandatix/stack-name":         pulumi.String(ctx.Stack()),
			},
		},
		Spec: netwv1.NetworkPolicySpecArgs{
			PolicyTypes: pulumi.ToStringArray([]string{
				"Ingress",
			}),
			PodSelector: metav1.LabelSelectorArgs{
				MatchLabels: gdg.n4j.PodLabels,
			},
			Ingress: netwv1.NetworkPolicyIngressRuleArray{
				netwv1.NetworkPolicyIngressRuleArgs{
					Ports: netwv1.NetworkPolicyPortArray{
						netwv1.NetworkPolicyPortArgs{
							Port: gdg.n4j.Endpoint.ApplyT(func(edp string) int {
								_, core, _ := strings.Cut(edp, ":")  // Filter out protocol (bolt://...)
								_, pStr, _ := strings.Cut(core, ":") // Get filter out port (some-name:port)
								p, _ := strconv.Atoi(pStr)
								return p
							}),
						},
					},
					From: netwv1.NetworkPolicyPeerArray{
						netwv1.NetworkPolicyPeerArgs{
							NamespaceSelector: metav1.LabelSelectorArgs{
								MatchLabels: pulumi.StringMap{
									"kubernetes.io/metadata.name": args.Namespace,
								},
							},
							PodSelector: metav1.LabelSelectorArgs{
								MatchLabels: gdg.gdg.PodLabels,
							},
						},
					},
				},
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	gdg.gdgNtp, err = netwv1.NewNetworkPolicy(ctx, "godepgraph-netpol", &netwv1.NetworkPolicyArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("neo4j"),
				"app.kubernetes.io/part-of":   pulumi.String("godepgraph"),
				"pandatix/stack-name":         pulumi.String(ctx.Stack()),
			},
		},
		Spec: netwv1.NetworkPolicySpecArgs{
			PolicyTypes: pulumi.ToStringArray([]string{
				"Egress",
			}),
			PodSelector: metav1.LabelSelectorArgs{
				MatchLabels: gdg.gdg.PodLabels,
			},
			Egress: netwv1.NetworkPolicyEgressRuleArray{
				netwv1.NetworkPolicyEgressRuleArgs{
					Ports: netwv1.NetworkPolicyPortArray{
						netwv1.NetworkPolicyPortArgs{
							Port: gdg.n4j.Endpoint.ApplyT(func(edp string) int {
								_, core, _ := strings.Cut(edp, ":")  // Filter out protocol (bolt://...)
								_, pStr, _ := strings.Cut(core, ":") // Get filter out port (some-name:port)
								p, _ := strconv.Atoi(pStr)
								return p
							}),
						},
					},
					To: netwv1.NetworkPolicyPeerArray{
						netwv1.NetworkPolicyPeerArgs{
							NamespaceSelector: metav1.LabelSelectorArgs{
								MatchLabels: pulumi.StringMap{
									"kubernetes.io/metadata.name": args.Namespace,
								},
							},
							PodSelector: metav1.LabelSelectorArgs{
								MatchLabels: gdg.n4j.PodLabels,
							},
						},
					},
				},
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	return
}

func (gdg *GoDepGraph) outputs(ctx *pulumi.Context, args *GoDepGraphArgs) error {
	gdg.Endpoint = gdg.gdg.Endpoint
	if args.ExposeNeo4J {
		gdg.Neo4JAPIPort = gdg.n4jSvc.Spec.Ports().Index(pulumi.Int(0)).NodePort()
		gdg.Neo4JUIPort = gdg.n4jSvc.Spec.Ports().Index(pulumi.Int(1)).NodePort()
	}
	gdg.Neo4JUser = gdg.n4j.Username
	gdg.Neo4JPass = gdg.n4j.Password
	gdg.Neo4JDBName = gdg.n4j.DBName
	if args.ExposeGoDepGraph {
		gdg.GoDepGraphPort = gdg.gdgSvc.Spec.Ports().Index(pulumi.Int(0)).NodePort()
	}

	return ctx.RegisterResourceOutputs(gdg, pulumi.Map{
		"endpoint":        gdg.Endpoint,
		"neo4j.api-port":  gdg.Neo4JAPIPort,
		"neo4j.ui-port":   gdg.Neo4JUIPort,
		"neo4j.user":      gdg.Neo4JUser,
		"neo4j.pass":      gdg.Neo4JPass,
		"neo4j.dbname":    gdg.Neo4JDBName,
		"godepgraph.port": gdg.GoDepGraphPort,
	})
}
