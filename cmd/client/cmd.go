package client

import (
	"fmt"

	"github.com/rancher/wins/cmd/client/app"
	"github.com/rancher/wins/cmd/client/hns"
	"github.com/rancher/wins/cmd/client/host"
	"github.com/rancher/wins/cmd/client/network"
	"github.com/rancher/wins/cmd/client/process"
	"github.com/rancher/wins/cmd/client/proxy"
	"github.com/rancher/wins/cmd/client/route"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/urfave/cli/v2"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "cli",
		Aliases:     []string{"client"},
		Description: fmt.Sprintf("The client side commands of %s", defaults.WindowsServiceDisplayName),
		Subcommands: []*cli.Command{
			hns.NewCommand(),
			host.NewCommand(),
			network.NewCommand(),
			process.NewCommand(),
			route.NewCommand(),
			app.NewCommand(),
			proxy.NewCommand(),
		},
	}
}
