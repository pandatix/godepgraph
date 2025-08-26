package parts

import (
	"strings"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type (
	GoDepGraph struct {
		pulumi.ResourceState

		dep *appsv1.Deployment
		svc *corev1.Service

		PodLabels pulumi.StringMapOutput
		Endpoint  pulumi.StringOutput
	}

	GoDepGraphArgs struct {
		// Tag defines the specific tag to run GoDepGraph to.
		// If not specified, defaults to "latest".
		Tag pulumi.StringPtrInput
		tag pulumi.StringOutput

		// Registry define from where to fetch the GoDepGraph Docker images.
		// If set empty, defaults to Docker Hub.
		// Authentication is not supported, please provide it as Kubernetes-level configuration.
		Registry pulumi.StringPtrInput
		registry pulumi.StringOutput

		// LogLevel defines the level at which to log.
		LogLevel pulumi.StringInput
		logLevel pulumi.StringOutput

		// Namespace to which deploy the GoDepGraph resources.
		Namespace pulumi.StringInput

		// Replicas of the GoDepGraph instance. If not specified, default to 1.
		Replicas pulumi.IntPtrInput
		replicas pulumi.IntOutput

		// Requests for the GoDepGraph container. For more infos:
		// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
		Requests pulumi.StringMapInput
		requests pulumi.StringMapOutput

		// Limits for the GoDepGraph container. For more infos:
		// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
		Limits pulumi.StringMapInput
		limits pulumi.StringMapOutput

		Swagger bool

		Neo4J GoDepGraphNeo4JArgs
	}

	GoDepGraphNeo4JArgs struct {
		URI      pulumi.StringInput
		Username pulumi.StringInput
		Password pulumi.StringInput
		DBName   pulumi.StringInput
	}
)

const (
	port    = 8080
	portKey = "grpc"

	defaultTag      = "latest"
	defaultLogLevel = "info"
)

// NewGoDepGraph is a Kubernetes resources builder for a GoDepGraph instance.
func NewGoDepGraph(ctx *pulumi.Context, name string, args *GoDepGraphArgs, opts ...pulumi.ResourceOption) (*GoDepGraph, error) {
	gdg := &GoDepGraph{}

	args = gdg.defaults(args)
	if err := ctx.RegisterComponentResource("pandatix:godepgraph:godepgraph", name, gdg, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(gdg))
	if err := gdg.provision(ctx, args, opts...); err != nil {
		return nil, err
	}
	if err := gdg.outputs(ctx); err != nil {
		return nil, err
	}
	return gdg, nil
}

func (gdg *GoDepGraph) defaults(args *GoDepGraphArgs) *GoDepGraphArgs {
	if args == nil {
		args = &GoDepGraphArgs{}
	}

	args.tag = pulumi.String(defaultTag).ToStringOutput()
	if args.Tag != nil {
		args.tag = args.Tag.ToStringPtrOutput().ApplyT(func(tag *string) string {
			if tag == nil || *tag == "" {
				return defaultTag
			}
			return *tag
		}).(pulumi.StringOutput)
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

	args.logLevel = pulumi.String(defaultLogLevel).ToStringOutput()
	if args.LogLevel != nil {
		args.logLevel = args.LogLevel.ToStringOutput()
	}

	args.replicas = pulumi.Int(1).ToIntOutput()
	if args.Replicas != nil {
		args.replicas = args.Replicas.ToIntPtrOutput().ApplyT(func(replicas *int) int {
			if replicas == nil || *replicas < 1 {
				return 1
			}
			return *replicas
		}).(pulumi.IntOutput)
	}

	args.requests = pulumi.StringMap{}.ToStringMapOutput()
	if args.Requests != nil {
		args.requests = args.Requests.ToStringMapOutput()
	}

	args.limits = pulumi.StringMap{}.ToStringMapOutput()
	if args.Limits != nil {
		args.limits = args.Limits.ToStringMapOutput()
	}

	return args
}

func (gdg *GoDepGraph) provision(ctx *pulumi.Context, args *GoDepGraphArgs, opts ...pulumi.ResourceOption) (err error) {
	// TODO create secret for Neo4J accesses

	// => Deployment
	envs := corev1.EnvVarArray{
		corev1.EnvVarArgs{
			Name:  pulumi.String("PORT"),
			Value: pulumi.Sprintf("%d", port),
		},
		corev1.EnvVarArgs{
			Name:  pulumi.String("SWAGGER"),
			Value: pulumi.Sprintf("%t", args.Swagger),
		},
		corev1.EnvVarArgs{
			Name:  pulumi.String("LOG_LEVEL"),
			Value: args.logLevel,
		},
		corev1.EnvVarArgs{
			Name:  pulumi.String("NEO4J_URI"),
			Value: args.Neo4J.URI,
		},
		corev1.EnvVarArgs{
			Name:  pulumi.String("NEO4J_USER"),
			Value: args.Neo4J.Username,
		},
		corev1.EnvVarArgs{
			Name:  pulumi.String("NEO4J_PASS"),
			Value: args.Neo4J.Password,
		},
		corev1.EnvVarArgs{
			Name:  pulumi.String("NEO4J_DBNAME"),
			Value: args.Neo4J.DBName,
		},
	}

	gdg.PodLabels = pulumi.StringMap{
		"app.kubernetes.io/name":      pulumi.String("godepgraph"),
		"app.kubernetes.io/version":   args.tag,
		"app.kubernetes.io/component": pulumi.String("godepgraph"),
		"app.kubernetes.io/part-of":   pulumi.String("godepgraph"),
		"pandatix/stack-name":         pulumi.String(ctx.Stack()),
	}.ToStringMapOutput()
	gdg.dep, err = appsv1.NewDeployment(ctx, "godepgraph-deployment", &appsv1.DeploymentArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels:    gdg.PodLabels,
		},
		Spec: appsv1.DeploymentSpecArgs{
			Replicas: args.replicas,
			Selector: metav1.LabelSelectorArgs{
				MatchLabels: gdg.PodLabels,
			},
			Template: corev1.PodTemplateSpecArgs{
				Metadata: metav1.ObjectMetaArgs{
					Namespace: args.Namespace,
					Labels:    gdg.PodLabels,
				},
				Spec: corev1.PodSpecArgs{
					InitContainers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Name:  pulumi.String("wait-for-neo4j"),
							Image: pulumi.Sprintf("%slibrary/neo4j:5.22.0", args.registry),
							Command: pulumi.ToStringArray([]string{
								"sh", "-c", `
echo "Waiting for Neo4j Cypher to respond..."
until cypher-shell -u "$NEO4J_USER" -p "$NEO4J_PASS" -d "$NEO4J_DBNAME" "RETURN 1" >/dev/null 2>&1; do
	echo "Still waiting..."
	sleep 5
done
echo "Neo4j is ready!"
`,
							}),
							Env: corev1.EnvVarArray{
								corev1.EnvVarArgs{
									Name:  pulumi.String("NEO4J_URI"),
									Value: args.Neo4J.URI,
								},
								corev1.EnvVarArgs{
									Name:  pulumi.String("NEO4J_USER"),
									Value: args.Neo4J.Username,
								},
								corev1.EnvVarArgs{
									Name:  pulumi.String("NEO4J_PASS"),
									Value: args.Neo4J.Password,
								},
								corev1.EnvVarArgs{
									Name:  pulumi.String("NEO4J_DBNAME"),
									Value: pulumi.String("neo4j"),
								},
							},
						},
					},
					Containers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Name:            pulumi.String("godepgraph"),
							Image:           pulumi.Sprintf("%spandatix/godepgraph:%s", args.registry, args.tag),
							Env:             envs,
							ImagePullPolicy: pulumi.String("Always"),
							Ports: corev1.ContainerPortArray{
								corev1.ContainerPortArgs{
									Name:          pulumi.String(portKey),
									ContainerPort: pulumi.Int(port),
								},
							},
							ReadinessProbe: corev1.ProbeArgs{
								HttpGet: corev1.HTTPGetActionArgs{
									Path: pulumi.String("/healthcheck"),
									Port: pulumi.Int(port),
								},
							},
							Resources: corev1.ResourceRequirementsArgs{
								Requests: args.requests,
								Limits:   args.limits,
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
	gdg.svc, err = corev1.NewService(ctx, "godepgraph-service", &corev1.ServiceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("godepgraph"),
				"app.kubernetes.io/part-of":   pulumi.String("godepgraph"),
				"pandatix/stack-name":         pulumi.String(ctx.Stack()),
			},
		},
		Spec: corev1.ServiceSpecArgs{
			ClusterIP: pulumi.String("None"), // Headless, for DNS purposes
			Ports: corev1.ServicePortArray{
				corev1.ServicePortArgs{
					Name: pulumi.String(portKey),
					Port: pulumi.Int(port),
				},
			},
			Selector: gdg.PodLabels,
		},
	}, opts...)
	if err != nil {
		return
	}

	return
}

func (gdg *GoDepGraph) outputs(ctx *pulumi.Context) error {
	// gdg.PodLabels is defined during provisionning such that it can be returned for
	// netpols. Then, they can be created to grant network traffic (gdg->neo4j)
	// necessary for the readiness probe to pass.

	gdg.Endpoint = pulumi.Sprintf(
		"%s.%s:%d",
		gdg.svc.Metadata.Name().Elem(),
		gdg.svc.Metadata.Namespace().Elem(),
		gdg.svc.Spec.Ports().Index(pulumi.Int(0)).Port(),
	)

	return ctx.RegisterResourceOutputs(gdg, pulumi.Map{
		"podLabels": gdg.PodLabels,
		"endpoint":  gdg.Endpoint,
	})
}
