package containers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"

	"github.com/rancher/wins/pkg/npipes"
)

type containerd struct{}

func (d containerd) GetContainer(ctx context.Context, labels map[string]string) (Container, error) {
	var cli, conn, err = newContainerdRuntimeClient()
	if err != nil {
		return Container{}, errors.Wrapf(ErrConnected, "failed to create containerd client: %v", err)
	}
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	filter := &runtime.ContainerFilter{LabelSelector: labels}
	containerRawListResp, err := cli.ListContainers(ctx,
		&runtime.ListContainersRequest{Filter: filter})
	if err != nil {
		return Container{}, errors.Wrap(err, "failed to list container")
	}
	var containerRawList = containerRawListResp.GetContainers()
	if len(containerRawList) == 0 {
		return Container{}, ErrNotFound
	}
	if len(containerRawList) > 1 {
		return Container{}, ErrMultipleFound
	}

	containerStatusResp, err := cli.ContainerStatus(ctx,
		&runtime.ContainerStatusRequest{ContainerId: containerRawList[0].Id})
	if err != nil {
		return Container{}, errors.Wrapf(err, "failed to query container: %s", containerRawList[0].Id)
	}

	var container Container
	for _, m := range containerStatusResp.GetStatus().GetMounts() {
		container.Mounts = append(container.Mounts, MountPoint{
			ReadOnly:      m.GetReadonly(),
			HostPath:      filepath.Clean(m.GetHostPath()),
			ContainerPath: filepath.Clean(m.GetContainerPath()),
		})
	}
	return container, nil
}

func newContainerdRuntimeClient() (cli runtime.RuntimeServiceClient, conn *grpc.ClientConn, err error) {
	var addr = npipes.GetFullPath("containerd-containerd")
	if host := os.Getenv("CONTAINERD_HOST"); host != "" {
		var parts = strings.SplitN(host, "://", 2)
		if len(parts) == 1 {
			err = errors.Errorf("unable to parse contaienrd host `%s`", host)
			return
		}
		var proto = parts[0]
		if proto != "npipe" {
			err = errors.Errorf("invalid containerd host `%s`", host)
		}
		addr = host
	}
	var dialer npipes.Dialer
	dialer, err = npipes.NewDialer(addr, 32*time.Second)
	if err != nil {
		return
	}
	conn, err = grpc.Dial(addr, grpc.WithInsecure(), grpc.WithContextDialer(dialer))
	if err != nil {
		return
	}
	cli = runtime.NewRuntimeServiceClient(conn)
	return
}
