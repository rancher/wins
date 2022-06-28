package network

import (
	"github.com/urfave/cli/v2"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:    "net",
		Aliases: []string{"network"},
		Usage:   "Manage Network Adapter",
		Subcommands: []*cli.Command{
			getCommand(),
		},
	}
}
