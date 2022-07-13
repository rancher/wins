package host

import (
	"context"

	"github.com/rancher/wins/cmd/client/internal"
	"github.com/rancher/wins/cmd/outputs"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/types"
	"github.com/urfave/cli/v2"
)

var _getVersionFlags = internal.NewGRPCClientConn([]cli.Flag{})

func _getVersionAction(cliCtx *cli.Context) (err error) {
	defer panics.Log()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// parse grpc client connection
	grpcClientConn, err := internal.ParseGRPCClientConn(cliCtx)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := grpcClientConn.Close()
		if err == nil {
			err = closeErr
		}
	}()

	// start client
	client := types.NewHostServiceClient(grpcClientConn)

	resp, err := client.GetVersion(ctx, &types.Void{})
	if err != nil {
		return
	}

	return outputs.JSON(cliCtx.App.Writer, resp.Data)
}

func getVersionCommand() *cli.Command {
	return &cli.Command{
		Name:   "get-version",
		Usage:  "Get host version",
		Flags:  _getVersionFlags,
		Action: _getVersionAction,
	}
}
