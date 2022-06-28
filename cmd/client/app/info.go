package app

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/rancher/wins/cmd/client/internal"
	"github.com/rancher/wins/cmd/outputs"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/paths"
	"github.com/rancher/wins/pkg/types"
	"github.com/urfave/cli/v2"
)

var _infoFlags = internal.NewGRPCClientConn()

func _infoAction(cliCtx *cli.Context) (err error) {
	defer panics.Log()

	clientChecksum, err := paths.GetBinarySHA1Hash(os.Args[0])
	if err != nil {
		return errors.Wrap(err, "failed to get checksum for execution binary")
	}

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
	client := types.NewApplicationServiceClient(grpcClientConn)
	infoResp, err := client.Info(ctx, &types.Void{})
	if err != nil {
		return err
	}

	return outputs.JSON(cliCtx.App.Writer, map[string]interface{}{
		"Client": &types.ApplicationInfo{Checksum: clientChecksum, Version: defaults.AppVersion, Commit: defaults.AppCommit},
		"Server": infoResp.Info,
	})
}

func infoCommand() *cli.Command {
	return &cli.Command{
		Name:   "info",
		Usage:  "Get application info",
		Flags:  _infoFlags,
		Action: _infoAction,
	}
}
