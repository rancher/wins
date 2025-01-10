package pkg

import (
	"errors"
	"fmt"

	"github.com/rancher/wins/suc/pkg/rancher"
	"github.com/rancher/wins/suc/pkg/service"
	"github.com/rancher/wins/suc/pkg/service/config"
	"github.com/rancher/wins/suc/pkg/service/state"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func Run(_ *cli.Context) error {
	var errs []error
	initialState, err := state.BuildInitialState()
	if err != nil {
		return fmt.Errorf("could not build initial state for rancher-wins: %w", err)
	}

	logrus.Info("Updating rancher connection info")
	output, err := rancher.UpdateConnectionInformation()
	if err != nil {
		logrus.Errorf("Could not update rancher connection information")
		logrus.Errorf("Script output:\n%s", output)
		return fmt.Errorf("error encountered while refreshing connection information: %w", err)
	}

	if output != "" {
		logrus.Debugf("Script output:\n%s", output)
	}

	// update the config using env vars
	restartServiceDueToConfigChange, updateErr := config.UpdateConfigFromEnvVars()
	if updateErr != nil {
		errs = append(errs, updateErr)
	}

	if restartServiceDueToConfigChange {
		err = service.RefreshWinsService()
		if err != nil {
			errs = append(errs, fmt.Errorf("error encountered while attempting to restart rancher-wins: %w", err))
		}
	}

	if errs != nil && len(errs) > 0 {
		logrus.Errorf("Attempting to restore initial state due to error(s) encountered while updating rancher-wins: %v", errors.Join(errs...))
		err = state.RestoreInitialState(initialState)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to restore initial state: %w", err))
		} else {
			logrus.Info("Successfully restored initial config state")
		}
		return errors.Join(errs...)
	}

	return nil
}
