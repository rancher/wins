package route

import (
	"github.com/urfave/cli/v3"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:  "route",
		Usage: "Manage Routes",
		Subcommands: []*cli.Command{
			addCommand(),
		},
	}
}
