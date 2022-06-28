package proxy

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rancher/remotedialer"
	"github.com/rancher/wins/cmd/cmds/flags"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/rancher/wins/pkg/npipes"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/proxy"
	"github.com/urfave/cli/v2"
)

var _proxyFlags = []cli.Flag{
	&cli.GenericFlag{
		Name:  "publish",
		Usage: "[required] [list-argument] Publish a port or a range of ports, e.g.: TCP:443 TCP:80-81 (note: only TCP is supported)",
		Value: flags.NewListValue(),
	},
	&cli.StringFlag{
		Name:  "proxy",
		Usage: "[optional] Specifies the name of the proxy listening named pipe",
		Value: defaults.ProxyPipeName,
	},
}

var _proxyPorts []int

func _proxyRequestParser(cliCtx *cli.Context) (err error) {
	// Check if ports are provided
	publishList := flags.GetListValue(cliCtx, "publish")
	if publishList.IsEmpty() {
		return fmt.Errorf("No ports to publish")
	}

	// Parse ports
	publishPorts, err := publishList.Get()
	if err != nil {
		return fmt.Errorf("failed to parse --publish: %v", err)
	}
	ports, err := parsePublishes(publishPorts)
	if err != nil {
		return fmt.Errorf("failed to parse --publish %s: %v", publishPorts, err)
	}
	_proxyPorts = ports

	return nil
}

func parsePublishes(publishPorts []string) (ports []int, err error) {
	for _, pub := range publishPorts {
		publishes := strings.SplitN(pub, ":", 2)
		if len(publishes) != 2 {
			return nil, fmt.Errorf("could not parse publish %s", publishes)
		}

		// TODO(aiyengar2): expand support for UDP ports if an alternative to tcpproxy exists for UDP
		protocol := publishes[0]
		if protocol != "TCP" {
			return nil, fmt.Errorf("unsupported protocol %s, only TCP is supported", protocol)
		}

		portRanges := strings.SplitN(publishes[1], "-", 2)
		if len(portRanges) == 1 {
			number, err := strconv.ParseUint(portRanges[0], 10, 16)
			if err != nil {
				return nil, fmt.Errorf("could not parse port %s from expose %s: %v", portRanges[0], pub, err)
			}
			ports = append(ports, int(number))
			continue
		}

		if len(portRanges) == 2 {
			low, err := strconv.ParseUint(portRanges[0], 10, 16)
			if err != nil {
				return nil, errors.Wrapf(err, "could not parse port %s from expose %s", portRanges[0], pub)
			}
			high, err := strconv.ParseUint(portRanges[1], 10, 16)
			if err != nil {
				return nil, errors.Wrapf(err, "could not parse port %s from expose %s", portRanges[1], pub)
			}
			if low >= high {
				return nil, fmt.Errorf("could not accept the range %d - %d from expose %s", low, high, pub)
			}
			for number := low; number <= high; number++ {
				ports = append(ports, int(number))
			}
			continue
		}
		// port range has invalid format
		return nil, fmt.Errorf("could not parse expose %s", pub)
	}

	return ports, nil
}

func _proxyAction(cliCtx *cli.Context) (err error) {
	defer panics.Log()

	// Get hostname to identify backend connection
	hostname, err := os.Hostname()
	if err != nil {
		return errors.Wrap(err, "unable to get hostname")
	}
	proxyHeaders := http.Header{}
	proxyHeaders.Set(proxy.ClientIDHeader, hostname)

	// Set up proxy
	ctx := context.Background()
	pipe := cliCtx.String("proxy")
	pipePath := npipes.GetFullPath(pipe)
	connAuth := proxy.GetClientConnectAuthorizer(_proxyPorts)
	dialer, err := proxy.NewClientDialer(pipePath)
	if err != nil {
		return fmt.Errorf("Unable to get dialer to named pipe: %v", err)
	}
	onConn := proxy.GetClientOnConnect(_proxyPorts)

	return remotedialer.ConnectToProxy(ctx, fmt.Sprintf("ws://%s", pipe), proxyHeaders, connAuth, dialer, onConn)
}
