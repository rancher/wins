package containers

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	ErrConnected     = errors.New("error connected")
	ErrNotFound      = errors.New("container not found")
	ErrMultipleFound = errors.New("multiple container found")
)

type Runtime interface {
	GetContainer(ctx context.Context, labels map[string]string) (Container, error)
}

type Container struct {
	Mounts []MountPoint
}

type MountPoint struct {
	ReadOnly      bool
	HostPath      string
	ContainerPath string
}

// Get returns the target container with given labels.
func Get(ctx context.Context, labels []string) (c Container, err error) {
	var labelSelector = make(map[string]string, len(labels))
	for i := range labels {
		var lbv = strings.SplitN(labels[i], "=", 2)
		if len(lbv) != 2 {
			err = errors.Errorf("failed to parse selector: %q", labels[i])
		}
		labelSelector[lbv[0]] = lbv[1]
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	var r Runtime = docker{}
	c, err = r.GetContainer(ctx, labelSelector)
	if errors.Is(err, ErrConnected) {
		r = containerd{}
		c, err = r.GetContainer(ctx, labelSelector)
	}
	return
}
