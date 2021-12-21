package stackdump

import (
	"github.com/urfave/cli"
)

func NewCommand() cli.Command {
	return cli.Command{
		Name:   "stackdump",
		Hidden: true,
		Action: _stackDumpAction,
	}
}
