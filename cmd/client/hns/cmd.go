package hns

import (
	"github.com/urfave/cli/v2"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:  "hns",
		Usage: "Manage Host Networking Service",
		Subcommands: []*cli.Command{
			getNetworkCommand(),
		},
	}
}
