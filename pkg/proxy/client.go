package proxy

import (
	"context"
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"
	"github.com/gorilla/websocket"
	"github.com/rancher/remotedialer"
	"github.com/rancher/wins/pkg/npipes"
	"inet.af/tcpproxy"
)

// NewClientDialer returns a websocket.Dialer that dials a named pipe
func NewClientDialer(path string) (dialer *websocket.Dialer, err error) {
	path, err = npipes.ParsePath(path)
	if err != nil {
		return nil, err
	}
	return &websocket.Dialer{
		NetDialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return winio.DialPipeContext(ctx, path)
		},
	}, nil
}

// GetClientConnectAuthorizer returns the client's connect authorizer based on the provided ports
func GetClientConnectAuthorizer(ports []int) remotedialer.ConnectAuthorizer {
	validAddresses := make(map[string]bool, len(ports))
	for _, p := range ports {
		validAddresses[fmt.Sprintf("localhost:%d", p)] = true
	}
	return func(proto, address string) bool {
		return proto == "tcp" && validAddresses[address]
	}
}

// GetClientOnConnect returns the onConnect function used by the client to set up the tcpproxy
func GetClientOnConnect(ports []int) func(context.Context, *remotedialer.Session) error {
	return func(c context.Context, s *remotedialer.Session) error {
		proxy := &tcpproxy.Proxy{}
		for _, p := range ports {
			listenAddress := fmt.Sprintf(":%d", p)
			forwardAddress := fmt.Sprintf("localhost:%d", p)
			dialContext := func(ctx context.Context, _, _ string) (net.Conn, error) {
				return s.Dial(ctx, "tcp", forwardAddress)
			}
			proxy.AddRoute(listenAddress, &tcpproxy.DialProxy{DialContext: dialContext})
		}
		if err := proxy.Start(); err != nil {
			return err
		}
		<-c.Done()
		proxy.Close()
		return nil
	}
}
