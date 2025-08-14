package main

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	netwv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	cm "github.com/ctfer-io/chall-manager/deploy/services"
	cmparts "github.com/ctfer-io/chall-manager/deploy/services/parts"
	monitoring "github.com/ctfer-io/monitoring/services"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/example/simulation/services"
	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/example/simulation/services/parts"
	"github.com/ctfer-io/chall-manager/deploy/common"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg, err := InitConfig(ctx)
		if err != nil {
			return err
		}

		// => Monitoring
		mon, err := monitoring.NewMonitoring(ctx, "monitoring", &monitoring.MonitoringArgs{
			ColdExtract: true, // We want to extract the OpenTelemetry traces for SIG
			Registry:    pulumi.String(cfg.Registry),
		})
		if err != nil {
			return err
		}

		// => Namespace to deploy the platform and services
		ns, err := parts.NewNamespace(ctx, "ctf", &parts.NamespaceArgs{
			Name: pulumi.String("fullchain"),
		})
		if err != nil {
			return err
		}

		// => Scenario registry
		reg, err := parts.NewRegistry(ctx, "registry", &parts.RegistryArgs{
			Registry:  pulumi.String(cfg.Registry),
			Namespace: ns.Name,
			Otel: &common.OtelArgs{
				Endpoint:    mon.OTEL.Endpoint,
				ServiceName: pulumi.String(ctx.Stack()),
				Insecure:    true,
			},
		})
		if err != nil {
			return err
		}

		// => Chall-Manager
		cm, err := cm.NewChallManager(ctx, "chall-manager", &cm.ChallManagerArgs{
			Namespace:    ns.Name,
			LogLevel:     pulumi.String("info"),
			Tag:          pulumi.String("v0.5.1"),
			Registry:     pulumi.String(cfg.Registry),
			EtcdReplicas: pulumi.IntPtr(1),
			OCIInsecure:  true,
			JanitorMode:  cmparts.JanitorModeTicker,
			Otel: &common.OtelArgs{
				Endpoint:    mon.OTEL.Endpoint,
				ServiceName: pulumi.String(ctx.Stack()),
				Insecure:    true,
			},
		})
		if err != nil {
			return err
		}

		// => CTFer/CTFd
		ctfer, err := services.NewCTFer(ctx, "platform", &services.CTFerArgs{
			Namespace:        ns.Name,
			Hostname:         pulumi.String("localhost"),
			CTFdImage:        pulumi.String("ctferio/ctfd:3.7.7-0.5.0"),
			ChallManagerUrl:  pulumi.Sprintf("http://%s", cm.Endpoint),
			ImagesRepository: pulumi.String(cfg.Registry),
			// don't bother with ChartsRepository, it is lightweight enough to avoid copying it
			CTFdStorageSize: pulumi.String("2Gi"),
			Otel: &parts.OtelArgs{
				Endpoint:    pulumi.Sprintf("dns://%s", mon.OTEL.Endpoint), // XXX for now, CTFer does not default the protocol if none set
				ServiceName: pulumi.String(ctx.Stack()),
				Insecure:    true,
			},
		})
		if err != nil {
			return err
		}

		if _, err := netwv1.NewNetworkPolicy(ctx, "ctfd-to-cm", &netwv1.NetworkPolicyArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: ns.Name,
				Labels: pulumi.StringMap{
					"app.kubernetes.io/part-of": pulumi.String("fullchain"),
					"ctfer.io/stack-name":       pulumi.String(ctx.Stack()),
				},
			},
			Spec: netwv1.NetworkPolicySpecArgs{
				PolicyTypes: pulumi.ToStringArray([]string{
					"Egress",
				}),
				PodSelector: metav1.LabelSelectorArgs{
					MatchLabels: ctfer.PodLabels,
				},
				Egress: netwv1.NetworkPolicyEgressRuleArray{
					netwv1.NetworkPolicyEgressRuleArgs{
						To: netwv1.NetworkPolicyPeerArray{
							netwv1.NetworkPolicyPeerArgs{
								NamespaceSelector: metav1.LabelSelectorArgs{
									MatchLabels: pulumi.StringMap{
										"kubernetes.io/metadata.name": ns.Name,
									},
								},
								PodSelector: metav1.LabelSelectorArgs{
									MatchLabels: cm.PodLabels,
								},
							},
						},
						Ports: netwv1.NetworkPolicyPortArray{
							netwv1.NetworkPolicyPortArgs{
								Port:     parsePort(cm.Endpoint),
								Protocol: pulumi.String("TCP"),
							},
						},
					},
				},
			},
		}); err != nil {
			return err
		}

		if _, err := netwv1.NewNetworkPolicy(ctx, "cm-from-ctfd", &netwv1.NetworkPolicyArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: ns.Name,
				Labels: pulumi.StringMap{
					"app.kubernetes.io/part-of": pulumi.String("fullchain"),
					"ctfer.io/stack-name":       pulumi.String(ctx.Stack()),
				},
			},
			Spec: netwv1.NetworkPolicySpecArgs{
				PolicyTypes: pulumi.ToStringArray([]string{
					"Ingress",
				}),
				PodSelector: metav1.LabelSelectorArgs{
					MatchLabels: cm.PodLabels,
				},
				Ingress: netwv1.NetworkPolicyIngressRuleArray{
					netwv1.NetworkPolicyIngressRuleArgs{
						From: netwv1.NetworkPolicyPeerArray{
							netwv1.NetworkPolicyPeerArgs{
								NamespaceSelector: metav1.LabelSelectorArgs{
									MatchLabels: pulumi.StringMap{
										"kubernetes.io/metadata.name": ns.Name,
									},
								},
								PodSelector: metav1.LabelSelectorArgs{
									MatchLabels: ctfer.PodLabels,
								},
							},
						},
						Ports: netwv1.NetworkPolicyPortArray{
							netwv1.NetworkPolicyPortArgs{
								Port:     parsePort(cm.Endpoint),
								Protocol: pulumi.String("TCP"),
							},
						},
					},
				},
			},
		}); err != nil {
			return err
		}

		if _, err := netwv1.NewNetworkPolicy(ctx, "cm-to-registry", &netwv1.NetworkPolicyArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: ns.Name,
				Labels: pulumi.StringMap{
					"app.kubernetes.io/part-of": pulumi.String("fullchain"),
					"ctfer.io/stack-name":       pulumi.String(ctx.Stack()),
				},
			},
			Spec: netwv1.NetworkPolicySpecArgs{
				PolicyTypes: pulumi.ToStringArray([]string{
					"Egress",
				}),
				PodSelector: metav1.LabelSelectorArgs{
					MatchLabels: cm.PodLabels,
				},
				Egress: netwv1.NetworkPolicyEgressRuleArray{
					netwv1.NetworkPolicyEgressRuleArgs{
						To: netwv1.NetworkPolicyPeerArray{
							netwv1.NetworkPolicyPeerArgs{
								NamespaceSelector: metav1.LabelSelectorArgs{
									MatchLabels: pulumi.StringMap{
										"kubernetes.io/metadata.name": ns.Name,
									},
								},
								PodSelector: metav1.LabelSelectorArgs{
									MatchLabels: reg.PodLabels,
								},
							},
						},
						Ports: netwv1.NetworkPolicyPortArray{
							netwv1.NetworkPolicyPortArgs{
								Port:     parsePort(reg.Endpoint),
								Protocol: pulumi.String("TCP"),
							},
						},
					},
				},
			},
		}); err != nil {
			return err
		}

		if _, err := netwv1.NewNetworkPolicy(ctx, "registry-from-cm", &netwv1.NetworkPolicyArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: ns.Name,
				Labels: pulumi.StringMap{
					"app.kubernetes.io/part-of": pulumi.String("fullchain"),
					"ctfer.io/stack-name":       pulumi.String(ctx.Stack()),
				},
			},
			Spec: netwv1.NetworkPolicySpecArgs{
				PolicyTypes: pulumi.ToStringArray([]string{
					"Ingress",
				}),
				PodSelector: metav1.LabelSelectorArgs{
					MatchLabels: reg.PodLabels,
				},
				Ingress: netwv1.NetworkPolicyIngressRuleArray{
					netwv1.NetworkPolicyIngressRuleArgs{
						From: netwv1.NetworkPolicyPeerArray{
							netwv1.NetworkPolicyPeerArgs{
								NamespaceSelector: metav1.LabelSelectorArgs{
									MatchLabels: pulumi.StringMap{
										"kubernetes.io/metadata.name": ns.Name,
									},
								},
								PodSelector: metav1.LabelSelectorArgs{
									MatchLabels: cm.PodLabels,
								},
							},
						},
						Ports: netwv1.NetworkPolicyPortArray{
							netwv1.NetworkPolicyPortArgs{
								Port:     parsePort(reg.Endpoint),
								Protocol: pulumi.String("TCP"),
							},
						},
					},
				},
			},
		}); err != nil {
			return err
		}

		if _, err := netwv1.NewNetworkPolicy(ctx, "ctfd-to-otel", &netwv1.NetworkPolicyArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: ns.Name,
				Labels: pulumi.StringMap{
					"app.kubernetes.io/part-of": pulumi.String("fullchain"),
					"ctfer.io/stack-name":       pulumi.String(ctx.Stack()),
				},
			},
			Spec: netwv1.NetworkPolicySpecArgs{
				PolicyTypes: pulumi.ToStringArray([]string{
					"Egress",
				}),
				PodSelector: metav1.LabelSelectorArgs{
					MatchLabels: ctfer.PodLabels,
				},
				Egress: netwv1.NetworkPolicyEgressRuleArray{
					netwv1.NetworkPolicyEgressRuleArgs{
						To: netwv1.NetworkPolicyPeerArray{
							netwv1.NetworkPolicyPeerArgs{
								NamespaceSelector: metav1.LabelSelectorArgs{
									MatchLabels: pulumi.StringMap{
										"kubernetes.io/metadata.name": mon.Namespace,
									},
								},
								PodSelector: metav1.LabelSelectorArgs{
									MatchLabels: mon.OTEL.PodLabels,
								},
							},
						},
						Ports: netwv1.NetworkPolicyPortArray{
							netwv1.NetworkPolicyPortArgs{
								Port:     parsePort(mon.OTEL.Endpoint),
								Protocol: pulumi.String("TCP"),
							},
						},
					},
				},
			},
		}); err != nil {
			return err
		}

		ctfdSvc, err := corev1.NewService(ctx, "ctfd-exposed", &corev1.ServiceArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: ns.Name,
				Labels: pulumi.StringMap{
					"app.kubernetes.io/part-of": pulumi.String("fullchain"),
					"ctfer.io/stack-name":       pulumi.String(ctx.Stack()),
				},
			},
			Spec: corev1.ServiceSpecArgs{
				Type:     pulumi.String("NodePort"),
				Selector: ctfer.PodLabels,
				Ports: corev1.ServicePortArray{
					corev1.ServicePortArgs{
						Port: pulumi.Int(8000), // CTFd default port
					},
				},
			},
		})
		if err != nil {
			return err
		}

		if _, err := netwv1.NewNetworkPolicy(ctx, "ctfd-exposed-netpol", &netwv1.NetworkPolicyArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: ns.Name,
				Labels: pulumi.StringMap{
					"app.kubernetes.io/components": pulumi.String("fullchain"),
					"app.kubernetes.io/part-of":    pulumi.String("fullchain"),
					"ctfer.io/stack-name":          pulumi.String(ctx.Stack()),
				},
			},
			Spec: netwv1.NetworkPolicySpecArgs{
				PodSelector: metav1.LabelSelectorArgs{
					MatchLabels: ctfer.PodLabels,
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
								Port:     ctfdSvc.Spec.Ports().Index(pulumi.Int(0)).Port(),
								Protocol: pulumi.String("TCP"),
							},
						},
					},
				},
			},
		}); err != nil {
			return err
		}

		otelSvc, err := corev1.NewService(ctx, "otel-exposed", &corev1.ServiceArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: mon.Namespace,
				Labels: pulumi.StringMap{
					"app.kubernetes.io/part-of": pulumi.String("fullchain"),
					"ctfer.io/stack-name":       pulumi.String(ctx.Stack()),
				},
			},
			Spec: corev1.ServiceSpecArgs{
				Type:     pulumi.String("NodePort"),
				Selector: mon.OTEL.PodLabels,
				Ports: corev1.ServicePortArray{
					corev1.ServicePortArgs{
						Port: pulumi.Int(4317), // OTEL Collector gRPC port
					},
				},
			},
		})
		if err != nil {
			return err
		}

		if _, err := netwv1.NewNetworkPolicy(ctx, "otel-exposed-netpol", &netwv1.NetworkPolicyArgs{
			Metadata: metav1.ObjectMetaArgs{
				Namespace: mon.Namespace,
				Labels: pulumi.StringMap{
					"app.kubernetes.io/components": pulumi.String("fullchain"),
					"app.kubernetes.io/part-of":    pulumi.String("fullchain"),
					"ctfer.io/stack-name":          pulumi.String(ctx.Stack()),
				},
			},
			Spec: netwv1.NetworkPolicySpecArgs{
				PodSelector: metav1.LabelSelectorArgs{
					MatchLabels: mon.OTEL.PodLabels,
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
								Port:     otelSvc.Spec.Ports().Index(pulumi.Int(0)).Port(),
								Protocol: pulumi.String("TCP"),
							},
						},
					},
				},
			},
		}); err != nil {
			return err
		}

		ctx.Export("registry.nodeport", reg.NodePort)
		ctx.Export("monitoring.namespace", mon.Namespace)
		ctx.Export("monitoring.otel-cold-extract-pvc-name", mon.OTEL.ColdExtractPVCName)
		ctx.Export("monitoring.nodeport", otelSvc.Spec.Ports().Index(pulumi.Int(0)).NodePort().Elem())
		ctx.Export("url", pulumi.Sprintf("http://localhost:%d", ctfdSvc.Spec.Ports().Index(pulumi.Int(0)).NodePort().Elem()))

		return nil
	})
}

type Config struct {
	Registry string
}

func InitConfig(ctx *pulumi.Context) (*Config, error) {
	cfg := config.New(ctx, "")
	return &Config{
		Registry: cfg.Require("registry"),
	}, nil
}

// parsePort cuts the input endpoint to return its port.
// Example: some.thing:port -> port
func parsePort(edp pulumi.StringInput) pulumi.IntOutput {
	return edp.ToStringOutput().ApplyT(func(edp string) (int, error) {
		_, pStr, _ := strings.Cut(edp, ":")
		p, err := strconv.Atoi(pStr)
		if err != nil {
			return 0, errors.Wrapf(err, "parsing endpoint %s for port", edp)
		}
		return p, nil
	}).(pulumi.IntOutput)
}
