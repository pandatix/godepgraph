package main

import (
	"log"
	"os"

	"github.com/pandatix/godepgraph/cmd"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "godepgraph",
		Usage: "Build dependency graphs out of Systems of Systems.",
		Commands: []*cli.Command{
			cmd.CDNCmd,
			cmd.RDGCmd,
		},
		Flags: []cli.Flag{
			cli.VersionFlag,
			cli.HelpFlag,
		},
		Authors: []*cli.Author{
			{
				Name:  "Lucas Tesson - PandatiX",
				Email: "lucastesson@protonmail.com",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
