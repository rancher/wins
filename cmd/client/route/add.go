package route

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/rancher/wins/cmd/client/internal"
	"github.com/rancher/wins/cmd/cmds/flags"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/types"
	"github.com/urfave/cli"
)

var _addFlags = internal.NewGRPCClientConn(
	[]cli.Flag{
		cli.GenericFlag{
			Name:  "addresses",
			Usage: "[required] [list-argument] Specifies the addresses or CIDRs as the destinations, e.g.: 8.8.8.8 6.6.6.6/32",
			Value: flags.NewListValue(),
		},
	},
)

var _addRequest *types.RouteAddRequest

func _addRequestParser(cliCtx *cli.Context) error {
	// validate
	addressList := flags.GetListValue(cliCtx, "addresses")
	if addressList.IsEmpty() {
		return errors.Errorf("--addresses is required")
	}

	// parse
	_addRequest = &types.RouteAddRequest{}
	_addRequest.Addresses = addressList.Get()
	for idx, address := range _addRequest.Addresses {
		if !strings.Contains(address, "/") {
			_addRequest.Addresses[idx] = fmt.Sprintf("%s/32", address)
		}
	}

	return nil
}

func _addAction(cliCtx *cli.Context) (err error) {
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
	client := types.NewRouteServiceClient(grpcClientConn)

	_, err = client.Add(ctx, _addRequest)

	return
}

func addCommand() cli.Command {
	return cli.Command{
		Name:   "add",
		Usage:  "Add a route",
		Flags:  _addFlags,
		Before: _addRequestParser,
		Action: _addAction,
	}
}
