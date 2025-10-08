package network

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rancher/wins/cmd/client/internal"
	"github.com/rancher/wins/cmd/outputs"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/types"
	"github.com/urfave/cli/v3"
)

var _getFlags = internal.NewGRPCClientConn(
	[]cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Usage: "[optional] Specifies the network name",
		},
		&cli.StringFlag{
			Name:  "address",
			Usage: "[optional] Specifies the network address",
		},
	},
)

var _getRequest *types.NetworkGetRequest

func _getRequestParser(cliCtx *cli.Context) error {
	// validate
	var (
		name    = cliCtx.String("name")
		address = cliCtx.String("address")
	)
	if name != "" && address != "" {
		return errors.New("--name and --address could not use together")
	}

	// parse
	_getRequest = &types.NetworkGetRequest{}
	if name != "" {
		_getRequest.Options = &types.NetworkGetRequest_Name{
			Name: name,
		}
	} else if address != "" {
		_getRequest.Options = &types.NetworkGetRequest_Address{
			Address: address,
		}
	}

	return nil
}

func _getAction(cliCtx *cli.Context) (err error) {
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
	client := types.NewNetworkServiceClient(grpcClientConn)

	resp, err := client.Get(ctx, _getRequest)
	if err != nil {
		return
	}

	return outputs.JSON(cliCtx.App.Writer, resp.Data)
}

func getCommand() *cli.Command {
	return &cli.Command{
		Name:   "get",
		Usage:  "Get network metadata",
		Flags:  _getFlags,
		Before: _getRequestParser,
		Action: _getAction,
	}
}
