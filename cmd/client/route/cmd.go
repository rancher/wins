package route

import (
	"github.com/urfave/cli"
)

func NewCommand() cli.Command {
	return cli.Command{
		Name:  "route",
		Usage: "Manage Routes",
		Subcommands: []cli.Command{
			addCommand(),
		},
	}
}
