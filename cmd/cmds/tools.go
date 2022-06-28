package cmds

import (
	"github.com/urfave/cli/v2"
)

func BoolAddr(b bool) *bool {
	boolVar := b
	return &boolVar
}

func JoinFlags(flagSlices []cli.Flag) []cli.Flag {
	var ret []cli.Flag
	for _, flags := range flagSlices {
		ret = append(ret, flags)
	}
	return ret
}

func ChainFuncs(funcs ...func(*cli.Context) error) func(*cli.Context) error {
	if len(funcs) == 0 {
		return nil
	}

	return func(cliCtx *cli.Context) error {
		for _, fn := range funcs {
			if fn == nil {
				continue
			}

			err := fn(cliCtx)
			if err != nil {
				return err
			}
		}

		return nil
	}
}
