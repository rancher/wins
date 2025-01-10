package service

import (
	"fmt"

	"github.com/rancher/wins/pkg/defaults"
)

type RancherWinsService struct {
	Service
}

func OpenRancherWinsService() (*RancherWinsService, bool, error) {
	winsSvc, exists, err := Open(defaults.WindowsServiceName)
	if err != nil {
		return nil, false, fmt.Errorf("error encountered opening %s service: %w", defaults.WindowsServiceName, err)
	}

	if !exists {
		return nil, false, fmt.Errorf("%s service does not exist", defaults.WindowsServiceName)
	}

	x := &RancherWinsService{
		*winsSvc,
	}

	return x, exists, nil
}
