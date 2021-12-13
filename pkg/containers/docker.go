package containers

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	runtime "github.com/docker/docker/client"
	"github.com/pkg/errors"
)

type docker struct{}

func (d docker) GetContainer(ctx context.Context, labels map[string]string) (Container, error) {
	var cli, err = newDockerRuntimeClient()
	if err != nil {
		return Container{}, errors.Wrapf(ErrConnected, "failed to create docker client: %v", err)
	}
	defer func() { _ = cli.Close() }()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	filter := filters.NewArgs()
	for lk, lv := range labels {
		filter.Add("label", lk+"="+lv)
	}
	containerRawList, err := cli.ContainerList(ctx, types.ContainerListOptions{Filters: filter})
	if err != nil {
		if strings.Contains(err.Error(), "error during connect") {
			return Container{}, errors.Wrapf(ErrConnected, "failed to list container: %v", err)
		}
		return Container{}, errors.Wrap(err, "failed to list container")
	}
	if len(containerRawList) == 0 {
		return Container{}, ErrNotFound
	}
	if len(containerRawList) > 1 {
		return Container{}, ErrMultipleFound
	}

	var container Container
	for _, m := range containerRawList[0].Mounts {
		container.Mounts = append(container.Mounts, MountPoint{
			ReadOnly:      !m.RW,
			HostPath:      filepath.Clean(m.Source),
			ContainerPath: filepath.Clean(m.Destination),
		})
	}
	return container, nil
}

func newDockerRuntimeClient() (*runtime.Client, error) {
	return runtime.NewClientWithOpts(
		runtime.FromEnv,
		runtime.WithAPIVersionNegotiation(),
	)
}
