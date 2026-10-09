package stackdump

import (
	"github.com/rancher/wins/pkg/profilings"
	"github.com/urfave/cli/v3"
)

func _stackDumpAction(_ *cli.Context) (err error) {
	err = profilings.StackDump()
	if err != nil {
		return err
	}
	return nil
}
