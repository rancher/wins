package service

import "github.com/rancher/wins/cmd/server/config"

// InitialState represents the configuration of
// rancher-wins and rke2 before any changes are made.
// In the event of an error during reconfiguration of the
// service or the related binaries, this struct should be used
// to roll back all changes. Once an InitialState struct is created
// (via BuildInitialState) it must not be updated.
type InitialState struct {
	InitialConfig *config.Config
}

// BuildInitialState retrieves the rancher-wins config file and any relevant service
// configuration settings and packages them into an InitialState struct. BuildInitialState
// must be called before any modifications are made to the host to ensure that all changes can be
// safely rolled back.
func BuildInitialState() (InitialState, error) {
	cfg, err := loadConfig("")
	if err != nil {
		return InitialState{}, err
	}

	return InitialState{
		InitialConfig: cfg,
	}, nil
}

// RestoreInitialState will clear all changes made to the host and reinstate the values set within InitialState.
func RestoreInitialState(state InitialState) error {
	return saveConfig(state.InitialConfig, "")
}
