package host

import (
	"fmt"
	"os"
	"strings"

	"github.com/rancher/wins/pkg/defaults"
	"github.com/rancher/wins/suc/pkg/service"
	"github.com/sirupsen/logrus"
)

// UpgradeRancherWinsBinary will attempt to upgrade the wins.exe binary installed on the host.
// The version to be installed is embedded within the SUC binary, located in the winsBinary variable.
// Upgrades will only be attempted if the CATTLE_WINS_SKIP_BINARY_UPGRADE environment variable is not set to 'true' or '$true',
// and the currently installed version differs from the one embedded (determined by the output of 'wins.exe --version').
// During an upgrade attempt the rancher-wins service will be temporarily stopped.
// A boolean is returned to indicate if the rancher-wins service needs to be restarted due to a successful upgrade.
func UpgradeRancherWinsBinary() (bool, error) {
	if strings.ToLower(os.Getenv(skipBinaryUpgradeEnvVar)) == "true" ||
		strings.ToLower(os.Getenv(skipBinaryUpgradeEnvVar)) == "$true" {
		logrus.Warnf("environment variable '%s' was set to true, will not attempt to upgrade binary", skipBinaryUpgradeEnvVar)
		return false, nil
	}

	// we use the AppVersion set during compilation to indicate
	// the version of wins.exe that is packaged in the SUC binary.
	// See magetools/gotool.go for more information.
	desiredVersion := defaults.AppVersion

	// We should never install a dirty version of wins.exe onto a host.
	if strings.Contains(desiredVersion, "-dirty") {
		return false, fmt.Errorf("will not attempt to upgrade wins.exe version, refusing to install embedded dirty version (version: %s)", desiredVersion)
	}

	currentVersion, err := getRancherWinsVersionFromBinary(defaultWinsPath)
	if err != nil {
		return false, fmt.Errorf("could not determine current wins.exe version: %w", err)
	}

	if currentVersion == desiredVersion {
		logrus.Debugf("wins.exe is up to date (%s)", currentVersion)
		return false, nil
	}

	restartService, upgradeErr := updateBinaries(desiredVersion)
	if upgradeErr != nil {
		return false, upgradeErr
	}

	return restartService, nil
}

// updateBinaries writes the embedded binary onto the disk in the rancher-wins config directory (c:\etc\rancher\wins, by default).
// Once written, the binary is invoked to ensure that it is not corrupted and is running the expected version.
// After confirming the version, the updated binary is moved into the wins.exe binary directory ('c:\usr\local\bin', by default)
// and 'c:\Windows' directories. Once the upgraded binary has been moved into place, it is invoked once again
// to confirm the file was copied correctly.
func updateBinaries(desiredVersion string) (bool, error) {
	logrus.Info("Writing updated wins.exe to disk")
	// write the embedded binary to disk
	updatedBinaryPath := fmt.Sprintf("%s/wins-%s.exe", getWinsConfigDir(), strings.Trim(desiredVersion, "\n"))
	err := os.WriteFile(updatedBinaryPath, winsBinary, os.ModePerm)
	if err != nil {
		return false, err
	}

	// confirm that the new binary works and returns the version that we expect
	err = confirmWinsBinaryVersion(desiredVersion, updatedBinaryPath)
	if err != nil {
		return false, fmt.Errorf("failed to stage updated binary: %w", err)
	}

	logrus.Info("Stopping rancher-wins...")
	rw, _, err := service.OpenRancherWinsService()
	if err != nil {
		return false, fmt.Errorf("failed to open rancher-wins service while attempting to upgrade binary: %w", err)
	}

	// The service needs to be stopped before we can modify
	// the binary it uses
	err = rw.Stop()
	if err != nil {
		return false, fmt.Errorf("failed to stop rancher-wins service while attempting to upgrade binary: %w", err)
	}

	logrus.Infof("Copying %s to %s", updatedBinaryPath, defaultWinsPath)
	err = copyFile(updatedBinaryPath, defaultWinsPath)
	if err != nil {
		return false, fmt.Errorf("failed to copy new wins.exe binary to %s: %w", defaultWinsPath, err)
	}

	// While the rancher-wins service looks for wins.exe in c:\Windows
	// for consistency’s sake we should also ensure it's updated in c:\usr\local\bin
	// as the install script places it there as well
	usrLocalBinPath := getWinsUsrLocalBinBinary()

	logrus.Infof("Copying %s to %s", updatedBinaryPath, usrLocalBinPath)
	err = copyFile(updatedBinaryPath, usrLocalBinPath)
	if err != nil {
		return false, fmt.Errorf("failed to copy new wins.exe binary to %s: %w", usrLocalBinPath, err)
	}

	logrus.Infof("Validating updated binaries...")
	err = confirmWinsBinaryVersion(desiredVersion, defaultWinsPath)
	if err != nil {
		return false, err
	}

	err = confirmWinsBinaryVersion(desiredVersion, getWinsUsrLocalBinBinary())
	if err != nil {
		return false, err
	}

	logrus.Infof("Removing %s", updatedBinaryPath)
	err = os.Remove(updatedBinaryPath)
	if err != nil {
		return false, fmt.Errorf("failed to remove temporary wins.exe binary (%s): %w", updatedBinaryPath, err)
	}

	logrus.Infof("Successfully upgraded wins.exe to version %s", desiredVersion)
	return true, nil
}
