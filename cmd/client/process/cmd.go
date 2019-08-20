package process

import (
	"github.com/urfave/cli"
)

func NewCommand() cli.Command {
	return cli.Command{
		Name:    "prc",
		Aliases: []string{"process"},
		Usage:   "Manage Processes",
		Subcommands: []cli.Command{
			runCommand(),
		},
	}
}
