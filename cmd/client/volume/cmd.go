package volume

import (
	"github.com/urfave/cli"
)

func NewCommand() cli.Command {
	return cli.Command{
		Name:    "vol",
		Aliases: []string{"volume"},
		Usage:   "Manage Volume",
		Subcommands: []cli.Command{
			mountCommand(),
		},
	}
}
