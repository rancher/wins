package service

import (
	"fmt"

	"github.com/rancher/wins/pkg/defaults"
	"github.com/sirupsen/logrus"
)

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
