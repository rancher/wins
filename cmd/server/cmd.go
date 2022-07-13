package server

import (
	"fmt"

	"github.com/rancher/wins/cmd/server/app"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/urfave/cli/v2"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "srv",
		Aliases:     []string{"server"},
		Description: fmt.Sprintf("The server side commands of %s", defaults.WindowsServiceDisplayName),
		Subcommands: []*cli.Command{
			app.NewCommand(),
		},
	}
}
