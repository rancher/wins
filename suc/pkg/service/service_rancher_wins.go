package service

import (
	"fmt"

	"github.com/rancher/wins/pkg/defaults"
	"github.com/sirupsen/logrus"
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

func (rw *RancherWinsService) ConfigureDelayedStart(enabled bool) error {
	logrus.Infof("%s service has delayed auto start configured: %t", defaults.WindowsServiceName, rw.Config.DelayedAutoStart)
	if rw.Config.DelayedAutoStart != enabled {
		logrus.Infof("updating %s delayed auto start setting to %t", defaults.WindowsServiceName, enabled)
		rw.Config.DelayedAutoStart = enabled
		err := rw.UpdateConfig()
		if err != nil {
			return fmt.Errorf("failed to update %s service configuration while configuring service start type: %w", defaults.WindowsServiceName, err)
		}
	} else {
		logrus.Infof("%s delayed start already set to %t, nothing to do", defaults.WindowsServiceName, enabled)
	}
	return nil
}
