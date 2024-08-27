package pkg

import (
	"errors"
	"fmt"

	"github.com/rancher/wins/suc/pkg/rancher"
	"github.com/rancher/wins/suc/pkg/service"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func Run(_ *cli.Context) error {
	var errs []error
	refreshService := false

	initialState, err := service.BuildInitialState()
	if err != nil {
		return fmt.Errorf("could not build initial state for rancher-wins: %v", err)
	}

	logrus.Infof("Updating rancher connection info")
	o, err := rancher.UpdateConnectionInformation()
	if err != nil {
		logrus.Errorf("Could not update rancher connection information. Script output:")
		logrus.Error(o)
		return fmt.Errorf("error encountered while refreshing connection information: %v", err)
	}

	logrus.Infof("successfully updated connection info")
	if o != "" {
		logrus.Infof(" Script output:\n%s", o)
	}

	// update the config using env vars
	restartServiceDueToConfigChange, err := service.UpdateConfigFromEnvVars("")
	refreshService = restartServiceDueToConfigChange
	if err != nil {
		errs = append(errs, err)
	}

	if refreshService && (errs == nil || len(errs) == 0) {
		if err = service.RefreshWinsService(); err != nil {
			return fmt.Errorf("error encountered while attempting to restart rancher-wins: %v", err)
		}
	} else if errs != nil && len(errs) > 0 {
		logrus.Errorf("Attempting to restore initial state due to error(s) encountered while updating rancher-wins: %v", combineErrors(errs))
		err = service.RestoreInitialState(initialState)
		if err != nil {
			errs = append(errs, err)
			return fmt.Errorf("failed to restore initial state: %v", combineErrors(errs))
		}
	}

	return combineErrors(errs)
}

func combineErrors(errs []error) error {
	var err error
	for _, e := range errs {
		err = errors.Join(err, e)
	}
	return err
}
