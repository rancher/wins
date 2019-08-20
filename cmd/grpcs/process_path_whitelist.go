package grpcs

import (
	"context"
	"path/filepath"

	"github.com/rancher/wins/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ProcessPathUnaryServerInterceptor(whitelist []string) grpc.UnaryServerInterceptor {
	whitelistIndex := make(map[string]struct{}, len(whitelist))
	for _, wl := range whitelist {
		whitelistIndex[filepath.Clean(wl)] = struct{}{}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if info.FullMethod == "/wins.ProcessService/Start" {
			if psr, ok := req.(*types.ProcessStartRequest); ok {
				if _, exist := whitelistIndex[filepath.Clean(psr.Path)]; !exist {
					return nil, status.Errorf(codes.InvalidArgument, "invalid path")
				}
			}
		}

		return handler(ctx, req)
	}
}
