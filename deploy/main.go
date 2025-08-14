package main

import (
	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/deploy/services"
	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/deploy/services/parts"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg, err := loadConfig(ctx)
		if err != nil {
			return errors.Wrap(err, "loading configuration")
		}

		// Create namespace
		ns, err := parts.NewNamespace(ctx, "namespace", &parts.NamespaceArgs{
			Name: pulumi.String(cfg.Name),
		})
		if err != nil {
			return err
		}

		// Deploy GoDepGraph
		gdg, err := services.NewGoDepGraph(ctx, "godepgraph", &services.GoDepGraphArgs{
			Namespace: ns.Name,
			Registry:  pulumi.String(cfg.Registry),
			GoDepGraphArgs: services.GoDepGraphGoDepGraphArgs{
				Tag:      pulumi.String(cfg.Tag),
				LogLevel: pulumi.String(cfg.LogLevel),
				Replicas: pulumi.Int(cfg.Replicas),
				Requests: pulumi.ToStringMap(cfg.Requests),
				Limits:   pulumi.ToStringMap(cfg.Limits),
				Swagger:  cfg.Swagger,
			},
			ExposeGoDepGraph: cfg.ExposeGoDepGraph,
			ExposeNeo4J:      cfg.ExposeNeo4J,
		})
		if err != nil {
			return err
		}

		// Namespace
		ctx.Export("namespace", ns.Name)

		// GoDepGraph
		ctx.Export("godepgraph-endpoint", gdg.Endpoint)
		ctx.Export("godepgraph-port", gdg.GoDepGraphPort)
		ctx.Export("neo4j-ui-port", gdg.Neo4JUIPort)
		ctx.Export("neo4j-api-port", gdg.Neo4JAPIPort)
		ctx.Export("neo4j-user", gdg.Neo4JUser)
		ctx.Export("neo4j-pass", gdg.Neo4JPass)
		ctx.Export("neo4j-dbname", gdg.Neo4JDBName)

		return nil
	})
}

type Config struct {
	Name     string
	Registry string
	Tag      string
	LogLevel string
	Replicas int
	Requests map[string]string
	Limits   map[string]string
	Swagger  bool

	ColdExtract bool

	ExposeGoDepGraph bool
	ExposeNeo4J      bool
}

func loadConfig(ctx *pulumi.Context) (*Config, error) {
	cfg := config.New(ctx, "")
	c := &Config{
		Name:             cfg.Get("name"),
		Registry:         cfg.Get("registry"),
		Tag:              cfg.Get("tag"),
		LogLevel:         cfg.Get("log-level"),
		Replicas:         cfg.GetInt("replicas"),
		Swagger:          cfg.GetBool("swagger"),
		ColdExtract:      cfg.GetBool("cold-extract"),
		ExposeGoDepGraph: cfg.GetBool("expose-godepgraph"),
		ExposeNeo4J:      cfg.GetBool("expose-neo4j"),
	}

	if err := cfg.TryObject("requests", &c.Requests); err != nil {
		return nil, err
	}
	if err := cfg.TryObject("limits", &c.Limits); err != nil {
		return nil, err
	}

	return c, nil
}
