package pkg

import (
	"fmt"

	"github.com/rancher/wins/suc/pkg/rancher"
	"github.com/rancher/wins/suc/pkg/service"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func Run(_ *cli.Context) error {
	initialState, err := service.BuildInitialState()
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

	logrus.Info("Successfully updated connection info")
	if output != "" {
		logrus.Infof("Script output:\n%s", output)
	}

	// update the config using env vars
	restartServiceDueToConfigChange, updateErr := service.UpdateConfigFromEnvVars()
	if updateErr != nil {
		logrus.Errorf("Attempting to restore initial state due to error encountered while updating rancher-wins: %v", updateErr)
		err = service.RestoreInitialState(initialState)
		if err != nil {
			return fmt.Errorf("failed to restore initial state: %w", err)
		}
		logrus.Info("Successfully restored initial config state")
		return fmt.Errorf("failed to update rancher-wins config file: %w", updateErr)
	}

	if restartServiceDueToConfigChange {
		if err = service.RefreshWinsService(); err != nil {
			return fmt.Errorf("error encountered while attempting to restart rancher-wins: %w", err)
		}
	}

	return nil
}
