package host

import (
	"github.com/urfave/cli/v3"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:    "hst",
		Aliases: []string{"host"},
		Usage:   "Manage Host",
		Subcommands: []*cli.Command{
			getVersionCommand(),
		},
	}
}
