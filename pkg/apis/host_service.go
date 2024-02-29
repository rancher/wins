package apis

import (
	"context"

	"github.com/rancher/wins/pkg/converters"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/types"
	"golang.org/x/sys/windows/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type hostService struct {
}

func (s *hostService) GetVersion(_ context.Context, _ *types.Void) (resp *types.HostGetVersionResponse, respErr error) {
	defer panics.DealWith(func(recoverObj interface{}) {
		respErr = status.Errorf(codes.Unknown, "panic %v", recoverObj)
	})

	currentVersionRegKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not open registry key: %v", err)
	}
	defer currentVersionRegKey.Close()

	// construct response
	return &types.HostGetVersionResponse{
		Data: registryKeyToHostVersion(currentVersionRegKey),
	}, nil
}

func registryKeyToHostVersion(k registry.Key) *types.HostVersion {
	return &types.HostVersion{
		CurrentMajorVersionNumber: converters.GetIntStringFormRegistryKey(k, "CurrentMajorVersionNumber"),
		CurrentMinorVersionNumber: converters.GetIntStringFormRegistryKey(k, "CurrentMinorVersionNumber"),
		CurrentBuildNumber:        converters.GetStringFromRegistryKey(k, "CurrentBuildNumber"),
		UBR:                       converters.GetIntStringFormRegistryKey(k, "UBR"),
		ReleaseId:                 converters.GetStringFromRegistryKey(k, "ReleaseId"),
		BuildLabEx:                converters.GetStringFromRegistryKey(k, "BuildLabEx"),
		CurrentBuild:              converters.GetStringFromRegistryKey(k, "CurrentBuild"),
	}
}
