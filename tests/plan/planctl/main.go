// Command planctl drives the wins-plan-e2e Kubernetes plan Secret for the
// Windows plan-execution e2e suite.

package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

const (
	namespace  = "wins-plan-e2e"
	secretName = "wins-plan-e2e-machine-plan"
	saName     = "wins-plan-e2e"
	roleName   = "wins-plan-e2e"
)

func main() {
	app := &cli.App{
		Name:  "planctl",
		Usage: "drive the wins-plan-e2e plan Secret for the Windows plan e2e suite",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "kubeconfig",
				Usage:    "path to an admin kubeconfig",
				Required: true,
			},
		},
		Commands: []*cli.Command{
			bootstrapCommand(),
			applyPlanCommand(),
			annotateCommand(),
			outcomeCommand(),
			resetCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
