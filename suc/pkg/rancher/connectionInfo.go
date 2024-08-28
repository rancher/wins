package rancher

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"fmt"
	"os"
	"os/exec"
)

const connInfoScriptName = "/hpc/update-connection-info.ps1"

func UpdateConnectionInformation() (string, error) {
	_, err := os.Stat(connInfoScriptName)
	if errors.Is(err, os.ErrNotExist) {
		logrus.Errorf("Could not find %s, will not attempt to update Rancher connection information.", connInfoScriptName)
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("failed to open %s: %w", connInfoScriptName, err)
	}

	// This command is expected to be run in a host process pod. Files packaged into
	// containers which are run as host process pods will be accessible from the `/hpc` directory.
	cmd := exec.Command("powershell", "-File", connInfoScriptName)
	o, err := cmd.CombinedOutput()
	if err != nil {
		return string(o), fmt.Errorf("failed to update connection info: %w", err)
	}
	return string(o), nil
}
