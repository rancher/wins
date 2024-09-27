package state

import (
	"encoding/json"
	"errors"
	"fmt"

	winsConfig "github.com/rancher/wins/cmd/server/config"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/rancher/wins/suc/pkg/service"
	sucConfig "github.com/rancher/wins/suc/pkg/service/config"
	"github.com/sirupsen/logrus"
)

// InitialState represents the configuration of
// rancher-wins and rke2 before any changes are made.
// In the event of an error during reconfiguration of the
// service or the related binaries, this struct should be used
// to roll back all changes. Once an InitialState struct is created
// (via BuildInitialState) it must not be updated.
type InitialState struct {
	InitialConfig        *winsConfig.Config
	InitialServiceConfig Configuration
}

type Configuration struct {
	winsDelayedStart bool
	rke2Dependencies []string
}

// BuildInitialState retrieves the rancher-wins config file and any relevant service
// configuration settings and packages them into an InitialState struct. BuildInitialState
// must be called before any modifications are made to the host to ensure that all changes can be
// safely rolled back.
func BuildInitialState() (InitialState, error) {
	logrus.Info("Building Initial State...")
	winsCfg, err := sucConfig.LoadConfig("")
	if err != nil {
		return InitialState{}, fmt.Errorf("could not open rancher-wins config while building initial state: %w", err)
	}

	winsSvc, winsExists, err := service.Open(defaults.WindowsServiceName)
	if err != nil {
		return InitialState{}, fmt.Errorf("could not open rancher-wins service while building initial state: %w", err)
	}

	if !winsExists {
		return InitialState{}, fmt.Errorf("the rancher-wins service does not exist")
	}
	defer winsSvc.Close()

	rke2Svc, rke2Exists, err := service.Open("rke2")
	if err != nil {
		return InitialState{}, fmt.Errorf("encountered error getting config file for %s service: %w", "rke2", err)
	}

	var rke2Deps []string
	if rke2Exists {
		rke2Deps = rke2Svc.Config.Dependencies
		rke2Svc.Close()
	} else {
		logrus.Warn("Could not find rke2 service while building initial state")
	}

	logrus.Debugf("rancher-wins delayed start is set to %t", winsSvc.Config.DelayedAutoStart)
	logrus.Debugf("rke2 service dependencies: %v", rke2Deps)
	j, _ := json.MarshalIndent(winsCfg, "", " ")
	logrus.Debugf("initial rancher-wins config file:\n%v", string(j))

	return InitialState{
		InitialConfig: winsCfg,
		InitialServiceConfig: Configuration{
			winsDelayedStart: winsSvc.Config.DelayedAutoStart,
			rke2Dependencies: rke2Deps,
		},
	}, nil
}

// RestoreInitialState will clear all changes made to the host and reinstate the values contained within InitialState.
func RestoreInitialState(state InitialState) error {
	var errs []error
	// restore rancher-wins service configuration
	winsSvc, _, err := service.Open(defaults.WindowsServiceName)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to open %s while restoring initial configuration: %w", defaults.WindowsServiceName, err))
	}

	saveWinsConfig := false
	if winsSvc.Config.DelayedAutoStart != state.InitialServiceConfig.winsDelayedStart {
		saveWinsConfig = true
		winsSvc.Config.DelayedAutoStart = state.InitialServiceConfig.winsDelayedStart
	}

	if saveWinsConfig {
		logrus.Infof("Restoring %s service config", defaults.WindowsServiceName)
		err = winsSvc.UpdateConfig()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to restore initial configuration of %s service: %w", defaults.WindowsServiceName, err))
		}
	}

	// restore rke2 service configuration
	saveRke2Config := false
	rke2, rke2Exists, err := service.Open("rke2")
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to open %s while restoring initial configuration: %w", "rke2", err))
	}

	if rke2Exists {
		logrus.Infof("Restoring rke2 service configuration")
		if !service.SlicesMatch(rke2.Config.Dependencies, state.InitialServiceConfig.rke2Dependencies) {
			saveRke2Config = true
			rke2.Config.Dependencies = state.InitialServiceConfig.rke2Dependencies
		}

		if saveRke2Config {
			err = rke2.UpdateConfig()
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to restore initial configuration of %s service: %w", "rke2", err))
			}
		}
	}

	// restore rancher-wins config file
	logrus.Infof("Restoring rancher-wins configuration file")
	err = sucConfig.SaveConfig(state.InitialConfig, "")
	if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
