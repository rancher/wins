package service

import (
	"fmt"
	"os"
	"strings"

	"github.com/rancher/wins/pkg/defaults"
	"github.com/sirupsen/logrus"
)

// ConfigureRKE2ServiceDependency creates a service dependency between rke2 and rancher-wins. This results in
// rancher-wins becoming a dependant service for rke2, preventing rke2 startup until rancher-wins is ready. This
// ensures that rancher-wins and rke2 do not interfere one another during start up (For example, due to CNI reconfiguration).
// As a side effect, the rancher-wins service cannot be stopped if rke2 is still running. To restart rancher-wins,
// this dependency must be temporarily removed.
func ConfigureRKE2ServiceDependency() error {
	logrus.Info("Configuring rke2 service dependencies")
	add := strings.ToLower(os.Getenv("CATTLE_ENABLE_WINS_SERVICE_DEPENDENCY")) == "true"

	rke2, serviceExists, err := OpenRKE2Service()
	if err != nil {
		return fmt.Errorf("failed to open rke2 service while configuring service dependencies: %w", err)
	}

	if !serviceExists {
		logrus.Warn("Could not find rke2 service, will not attempt to configure service dependencies")
		return nil
	}
	defer rke2.Close()

	found, err := rke2.HasRancherWinsServiceDependency()
	if err != nil {
		return fmt.Errorf("error encountered determining rke2 service dependencies: %w", err)
	}

	if !found && !add {
		logrus.Info("rke2 service dependency not enabled, nothing to do")
		return nil
	}

	if found && add {
		logrus.Info("rke2 service dependency already configured, nothing to do")
		return nil
	}

	if !found && add {
		logrus.Info("Adding rancher-wins dependency on rke2 service")
		err = rke2.AddRancherWinsServiceDependency()
		if err != nil {
			return fmt.Errorf("error encountered adding rke2 service dependency: %w", err)
		}
	}

	if found && !add {
		logrus.Info("Removing rancher-wins dependency on rke2 service")
		err = rke2.RemoveRancherWinsServiceDependency()
		if err != nil {
			return fmt.Errorf("error encountered adding rke2 service dependency: %w", err)
		}
	}

	return nil
}

// ConfigureWinsDelayedStart opens the rancher-wins service and enables the `DelayedAutoStart` flag.
// Enabling this flag does not require a restart of the service.
func ConfigureWinsDelayedStart() error {
	logrus.Info("Configuring start type for rancher-wins")
	delayedStart := strings.ToLower(os.Getenv("CATTLE_ENABLE_WINS_DELAYED_START")) == "true"

	wins, exists, err := OpenRancherWinsService()
	if err != nil {
		return fmt.Errorf("failed to open %s service while configuring start type: %w", defaults.WindowsServiceName, err)
	}

	if !exists {
		logrus.Warnf("could not find the %s service, cannot configure service start type", defaults.WindowsServiceName)
		return nil
	}

	defer wins.Close()

	err = wins.ConfigureDelayedStart(delayedStart)
	if err != nil {
		return fmt.Errorf("error encountered configuring delayed start for %s service: %w", defaults.WindowsServiceName, err)
	}

	return nil
}

// RefreshWinsService restarts the rancher-wins service. If a service dependency has
// been configured on the rke2 service, the dependency will be temporarily removed and
// restored once the service restart has completed.
func RefreshWinsService() error {
	winSrv, exists, err := OpenRancherWinsService()
	if err != nil {
		logrus.Errorf("Cannot restart %s as the service failed to open: %v", defaults.WindowsServiceName, err)
		return fmt.Errorf("failed to refresh the %s service: %w", winSrv.Name, err)
	}

	if !exists {
		logrus.Errorf("Cannot restart %s as the service does not exist", defaults.WindowsServiceName)
		return nil
	}

	defer winSrv.Close()

	// We cannot restart a service which another service depends on.
	// In the event that we need to update the rancher-wins config file
	// (and thus restart the rancher-wins service),
	// we will need to temporarily remove the service dependency from
	// the rke2 service if it exists. This ensures that rancher-wins can be updated
	// without potentially impacting node functionality due to a restart of rke2.

	rke2Srv, rke2Exists, err := OpenRKE2Service()
	if err != nil {
		logrus.Errorf("error opening rke2 service while restarting rancher-wins: %v", err)
	}

	depRemoved := false
	if rke2Exists {
		hasDep, err := rke2Srv.HasRancherWinsServiceDependency()
		if err != nil {
			return fmt.Errorf("error encountered while temporarily removing rke2 service dependency: %w", err)
		}
		if hasDep {
			logrus.Info("Temporarily removing rke2 service dependency")
			depRemoved = true
			err = rke2Srv.RemoveRancherWinsServiceDependency()
			rke2Srv.Close()
			if err != nil {
				return fmt.Errorf("error encountered while temporarily removing rke2 service dependency: %w", err)
			}
		}
	}

	err = winSrv.Restart()
	if err != nil {
		return fmt.Errorf("failed to restart the %s service: %w", winSrv.Name, err)
	}

	if depRemoved {
		// if we removed the dependency then we know the service exists
		rke2Srv, _, err = OpenRKE2Service()
		if err != nil {
			logrus.Errorf("error opening rke2 service while restarting rancher-wins: %v", err)
		}
		logrus.Info("Restoring rke2 service dependency")
		err = rke2Srv.AddRancherWinsServiceDependency()
		rke2Srv.Close()
		if err != nil {
			return fmt.Errorf("error encountered while restoring rke2 service dependency: %w", err)
		}
	}

	return nil
}
