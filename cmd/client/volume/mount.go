package volume

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"github.com/rancher/wins/cmd/client/internal"
	"github.com/rancher/wins/cmd/cmds/flags"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/types"
)

var _mountFlags = internal.NewGRPCClientConn(
	[]cli.Flag{
		cli.GenericFlag{
			Name:  "selectors",
			Usage: "[required] [list-argument] Specifies the labels of the container where running inside, it's used to identify the process proxying handler, e.g.: x=y b=c",
			Value: flags.NewListValue(),
		},
		cli.GenericFlag{
			Name:  "paths",
			Usage: "[required] [list-argument] Specifies the paths to link, in form of ${container path}:${host path}, e.g.: c:\\var\\run\\secrets\\kubernetes.io\\serviceaccount:c:\\etc\\kube-flannel\\serviceaccount",
			Value: flags.NewListValue(),
		},
	},
)

var _mountRequest *types.VolumeMountRequest

func _mountRequestParser(cliCtx *cli.Context) (err error) {
	// validate
	selectors, err := flags.GetListValue(cliCtx, "selectors").Get()
	if err != nil {
		return errors.Wrapf(err, "failed to parse --selectors")
	}
	if len(selectors) == 0 {
		return errors.New("must specifies --selectors to identify the running container")
	}
	var paths []*types.VolumePath
	if pathsList := flags.GetListValue(cliCtx, "paths"); !pathsList.IsEmpty() {
		pathsListValue, err := pathsList.Get()
		if err != nil {
			return errors.Wrapf(err, "failed to parse --paths")
		}
		paths, err = parsePaths(pathsListValue)
		if err != nil {
			return errors.Wrapf(err, "failed to parse --paths=\"%s\"", strings.Join(pathsListValue, " "))
		}
	}

	// parse
	_mountRequest = &types.VolumeMountRequest{
		Selectors: selectors,
		Paths:     paths,
	}

	return nil
}

func parsePaths(paths []string) ([]*types.VolumePath, error) {
	var volumePaths []*types.VolumePath
	for _, p := range paths {
		if strings.HasSuffix(p, ":") || strings.HasPrefix(p, ":") {
			return nil, errors.Errorf("could not parse path %s", p)
		}
		var src, dest string
		path := strings.Split(p, ":")
		switch len(path) {
		case 1, 2:
			// /container_path
			// c:/container_path
			src = p
			dest = p
		case 3:
			// c:/container_path:/host_path
			// /container_path:c:/host_path
			if len(path[0]) == 1 {
				src = path[0] + ":" + path[1]
				dest = path[2]
			} else {
				src = path[0]
				dest = path[1] + ":" + path[2]
			}
		case 4:
			// c:/container_path:c:/host_path
			src = path[0] + ":" + path[1]
			dest = path[2] + ":" + path[3]
		default:
			return nil, errors.Errorf("could not parse path %s", p)
		}
		volumePaths = append(volumePaths, &types.VolumePath{
			Source:      filepath.Clean(src),
			Destination: filepath.Clean(dest),
		})
	}

	return volumePaths, nil
}

func _mountAction(cliCtx *cli.Context) (err error) {
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
	client := types.NewVolumeServiceClient(grpcClientConn)

	_, err = client.Mount(ctx, _mountRequest)

	return
}

func mountCommand() cli.Command {
	return cli.Command{
		Name:   "mount",
		Usage:  "Mount paths",
		Flags:  _mountFlags,
		Before: _mountRequestParser,
		Action: _mountAction,
	}
}
