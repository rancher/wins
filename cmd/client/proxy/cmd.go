package proxy

import (
	"fmt"

	"github.com/rancher/wins/pkg/defaults"
	"github.com/urfave/cli/v2"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:   "proxy",
		Usage:  fmt.Sprintf("Set up a proxy for a port via %s (note: only TCP is supported)", defaults.WindowsServiceDisplayName),
		Flags:  _proxyFlags,
		Before: _proxyRequestParser,
		Action: _proxyAction,
	}
}
