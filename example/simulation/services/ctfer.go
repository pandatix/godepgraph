package services

import (
	"sync"

	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	netwv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"go.uber.org/multierr"

	"github.com/pandatix/godepgraph/example/simulation/services/parts"
)

// CTFer is a pulumi Component that deploy a pre-configured CTFd stack
// in an on-premise K8s cluster with Traefik as Ingress Controller.
type CTFer struct {
	pulumi.ResourceState

	maria *parts.MariaDB
	redis *parts.Redis
	ctfd  *parts.CTFd

	ctfdNetpol *netwv1.NetworkPolicy

	// URL contains the CTFd's URL once provided.
	URL       pulumi.StringOutput
	PodLabels pulumi.StringMapOutput
}

type CTFerArgs struct {
	Namespace       pulumi.StringInput
	CTFdImage       pulumi.StringInput
	ChallManagerUrl pulumi.StringInput

	CTFdCrt         pulumi.StringInput
	CTFdKey         pulumi.StringInput
	CTFdStorageSize pulumi.StringInput
	CTFdWorkers     pulumi.IntInput
	CTFdReplicas    pulumi.IntInput
	CTFdRequests    pulumi.StringMapInput
	CTFdLimits      pulumi.StringMapInput

	Hostname         pulumi.StringInput
	ChartsRepository pulumi.StringInput
	ImagesRepository pulumi.StringInput
	StorageClassName pulumi.StringInput

	IngressNamespace pulumi.StringInput
	IngressLabels    pulumi.StringMapInput

	Otel *parts.OtelArgs
}

// NewCTFer creates a new pulumi Component Resource and registers it.
func NewCTFer(ctx *pulumi.Context, name string, args *CTFerArgs, opts ...pulumi.ResourceOption) (*CTFer, error) {
	ctfer := &CTFer{}
	args = ctfer.defaults(args)
	if err := ctfer.check(args); err != nil {
		return nil, err
	}
	err := ctx.RegisterComponentResource("ctfer-io:ctfer", name, ctfer, opts...)
	if err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(ctfer))

	if err := ctfer.provision(ctx, args, opts...); err != nil {
		return nil, err
	}
	if err := ctfer.outputs(ctx); err != nil {
		return nil, err
	}

	return ctfer, nil
}

func (ctfer *CTFer) defaults(args *CTFerArgs) *CTFerArgs {
	if args == nil {
		args = &CTFerArgs{}
	}
	return args
}

func (ctfer *CTFer) check(args *CTFerArgs) error {
	checks := 0
	wg := &sync.WaitGroup{}
	wg.Add(checks)
	cerr := make(chan error, checks)

	// TODO perform validation checks
	// smth.ApplyT(func(abc def) ghi {
	//     defer wg.Done()
	//
	//     ... the actual test
	//     if err != nil {
	//         cerr <- err
	//         return
	//     }
	// })

	wg.Wait()
	close(cerr)

	var merr error
	for err := range cerr {
		merr = multierr.Append(merr, err)
	}
	return merr
}

func (ctfer *CTFer) provision(ctx *pulumi.Context, args *CTFerArgs, opts ...pulumi.ResourceOption) (err error) {
	// Deploy HA MariaDB
	// TODO scale up to >=3
	// FIXME when scaled to 3, ctfd replicas errors
	ctfer.maria, err = parts.NewMariaDB(ctx, "database", &parts.MariaDBArgs{
		Namespace:        args.Namespace,
		ChartsRepository: args.ChartsRepository,
		ChartVersion:     pulumi.String("20.5.3"),
		Registry:         args.ImagesRepository,
		StorageClassName: args.StorageClassName,
	}, opts...)
	if err != nil {
		return
	}

	// Deploy Redis
	// TODO scale up to >=3
	// FIXME when scaled to 3, ctfd replicas errors
	ctfer.redis, err = parts.NewRedis(ctx, "cache", &parts.RedisArgs{
		Namespace:        args.Namespace,
		ChartsRepository: args.ChartsRepository,
		ChartVersion:     pulumi.String("20.13.4"),
		Registry:         args.ImagesRepository,
		StorageClassName: args.StorageClassName,
	}, opts...)
	if err != nil {
		return
	}

	ctfdArgs := &parts.CTFdArgs{
		Namespace:        args.Namespace,
		RedisURL:         ctfer.redis.URL,
		MariaDBURL:       ctfer.maria.URL,
		Image:            args.CTFdImage,
		Registry:         args.ImagesRepository,
		StorageClassName: args.StorageClassName,
		Hostname:         args.Hostname,
		CTFdCrt:          args.CTFdCrt,
		CTFdKey:          args.CTFdKey,
		CTFdStorageSize:  args.CTFdStorageSize,
		CTFdWorkers:      args.CTFdWorkers,
		CTFdReplicas:     args.CTFdReplicas,
		ChallManagerUrl:  args.ChallManagerUrl,
		CTFdRequests:     args.CTFdRequests,
		CTFdLimits:       args.CTFdLimits,
	}
	if args.Otel != nil {
		ctfdArgs.Otel = &parts.OtelArgs{
			ServiceName: pulumi.Sprintf("%s-ctfd", args.Otel.ServiceName),
			Endpoint:    args.Otel.Endpoint,
			Insecure:    args.Otel.Insecure,
		}
	}
	ctfer.ctfd, err = parts.NewCTFd(ctx, "platform", ctfdArgs, append(opts, pulumi.DependsOn([]pulumi.Resource{
		ctfer.maria,
		ctfer.redis,
	}))...)
	if err != nil {
		return
	}

	// Top-level NetworkPolicies
	// - IngressController -> CTFd
	// - CTFd -> Redis
	// - CTFd -> MariaDB
	ctfer.ctfdNetpol, err = netwv1.NewNetworkPolicy(ctx, "ctfd-netpol", &netwv1.NetworkPolicyArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("ctfer"),
				"app.kubernetes.io/part-of":   pulumi.String("ctfer"),
				"ctfer.io/stack-name":         pulumi.String(ctx.Stack()),
			},
		},
		Spec: netwv1.NetworkPolicySpecArgs{
			PolicyTypes: pulumi.ToStringArray([]string{
				"Ingress",
				"Egress",
			}),
			PodSelector: metav1.LabelSelectorArgs{
				MatchLabels: ctfer.ctfd.PodLabels,
			},
			Ingress: netwv1.NetworkPolicyIngressRuleArray{
				// Ingress ->
				netwv1.NetworkPolicyIngressRuleArgs{
					From: netwv1.NetworkPolicyPeerArray{
						netwv1.NetworkPolicyPeerArgs{
							NamespaceSelector: metav1.LabelSelectorArgs{
								MatchLabels: pulumi.StringMap{
									"kubernetes.io/metadata.name": args.IngressNamespace,
								},
							},
							PodSelector: metav1.LabelSelectorArgs{
								MatchLabels: args.IngressLabels,
							},
						},
					},
					Ports: netwv1.NetworkPolicyPortArray{
						netwv1.NetworkPolicyPortArgs{
							Port: pulumi.Int(8000),
						},
					},
				},
			},
			Egress: netwv1.NetworkPolicyEgressRuleArray{
				// -> Redis
				netwv1.NetworkPolicyEgressRuleArgs{
					To: netwv1.NetworkPolicyPeerArray{
						netwv1.NetworkPolicyPeerArgs{
							NamespaceSelector: metav1.LabelSelectorArgs{
								MatchLabels: pulumi.StringMap{
									"kubernetes.io/metadata.name": args.Namespace,
								},
							},
							PodSelector: metav1.LabelSelectorArgs{
								MatchLabels: ctfer.redis.PodLabels,
							},
						},
					},
					Ports: netwv1.NetworkPolicyPortArray{
						netwv1.NetworkPolicyPortArgs{
							Port:     parseURLPort(ctfer.redis.URL),
							Protocol: pulumi.String("TCP"),
						},
					},
				},
				// -> MariaDB
				netwv1.NetworkPolicyEgressRuleArgs{
					To: netwv1.NetworkPolicyPeerArray{
						netwv1.NetworkPolicyPeerArgs{
							NamespaceSelector: metav1.LabelSelectorArgs{
								MatchLabels: pulumi.StringMap{
									"kubernetes.io/metadata.name": args.Namespace,
								},
							},
							PodSelector: metav1.LabelSelectorArgs{
								MatchLabels: ctfer.maria.PodLabels,
							},
						},
					},
					Ports: netwv1.NetworkPolicyPortArray{
						netwv1.NetworkPolicyPortArgs{
							Port:     parseURLPort(ctfer.maria.URL),
							Protocol: pulumi.String("TCP"),
						},
					},
				},
			},
		},
	}, opts...)

	return
}

func (ctfer *CTFer) outputs(ctx *pulumi.Context) error {
	ctfer.URL = ctfer.ctfd.URL
	ctfer.PodLabels = ctfer.ctfd.PodLabels

	return ctx.RegisterResourceOutputs(ctfer, pulumi.Map{
		"url":       ctfer.URL,
		"podLabels": ctfer.PodLabels,
	})
}
