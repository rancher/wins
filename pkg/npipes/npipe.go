package npipes

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/pkg/errors"
	"github.com/rancher/wins/pkg/defaults"
)

type Dialer func(context.Context, string) (net.Conn, error)

// NewDialer creates a Dialer to connect to a named pipe by `path`.
func NewDialer(path string, timeout time.Duration) (Dialer, error) {
	path, err := parsePath(path)
	if err != nil {
		return nil, err
	}

	return func(_ context.Context, _ string) (conn net.Conn, e error) {
		return winio.DialPipe(path, &timeout)
	}, nil
}

// New creates a named pipe with `path`, `sddl` and `bufferSize`
// `sddl`: a format string of the Security Descriptor Definition Language, default is builtin administrators and local system
// `bufferSize`: measurement is KB, default is 64
// refer:
// 	- https://docs.microsoft.com/en-us/windows/desktop/secauthz/security-descriptor-string-format
//  - https://docs.microsoft.com/en-us/windows/desktop/secauthz/ace-strings
func New(path, sddl string, bufferSize int32) (net.Listener, error) {
	path, err := parsePath(path)
	if err != nil {
		return nil, err
	}

	if sddl == "" {
		// Allow Administrators and SYSTEM, plus whatever additional users or groups were specified
		sddl = defaults.PermissionBuiltinAdministratorsAndLocalSystem
	}
	if bufferSize == 0 {
		// Use 64KB buffers to improve performance
		bufferSize = 64
	}
	bufferSize *= int32(1 << 10)

	pipeConfig := winio.PipeConfig{
		SecurityDescriptor: sddl,
		MessageMode:        true,
		InputBufferSize:    bufferSize,
		OutputBufferSize:   bufferSize,
	}

	listener, err := winio.ListenPipe(path, &pipeConfig)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

// GetFullPath returns the full path with the named pipe name
func GetFullPath(name string) string {
	return fmt.Sprintf("npipe:////./pipe/%s", name)
}

func parsePath(path string) (string, error) {
	sps := strings.SplitN(path, "://", 2)
	if len(sps) != 2 {
		return "", errors.Errorf("could not recognize path: %s", path)
	}

	if sps[0] != "npipe" {
		return "", errors.Errorf("could not recognize schema: %s", sps[0])
	}
	npipePath := sps[1]

	return strings.ReplaceAll(npipePath, "/", `\`), nil
}
