package cmd

import (
	"fmt"
	"os"

	"git.cvewatcher.la-ruche.fr/CVEWatcher/godepgraph/rdg"
	"github.com/urfave/cli/v2"
)

var RDGCmd = &cli.Command{
	Name:  "rdg",
	Usage: "Parse the RDG of a Pulumi stack",
	Flags: []cli.Flag{
		cli.HelpFlag,
		&cli.StringFlag{
			Name:     "file",
			Usage:    "The file to parse.",
			Required: true,
			Category: "in",
		},
	},
	Action: func(ctx *cli.Context) error {
		f, err := os.Open(ctx.String("file"))
		if err != nil {
			return err
		}
		defer func() {
			_ = f.Close()
		}()

		a := rdg.NewAnalyser(nil)
		ana, err := a.Process(ctx.Context, f)
		if err != nil {
			return err
		}

		fmt.Printf("ana: %v\n", ana)
		return nil
	},
}
