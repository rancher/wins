package app

import (
	"fmt"

	"github.com/rancher/wins/pkg/defaults"
	"github.com/urfave/cli/v2"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:    "app",
		Aliases: []string{"application"},
		Usage:   fmt.Sprintf("Manage %s Application", defaults.WindowsServiceDisplayName),
		Subcommands: []*cli.Command{
			runCommand(),
		},
	}
}
