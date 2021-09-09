package process

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/wins/cmd/client/internal"
	"github.com/rancher/wins/cmd/cmds/flags"
	"github.com/rancher/wins/cmd/outputs"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/paths"
	"github.com/rancher/wins/pkg/types"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var _runFlags = internal.NewGRPCClientConn(
	[]cli.Flag{
		cli.StringFlag{
			Name:  "path",
			Usage: "[required] Runs the binary located in the host",
		},
		cli.GenericFlag{
			Name:  "args",
			Usage: "[optional] [list-argument] Specifies the arguments for binary when running, e.g.: x=y b=c",
			Value: flags.NewListValue(),
		},
		cli.GenericFlag{
			Name:  "exposes",
			Usage: "[optional] [list-argument] Exposes a port or a range of ports, e.g.: TCP:443 UDP:4789-4790",
			Value: flags.NewListValue(),
		},
		cli.GenericFlag{
			Name:  "envs",
			Usage: "[optional] [list-argument] Specifies the environment variables for binary when running, e.g.: x=y b=c",
			Value: flags.NewListValue(),
		},
		cli.StringFlag{
			Name:  "dir",
			Usage: "[optional] Specifies the running directory, otherwise run in the path parent directory",
		},
	},
)

var _runStartRequest *types.ProcessStartRequest

func _runRequestParser(cliCtx *cli.Context) (err error) {
	// validate
	path := cliCtx.String("path")
	if path == "" {
		return errors.Errorf("--path is required")
	}
	path, err = paths.GetBinaryPath(path)
	if err != nil {
		return errors.Wrapf(err, "--path is invalid")
	}
	checksum, err := paths.GetFileSHA1Hash(path)
	if err != nil {
		return errors.Wrapf(err, "failed to get checksum for --path %s", path)
	}
	var exposes []*types.ProcessExpose
	if exposesList := flags.GetListValue(cliCtx, "exposes"); !exposesList.IsEmpty() {
		exposesListValue, err := exposesList.Get()
		if err != nil {
			return errors.Wrapf(err, "failed to parse --exposes")
		}
		exposes, err = parseExposes(exposesListValue)
		if err != nil {
			return errors.Wrapf(err, "failed to parse --exposes %s", exposes)
		}
	}
	args, err := flags.GetListValue(cliCtx, "args").Get()
	if err != nil {
		return errors.Wrapf(err, "failed to parse --args")
	}
	envs, err := flags.GetListValue(cliCtx, "envs").Get()
	if err != nil {
		return errors.Wrapf(err, "failed to parse --envs")
	}
	dir := cliCtx.String("dir")
	if dir == "" {
		dir = filepath.Dir(path)
	}

	// parse
	_runStartRequest = &types.ProcessStartRequest{
		Checksum: checksum,
		Path:     path,
		Args:     args,
		Exposes:  exposes,
		Envs:     envs,
		Dir:      dir,
	}

	return nil
}

func parseExposes(exposes []string) ([]*types.ProcessExpose, error) {
	var runExposes []*types.ProcessExpose
	for _, exp := range exposes {
		exposes := strings.SplitN(exp, ":", 2)
		if len(exposes) != 2 {
			return nil, errors.Errorf("could not parse expose %s", exp)
		}

		protocol := exposes[0]
		portRanges := strings.SplitN(exposes[1], "-", 2)
		if len(portRanges) == 1 {
			number, err := strconv.Atoi(portRanges[0])
			if err != nil {
				return nil, errors.Wrapf(err, "could not parse port %s from expose %s", portRanges[0], exp)
			}

			runExposes = append(runExposes, &types.ProcessExpose{
				Protocol: types.RunExposeProtocol(types.RunExposeProtocol_value[protocol]),
				Port:     int32(number),
			})
		} else if len(portRanges) == 2 {
			low, err := strconv.Atoi(portRanges[0])
			if err != nil {
				return nil, errors.Wrapf(err, "could not parse port %s from expose %s", portRanges[0], exp)
			}
			hig, err := strconv.Atoi(portRanges[1])
			if err != nil {
				return nil, errors.Wrapf(err, "could not parse port %s from expose %s", portRanges[1], exp)
			}
			if low >= hig {
				return nil, errors.Errorf("could not accept the range %d - %d from expose %s", low, hig, exp)
			}

			for number := low; number <= hig; number++ {
				runExposes = append(runExposes, &types.ProcessExpose{
					Protocol: types.RunExposeProtocol(types.RunExposeProtocol_value[protocol]),
					Port:     int32(number),
				})
			}
		} else {
			return nil, errors.Errorf("could not parse expose %s", exp)
		}
	}

	return runExposes, nil
}

func _runAction(cliCtx *cli.Context) (err error) {
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
	client := types.NewProcessServiceClient(grpcClientConn)

	// start
	startResp, err := client.Start(ctx, _runStartRequest)
	if err != nil {
		return err
	}
	processName := startResp.GetData().GetValue()

	// keep alive
	keepAliveStream, err := client.KeepAlive(ctx)
	if err != nil {
		return err
	}
	go func() {
		doneC := keepAliveStream.Context().Done()
		for {
			select {
			case <-doneC:
				return
			default:
			}

			if err := keepAliveStream.Send(&types.ProcessKeepAliveRequest{
				Data: &types.ProcessName{Value: processName},
			}); err != nil {
				logrus.Errorf("Failed to keep alive: %v", err)
				return
			}

			time.Sleep(5 * time.Minute)
		}
	}()

	// wait
	waitC := make(chan struct{})
	stopC := combineSignals(waitC, func() { // stop the process while finish waiting or kill by signals
		_, closeErr := keepAliveStream.CloseAndRecv()
		if err == nil {
			err = closeErr
		}
	})

	waitStream, err := client.Wait(ctx, &types.ProcessWaitRequest{
		Data: &types.ProcessName{Value: processName},
	})
	if err != nil {
		return err
	}
	writer, errWriter := cliCtx.App.Writer, cliCtx.App.ErrWriter
	for err == nil {
		resp, recvErr := waitStream.Recv()
		if recvErr != nil {
			err = recvErr
			continue
		}

		switch opts := resp.GetOptions().(type) {
		case *types.ProcessWaitResponse_StdOut:
			err = outputs.Json(writer, opts.StdOut)
		case *types.ProcessWaitResponse_StdErr:
			err = outputs.Json(errWriter, opts.StdErr)
		}
	}

	close(waitC)
	<-stopC

	return
}

func combineSignals(doneC <-chan struct{}, cleanupFn func()) <-chan struct{} {
	stopChan := make(chan struct{})
	closeStopChan := func() {
		defer panics.Ignore()

		if cleanupFn != nil {
			cleanupFn()
		}
		close(stopChan)
	}

	signals := make(chan os.Signal, 1<<10)
	go func() {
		for {
			select {
			case <-signals:
				closeStopChan()
				return
			case <-doneC:
				closeStopChan()
				return
			}
		}
	}()
	signal.Notify(signals, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM)

	return stopChan
}

func runCommand() cli.Command {
	return cli.Command{
		Name:   "run",
		Usage:  "Run a process",
		Flags:  _runFlags,
		Before: _runRequestParser,
		Action: _runAction,
	}
}
