package apis

import (
	"context"
	"net"
	"net/http"

	"github.com/Microsoft/go-winio"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/rancher/remotedialer"
	"github.com/rancher/wins/pkg/npipes"
	"github.com/rancher/wins/pkg/proxy"
	"github.com/rancher/wins/pkg/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type Server struct {
	listener net.Listener
	proxy    proxyServer
	server   *grpc.Server
}

type proxyServer struct {
	listener net.Listener
	ports    []int
}

func (s *Server) Close() error {
	s.server.Stop()
	return multierror.Append(s.listener.Close(), s.proxy.listener.Close())
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
	types.RegisterVolumeServiceServer(srv, &volumeService{})

	errg, _ := errgroup.WithContext(ctx)

	errg.Go(func() error {
		logrus.Infof("Listening on %v", s.listener.Addr())
		return srv.Serve(s.listener)
	})

	errg.Go(func() error {
		logrus.Infof("Listening on %v", s.proxy.listener.Addr())
		handler := remotedialer.New(proxy.GetServerAuthorizer(), remotedialer.DefaultErrorWriter)
		handler.ClientConnectAuthorizer = proxy.GetClientConnectAuthorizer(s.proxy.ports)
		return http.Serve(s.proxy.listener, handler)
	})

	return errg.Wait()
}

func NewServer(listen string, serverOptions []grpc.ServerOption, proxy string, proxyPorts []int) (*Server, error) {
	listenPath := npipes.GetFullPath(listen)
	listener, err := npipes.New(listenPath, "", 0)
	if err != nil {
		return nil, errors.Wrapf(err, "could not listen %s", listenPath)
	}

	proxyPath := npipes.GetFullPath(proxy)
	path, err := npipes.ParsePath(proxyPath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse path %s", proxyPath)
	}
	proxyListener, err := winio.ListenPipe(path, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "could not listen %s", proxyPath)
	}
	logrus.Infof("listening for tcp requests on %s destined for: %v", proxy, proxyPorts)

	return &Server{
		listener: listener,
		proxy: proxyServer{
			listener: proxyListener,
			ports:    proxyPorts,
		},
		server: grpc.NewServer(serverOptions...),
	}, nil
}
