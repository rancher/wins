package grpcs

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/rancher/wins/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ProcessPathUnaryServerInterceptor(whitelist []string) grpc.UnaryServerInterceptor {
	whitelistIndex := make(map[string]struct{}, len(whitelist))
	for _, wl := range whitelist {
		path := strings.ToLower(filepath.Clean(wl))
		whitelistIndex[path] = struct{}{}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if info.FullMethod == "/wins.ProcessService/Start" {
			if psr, ok := req.(*types.ProcessStartRequest); ok {
				path := strings.ToLower(filepath.Clean(psr.Path))
				if _, exist := whitelistIndex[path]; !exist {
					return nil, status.Errorf(codes.InvalidArgument, "invalid path")
				}
			}
		}

		return handler(ctx, req)
	}
}
