package app

import (
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/rancher/wins/cmd/grpcs"
	"github.com/rancher/wins/cmd/server/config"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func setupGRPCServerOptions(serverOptions []grpc.ServerOption, cfg *config.Config) ([]grpc.ServerOption, error) {
	ui := make([]grpc.UnaryServerInterceptor, 0)
	si := make([]grpc.StreamServerInterceptor, 0)

	// add logging middleware
	debug := cfg.Debug
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
		ui = append(ui,
			grpcs.LogrusUnaryServerInterceptor(),
		)
		si = append(si,
			grpcs.LogrusStreamServerInterceptor(),
		)
	}

	// add process path whitelist middleware
	processPathWhiteList := cfg.WhiteList.ProcessPaths
	if len(processPathWhiteList) != 0 {
		logrus.Debugf("Process path whitelist: %v", processPathWhiteList)
		ui = append(ui,
			grpcs.ProcessPathUnaryServerInterceptor(processPathWhiteList),
		)
	}

	if len(ui) != 0 {
		serverOptions = append(serverOptions,
			grpc_middleware.WithUnaryServerChain(ui...),
		)
	}
	if len(si) != 0 {
		serverOptions = append(serverOptions,
			grpc_middleware.WithStreamServerChain(si...),
		)
	}
	return serverOptions, nil
}
