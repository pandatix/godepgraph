package parts

import (
	"strings"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type (
	Neo4J struct {
		pulumi.ResourceState

		rand *random.RandomString
		dep  *appsv1.Deployment
		svc  *corev1.Service

		PodLabels pulumi.StringMapOutput
		Endpoint  pulumi.StringOutput
		Username  pulumi.StringOutput
		Password  pulumi.StringOutput
		DBName    pulumi.StringOutput
	}

	Neo4JArgs struct {
		// Namespace to which deploy the Neo4J resources.
		Namespace pulumi.StringInput

		// Registry define from where to fetch the Neo4J Docker images.
		// If set empty, defaults to Docker Hub.
		// Authentication is not supported, please provide it as Kubernetes-level configuration.
		Registry pulumi.StringPtrInput
		registry pulumi.StringOutput
	}
)

// NewNeo4J is a Kubernetes resources builder for a Neo4J instance.
func NewNeo4J(ctx *pulumi.Context, name string, args *Neo4JArgs, opts ...pulumi.ResourceOption) (*Neo4J, error) {
	n4j := &Neo4J{}

	args = n4j.defaults(args)
	if err := ctx.RegisterComponentResource("pandatix:godepgraph:neo4j", name, n4j, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(n4j))
	if err := n4j.provision(ctx, args, opts...); err != nil {
		return nil, err
	}
	if err := n4j.outputs(ctx); err != nil {
		return nil, err
	}
	return n4j, nil
}

func (n4j *Neo4J) defaults(args *Neo4JArgs) *Neo4JArgs {
	if args == nil {
		args = &Neo4JArgs{}
	}

	args.registry = pulumi.String("").ToStringOutput()
	if args.Registry != nil {
		args.registry = args.Registry.ToStringPtrOutput().ApplyT(func(in *string) string {
			// No private registry -> defaults to Docker Hub
			if in == nil {
				return ""
			}

			str := *in
			// If one set, make sure it ends with one '/'
			if str != "" && !strings.HasSuffix(str, "/") {
				str = str + "/"
			}
			return str
		}).(pulumi.StringOutput)
	}

	return args
}

func (n4j *Neo4J) provision(ctx *pulumi.Context, args *Neo4JArgs, opts ...pulumi.ResourceOption) (err error) {
	n4j.rand, err = random.NewRandomString(ctx, "neo4j-password", &random.RandomStringArgs{
		Length:  pulumi.Int(16),
		Special: pulumi.Bool(false),
	}, opts...)
	if err != nil {
		return err
	}

	// => Deployment
	n4j.dep, err = appsv1.NewDeployment(ctx, "neo4j-deployment", &appsv1.DeploymentArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/name":      pulumi.String("neo4j"),
				"app.kubernetes.io/component": pulumi.String("neo4j"),
				"app.kubernetes.io/part-of":   pulumi.String("godepgraph"),
				"pandatix/stack-name":         pulumi.String(ctx.Stack()),
			},
		},
		Spec: appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app.kubernetes.io/name":      pulumi.String("neo4j"),
					"app.kubernetes.io/component": pulumi.String("neo4j"),
					"app.kubernetes.io/part-of":   pulumi.String("godepgraph"),
					"pandatix/stack-name":         pulumi.String(ctx.Stack()),
				},
			},
			Template: corev1.PodTemplateSpecArgs{
				Metadata: metav1.ObjectMetaArgs{
					Namespace: args.Namespace,
					Labels: pulumi.StringMap{
						"app.kubernetes.io/name":      pulumi.String("neo4j"),
						"app.kubernetes.io/component": pulumi.String("neo4j"),
						"app.kubernetes.io/part-of":   pulumi.String("godepgraph"),
						"pandatix/stack-name":         pulumi.String(ctx.Stack()),
					},
				},
				Spec: corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Name:  pulumi.String("neo4j"),
							Image: pulumi.Sprintf("%slibrary/neo4j:5.22.0", args.registry),
							Env: corev1.EnvVarArray{
								corev1.EnvVarArgs{
									Name:  pulumi.String("NEO4J_AUTH"),
									Value: pulumi.Sprintf("neo4j/%s", n4j.rand.Result),
								},
								// Following comes from https://stackoverflow.com/a/77261518
								corev1.EnvVarArgs{
									Name:  pulumi.String("NEO4J_server_config_strict__validation_enabled"),
									Value: pulumi.String("false"),
								},
							},
							Ports: corev1.ContainerPortArray{
								corev1.ContainerPortArgs{
									Name:          pulumi.String("ui"),
									ContainerPort: pulumi.Int(7474),
								},
								corev1.ContainerPortArgs{
									Name:          pulumi.String("api"),
									ContainerPort: pulumi.Int(7687),
								},
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

	// => Service
	n4j.svc, err = corev1.NewService(ctx, "neo4j-service", &corev1.ServiceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("neo4j"),
				"app.kubernetes.io/part-of":   pulumi.String("godepgraph"),
				"pandatix/stack-name":         pulumi.String(ctx.Stack()),
			},
		},
		Spec: corev1.ServiceSpecArgs{
			ClusterIP: pulumi.String("None"), // Headless, for DNS purposes
			Ports: corev1.ServicePortArray{
				corev1.ServicePortArgs{
					Name: pulumi.String("ui"),
					Port: pulumi.Int(7474),
				},
				corev1.ServicePortArgs{
					Name: pulumi.String("api"),
					Port: pulumi.Int(7687),
				},
			},
			Selector: n4j.dep.Spec.Template().Metadata().Labels(),
		},
	}, opts...)
	if err != nil {
		return
	}

	return
}

func (n4j *Neo4J) outputs(ctx *pulumi.Context) error {
	n4j.PodLabels = n4j.dep.Spec.Template().Metadata().Labels()
	n4j.Username = pulumi.String("neo4j").ToStringOutput() // hardcoded value, is not important for PoC
	n4j.Password = n4j.rand.Result
	n4j.Endpoint = pulumi.Sprintf(
		"bolt://%s.%s:7687",
		n4j.svc.Metadata.Name().Elem(),
		n4j.svc.Metadata.Namespace().Elem(),
	)
	n4j.DBName = pulumi.String("neo4j").ToStringOutput() // hardcoded value, is not important for PoC

	return ctx.RegisterResourceOutputs(n4j, pulumi.Map{
		"podLabels": n4j.PodLabels,
		"endpoint":  n4j.Endpoint,
		"username":  n4j.Username,
		"password":  n4j.Password,
		"dbname":    n4j.DBName,
	})
}
