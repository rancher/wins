package rancher

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"fmt"
	"os"
	"os/exec"
)

func UpdateConnectionInformation() (string, error) {
	connInfoScriptName := "/hpc/update-connection-info.ps1"
	_, err := os.Stat(connInfoScriptName)
	if errors.Is(err, os.ErrNotExist) {
		logrus.Errorf("Could not find update-connection-info.ps1, will not attempt to update Rancher connection information.")
		return "", nil
	}

	// This command is expected to be run in a host process pods. Files packaged into
	// containers which are run as host process pods will be accessible from the `/hpc` directory.
	cmd := exec.Command("powershell", "-File", connInfoScriptName)
	o, err := cmd.CombinedOutput()
	if err != nil {
		return string(o), fmt.Errorf("failed to update connection info (%s): %v", string(o), err)
	}
	return string(o), nil
}
