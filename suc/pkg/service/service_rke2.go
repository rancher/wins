package service

import (
	"fmt"

	"github.com/rancher/wins/pkg/defaults"
	"github.com/sirupsen/logrus"
)

type RKE2Service struct {
	Service
}

func OpenRKE2Service() (*RKE2Service, bool, error) {
	rke2Svc, exists, err := Open("rke2")
	if err != nil {
		return nil, exists, fmt.Errorf("failed to open rke2 service: %w", err)
	}

	if !exists {
		logrus.Warn("Could not find rke2 service")
		return nil, exists, nil
	}

	x := &RKE2Service{
		*rke2Svc,
	}

	return x, exists, nil
}

func (rke2 *RKE2Service) HasRancherWinsServiceDependency() (bool, error) {
	for _, dep := range rke2.Config.Dependencies {
		logrus.Debugf("Found rke2 service dependency '%s'", dep)
		if dep == defaults.WindowsServiceName {
			logrus.Debug("Found rancher-wins dependency set on rke2 service")
			return true, nil
		}
	}
	return false, nil
}

func (rke2 *RKE2Service) AddRancherWinsServiceDependency() error {
	rke2.Config.Dependencies = append(rke2.Config.Dependencies, defaults.WindowsServiceName)
	return rke2.UpdateConfig()
}

func (rke2 *RKE2Service) RemoveRancherWinsServiceDependency() error {
	rke2.Config.Dependencies = removeAllFromSlice(defaults.WindowsServiceName, rke2.Config.Dependencies)
	if len(rke2.Config.Dependencies) == 0 {
		// Updating a service config with a nil or empty Dependencies slice will not have any effect.
		// Instead, '/' must be used to clear any remaining service dependencies.
		rke2.Config.Dependencies = append(rke2.Config.Dependencies, "/")
	}
	return rke2.UpdateConfig()
}
