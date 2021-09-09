package apis

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/paths"
	"github.com/rancher/wins/pkg/types"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	processPrefix = "rancher-wins-"
)

type processService struct {
}

func (s *processService) Start(ctx context.Context, req *types.ProcessStartRequest) (resp *types.ProcessStartResponse, respErr error) {
	defer panics.DealWith(func(recoverObj interface{}) {
		respErr = status.Errorf(codes.Unknown, "panic %v", recoverObj)
	})

	// ensure target bin & checksum
	binaryPath := filepath.Clean(req.GetPath())
	if err := paths.EnsureBinary(binaryPath, req.GetChecksum()); err != nil {
		return nil, status.Errorf(codes.NotFound, "could not found binary: %v", err)
	}

	// could not change the name of process in windows by default, a trick way is to rename the execution binary with a special prefix
	binaryPathRN := renameBinary(binaryPath)
	if err := paths.MoveFile(binaryPath, binaryPathRN); err != nil {
		return nil, status.Errorf(codes.Internal, "could not rename binary: %v", err)
	}

	// create process
	p, err := s.create(ctx, binaryPathRN, req.GetDir(), req.GetArgs(), req.GetEnvs(), toFirewallRules(req.GetExposes()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not create process: %v", err)
	}

	return &types.ProcessStartResponse{
		Data: &types.ProcessName{Value: p.name},
	}, nil
}

func renameBinary(srcPath string) string {
	return filepath.Join(filepath.Dir(srcPath), processPrefix+filepath.Base(srcPath))
}

func toFirewallRules(exposes []*types.ProcessExpose) string {
	exposePair := make([]string, 0, len(exposes))

	for _, expose := range exposes {
		if expose.GetPort() != 0 {
			exposePair = append(exposePair, fmt.Sprintf("%s-%d", expose.GetProtocol().String(), expose.GetPort()))
		}
	}

	return strings.Join(exposePair, " ")
}

func (s *processService) Wait(req *types.ProcessWaitRequest, stream types.ProcessService_WaitServer) (respErr error) {
	defer panics.DealWith(func(recoverObj interface{}) {
		respErr = status.Errorf(codes.Unknown, "panic %v", recoverObj)
	})

	pname := req.GetData().GetValue()

	p, err := s.getFromPool(pname)
	if err != nil {
		return status.Errorf(codes.Internal, "could not get process %s: %v", pname, err)
	}
	if p == nil {
		return status.Errorf(codes.NotFound, "could not find process %s", pname)
	}

	errg := &errgroup.Group{}
	errg.Go(func() error {
		defer panics.Log()
		defer p.stdout.Close()

		bs := make([]byte, 1<<10)
		for {
			readSize, err := p.stdout.Read(bs)
			if err != nil {
				if io.EOF != err && io.ErrClosedPipe != err {
					return err
				}
				break
			}

			if readSize > 0 {
				err = stream.Send(&types.ProcessWaitResponse{
					Options: &types.ProcessWaitResponse_StdOut{
						StdOut: bs[:readSize],
					},
				})
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
	errg.Go(func() error {
		defer panics.Log()
		defer p.stderr.Close()

		bs := make([]byte, 1<<10)
		for {
			readSize, err := p.stderr.Read(bs)
			if err != nil {
				if io.EOF != err && io.ErrClosedPipe != err {
					return err
				}
				break
			}

			if readSize > 0 {
				err = stream.Send(&types.ProcessWaitResponse{
					Options: &types.ProcessWaitResponse_StdErr{
						StdErr: bs[:readSize],
					},
				})
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
	errg.Go(func() error {
		defer panics.Log()

		return p.wait()
	})
	if err := errg.Wait(); err != nil {
		return status.Errorf(codes.Internal, "could not wait process %s: %v", pname, err)
	}

	return nil
}

func (s *processService) KeepAlive(stream types.ProcessService_KeepAliveServer) (respErr error) {
	defer panics.DealWith(func(recoverObj interface{}) {
		respErr = status.Errorf(codes.Unknown, "panic %v", recoverObj)
	})

	var pname string
	for {
		req, err := stream.Recv()
		if err != nil {
			break
		}

		pname = req.GetData().GetValue()
	}

	if pname == "" {
		return status.Errorf(codes.InvalidArgument, "could not find process with a blank string %s", pname)
	}

	p, err := s.getFromPool(pname)
	if err != nil {
		return status.Errorf(codes.Internal, "could not get process %s: %v", pname, err)
	}
	if p != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = p.kill(ctx)
		if err != nil {
			return status.Errorf(codes.Internal, "could not kill process %s: %v", pname, err)
		}
	}

	stream.SendAndClose(&types.Void{})
	return nil
}
