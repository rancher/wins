package proxy

import (
	"net/http"

	"github.com/rancher/remotedialer"
)

const (
	// ClientIDHeader is the key used in the HTTP header to identify a given incoming connection to the server
	ClientIDHeader = "rancher-wins-cli-proxy"
)

// GetServerAuthorizer returns authorizer used to get client information from the request made to the server
func GetServerAuthorizer() remotedialer.Authorizer {
	return func(req *http.Request) (clientKey string, authed bool, err error) {
		return req.Header.Get(ClientIDHeader), true, nil
	}
}
