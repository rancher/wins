package internal

import (
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/wins/cmd/cmds"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/rancher/wins/pkg/npipes"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

func NewGRPCClientConn(prependFlags ...[]cli.Flag) []cli.Flag {
	prependFlags = append(prependFlags,
		[]cli.Flag{
			cli.StringFlag{
				Name:  "server",
				Usage: "[optional] Specifies the name of the server listening named pipe",
				Value: defaults.NamedPipeName,
			},
		},
	)
	return cmds.JoinFlags(prependFlags...)
}

func ParseGRPCClientConn(cliCtx *cli.Context) (*grpc.ClientConn, error) {
	dialOptions := []grpc.DialOption{
		grpc.WithInsecure(),
	}

	// setup dialer
	server := cliCtx.String("server")
	serverPath := npipes.GetFullPath(server)
	npipeDialer, err := npipes.NewDialer(serverPath, 5*time.Minute)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect %s", serverPath)
	}
	dialOptions = append(dialOptions,
		grpc.WithContextDialer(npipeDialer),
	)

	// dial server
	grpcClientConn, err := grpc.Dial(serverPath, dialOptions...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect with %s", serverPath)
	}

	return grpcClientConn, nil
}
