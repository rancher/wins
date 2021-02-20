package upgrade

import (
	"fmt"

	"github.com/rancher/wins/pkg/defaults"
	"github.com/urfave/cli"
)

func NewCommand() cli.Command {
	return cli.Command{
		Name:    "up",
		Aliases: []string{"upgrade"},
		Usage:   fmt.Sprintf("Manage %s Application", defaults.WindowsServiceDisplayName),
		Flags:   _upgradeFlags,
		Before:  _upgradeRequestParser,
		Action:  _upgradeAction,
	}
}
