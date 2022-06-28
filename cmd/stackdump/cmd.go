package stackdump

import (
	"github.com/urfave/cli/v2"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:   "stackdump",
		Hidden: true,
		Action: _stackDumpAction,
	}
}
