package hns

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/rancher/wins/cmd/client/internal"
	"github.com/rancher/wins/cmd/outputs"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/types"
	"github.com/urfave/cli/v2"
)

var _getNetworkFlags = internal.NewGRPCClientConn(
	[]cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Usage: "[optional] Specifies the HNS network name",
		},
		&cli.StringFlag{
			Name:  "address",
			Usage: "[optional] Specifies the HNS network subnet address CIDR",
		},
	},
)

var _getNetworkRequest *types.HnsGetNetworkRequest

func _getNetworkRequestParser(cliCtx *cli.Context) error {
	// validate
	var (
		name    = cliCtx.String("name")
		address = cliCtx.String("address")
	)
	if name == "" && address == "" {
		return errors.New("specifies --name or --address")
	}
	if name != "" && address != "" {
		return errors.New("--name and --address could not use together")
	}

	// parse
	_getNetworkRequest = &types.HnsGetNetworkRequest{}
	if name != "" {
		_getNetworkRequest.Options = &types.HnsGetNetworkRequest_Name{
			Name: name,
		}
	} else if address != "" {
		if !strings.Contains(address, "/") {
			return errors.New("--address should be a CIDR format")
		}
		_getNetworkRequest.Options = &types.HnsGetNetworkRequest_Address{
			Address: address,
		}
	}

	return nil
}

func _getNetworkAction(cliCtx *cli.Context) (err error) {
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
	client := types.NewHnsServiceClient(grpcClientConn)

	resp, err := client.GetNetwork(ctx, _getNetworkRequest)
	if err != nil {
		return
	}

	return outputs.JSON(cliCtx.App.Writer, resp.Data)
}

func getNetworkCommand() *cli.Command {
	return &cli.Command{
		Name:   "get-network",
		Usage:  "Get HNS network metadata",
		Flags:  _getNetworkFlags,
		Before: _getNetworkRequestParser,
		Action: _getNetworkAction,
	}
}
