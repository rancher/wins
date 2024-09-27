package service

import (
	"fmt"
	"os"
	"strings"

	"github.com/rancher/wins/pkg/defaults"
	"github.com/sirupsen/logrus"
)

func ConfigureRKE2ServiceDependency() error {
	logrus.Info("Configuring rke2 service dependencies")
	add := strings.ToLower(os.Getenv("CATTLE_ENABLE_WINS_SERVICE_DEPENDENCY")) == "true"

	rke2, serviceExists, err := Open("rke2")
	if err != nil {
		return fmt.Errorf("failed to open rke2 service while configuring service dependencies: %w", err)
	}

	if !serviceExists {
		logrus.Warn("Could not find rke2 service, will not attempt to configure service dependencies")
		return nil
	}
	defer rke2.Close()

	found := false
	for _, dep := range rke2.Config.Dependencies {
		logrus.Debugf("Found rke2 service dependency '%s'", dep)
		if dep == defaults.WindowsServiceName {
			logrus.Debugf("Found rancher-wins dependency set on rke2 service")
			found = true
			break
		}
	}

	saveChanges := false
	if found && add {
		logrus.Info("rke2 service dependency already configured")
		return nil
	}

	if !found && add {
		logrus.Info("Adding rancher-wins dependency on rke2 service")
		saveChanges = true
		rke2.Config.Dependencies = append(rke2.Config.Dependencies, defaults.WindowsServiceName)
	}

	if found && !add {
		logrus.Info("Removing rancher-wins dependency on rke2 service")
		saveChanges = true
		rke2.Config.Dependencies = removeFromSlice(defaults.WindowsServiceName, rke2.Config.Dependencies)
		if len(rke2.Config.Dependencies) == 0 {
			// Updating a service config with a nil or empty Dependencies slice will not have any effect.
			// Instead, '/' must be used to clear any remaining service dependencies.
			rke2.Config.Dependencies = append(rke2.Config.Dependencies, "/")
		}
	}

	if !saveChanges {
		logrus.Info("No modification to rke2 service dependencies required")
		return nil
	}

	logrus.Info("Updating rke2 service dependencies")
	err = rke2.UpdateConfig()
	if err != nil {
		return fmt.Errorf("failed to update rke2 config while creating service dependencies: %w", err)
	}

	return nil
}

func ConfigureDelayedStart() error {
	logrus.Info("Configuring start type for rancher-wins")
	delayedStart := strings.ToLower(os.Getenv("CATTLE_ENABLE_WINS_DELAYED_START")) == "true"

	wins, exists, err := Open(defaults.WindowsServiceName)
	if err != nil {
		return fmt.Errorf("failed to open %s service while configuring start type: %w", defaults.WindowsServiceName, err)
	}

	if !exists {
		logrus.Warnf("could not find the %s service, cannot configure service start type", defaults.WindowsServiceName)
		return nil
	}

	defer wins.Close()

	logrus.Infof("%s service has delayed auto start configured: %t", defaults.WindowsServiceName, wins.Config.DelayedAutoStart)

	if wins.Config.DelayedAutoStart != delayedStart {
		logrus.Infof("updating %s delayed auto start setting to %t", defaults.WindowsServiceName, delayedStart)
		wins.Config.DelayedAutoStart = delayedStart
		err = wins.UpdateConfig()
		if err != nil {
			return fmt.Errorf("failed to update %s service configuration while configuring service start type: %w", defaults.WindowsServiceName, err)
		}
		return nil
	}

	logrus.Infof("%s delayed start already set to %t", defaults.WindowsServiceName, delayedStart)

	return nil
}

func RefreshWinsService() error {
	logrus.Infof("Restarting the %s service", defaults.WindowsServiceDisplayName)
	winSrv, exists, err := Open(defaults.WindowsServiceName)
	if err != nil {
		logrus.Errorf("Cannot restart %s as the service failed to open: %v", defaults.WindowsServiceName, err)
		return fmt.Errorf("failed to refresh the %s service: %w", winSrv.Name, err)
	}

	if !exists {
		logrus.Errorf("Cannot restart %s as the service does not exist", defaults.WindowsServiceName)
		return nil
	}

	defer winSrv.Close()

	err = winSrv.Restart()
	if err != nil {
		return fmt.Errorf("failed to restart the %s service: %w", winSrv.Name, err)
	}

	return nil
}
