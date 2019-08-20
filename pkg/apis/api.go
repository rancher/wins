package apis

import (
	"context"
	"net"

	"github.com/pkg/errors"
	"github.com/rancher/wins/pkg/npipes"
	"github.com/rancher/wins/pkg/types"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Server struct {
	listener net.Listener
	server   *grpc.Server
}

func (s *Server) Close() error {
	s.server.Stop()
	return s.listener.Close()
}

func (s *Server) Serve(ctx context.Context) error {
	srv := s.server

	// register service
	types.RegisterHostServiceServer(srv, &hostService{})
	types.RegisterNetworkServiceServer(srv, &networkService{})
	types.RegisterHnsServiceServer(srv, &hnsService{})
	types.RegisterRouteServiceServer(srv, &routeService{})
	types.RegisterProcessServiceServer(srv, &processService{})
	types.RegisterApplicationServiceServer(srv, &applicationService{})

	logrus.Infof("Listening on %v", s.listener.Addr())

	return srv.Serve(s.listener)
}

func NewServer(listen string, serverOptions []grpc.ServerOption) (*Server, error) {
	listenPath := npipes.GetFullPath(listen)
	listener, err := npipes.New(listenPath, "", 0)
	if err != nil {
		return nil, errors.Wrapf(err, "could not listen %s", listenPath)
	}

	return &Server{
		listener: listener,
		server:   grpc.NewServer(serverOptions...),
	}, nil
}
