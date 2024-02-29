package apis

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/paths"
	"github.com/rancher/wins/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type applicationService struct {
}

func (s *applicationService) Info(_ context.Context, _ *types.Void) (resp *types.ApplicationInfoResponse, respErr error) {
	defer panics.DealWith(func(recoverObj interface{}) {
		respErr = status.Errorf(codes.Unknown, "panic %v", recoverObj)
	})

	info, err := getActualInfo()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get actual info: %v", err)
	}

	return &types.ApplicationInfoResponse{
		Info: info,
	}, nil
}

func getActualInfo() (*types.ApplicationInfo, error) {
	serverChecksum, err := paths.GetBinarySHA1Hash(os.Args[0])
	if err != nil {
		return nil, errors.Wrap(err, "could not get file checksum")
	}

	return &types.ApplicationInfo{
		Checksum: serverChecksum,
		Version:  defaults.AppVersion,
		Commit:   defaults.AppCommit,
	}, nil
}
