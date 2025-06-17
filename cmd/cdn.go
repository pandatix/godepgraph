package cmd

import (
	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/cdn"
	"github.com/urfave/cli/v2"
)

var CDNCmd = &cli.Command{
	Name:  "cdn",
	Usage: "Parse the CDN of a Go module",
	Flags: []cli.Flag{
		cli.HelpFlag,
		&cli.StringFlag{
			Name:     "module",
			Usage:    "The Go module name.",
			Required: true,
			Category: "in",
		},
		&cli.StringFlag{
			Name:     "version",
			Usage:    "The version to analyze (e.g. git tag).",
			Required: true,
			Category: "in",
		},
		&cli.BoolFlag{
			Name:     "tests",
			Usage:    "If turned on, also parse test files.",
			Category: "in",
		},
		&cli.StringFlag{
			Name:     "neo4j",
			Usage:    "The Neo4J URL to export CDN into.",
			Category: "out",
			Value:    "neo4j://localhost:7687",
		},
	},
	Action: func(ctx *cli.Context) error {
		a := cdn.NewAnalyser(&cdn.AnalyserParams{
			Tests: ctx.Bool("tests"),
		})
		ana, mod, err := a.Process(ctx.Context, ctx.String("module"), ctx.String("version"))
		if err != nil {
			return err
		}

		con, err := cdn.NewNeo4JConnector(ctx.Context, ctx.String("neo4j"))
		if err != nil {
			return err
		}
		return ana.Export(ctx.Context, con, mod)
	},
}

// Example queries:
// - Get all modules, their packages and their functions
/*
   MATCH (m:Module)
   OPTIONAL MATCH (p:Package) WHERE (m)--(p)
   OPTIONAL MATCH (f:Function) WHERE (p)--(f)
   RETURN m, p, f
*/
