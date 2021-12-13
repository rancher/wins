package apis

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rancher/wins/pkg/containers"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/types"
)

type volumeService struct {
}

func (s *volumeService) Mount(ctx context.Context, req *types.VolumeMountRequest) (resp *types.Void, respErr error) {
	defer panics.DealWith(func(recoverObj interface{}) {
		respErr = status.Errorf(codes.Unknown, "panic %v", recoverObj)
	})

	// find out container
	selectors := req.GetSelectors()
	container, err := containers.Get(ctx, selectors)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not find container with %s: %v", strings.Join(selectors, ","), err)
	}

	// container:   c:/var/lib/kubelet/pods/xxx/volumes/kubernetes.io~secret/terway-token-yyy -> c:/var/run/secrets/kubernetes.io/serviceaccount
	// wins cli:    c:/var/run/secrets/kubernetes.io/serviceaccount -> c:/etc/kube-flannel/serviceaccount
	// wins server: c:/var/lib/kubelet/pods/xxx/volumes/kubernetes.io~secret/terway-token-yyy -> c:/etc/kube-flannel/serviceaccount

	// iterate to mount
	containerHostPathsMapping := make(map[string]string, len(container.Mounts))
	for _, m := range container.Mounts {
		// container_path -> real_host_path
		containerHostPathsMapping[m.ContainerPath] = m.HostPath
	}
	for _, m := range req.GetPaths() { // wins cli: container_path -> host_path
		hostPathSrc, ok := containerHostPathsMapping[m.GetSource()]
		if !ok {
			// NB(thxCode): nothing to do if not found
			continue
		}
		if _, err = os.Lstat(hostPathSrc); err != nil {
			// NB(thxCode): nothing to do if source is not found
			continue
		}

		// create parent directory
		hostPathDest := m.GetDestination()
		if err = os.MkdirAll(filepath.Dir(hostPathDest), os.ModePerm); err != nil {
			return nil, status.Errorf(codes.Internal, "could not prepare parent directory of %s: %v", hostPathDest, err)
		}

		// remove the legacy path
		if err = os.RemoveAll(hostPathDest); err != nil {
			return nil, status.Errorf(codes.Internal, "could not remove legacy path of %s: %v", hostPathDest, err)
		}

		// create symbolic link
		if err = os.Symlink(hostPathSrc, hostPathDest); err != nil {
			return nil, status.Errorf(codes.Internal, "could not mount %s to %s: %v", m.GetSource(), hostPathDest, err)
		}
	}

	return &types.Void{}, nil
}
