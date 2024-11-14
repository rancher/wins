package host

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	defaultWinsPath                    = "c:\\Windows\\wins.exe"
	defaultWinsUsrLocalBinPath         = "c:\\usr\\local\\bin\\wins.exe"
	defaultConfigDir                   = "c:\\etc\\rancher\\wins"
	fileOperationAttempts              = 5
	fileOperationAttemptDelayInSeconds = 5

	// skipBinaryUpgradeEnvVar prevents the suc image from attempting to upgrade the wins binary.
	// This is primarily used in CI, to allow for test cases to run without having to completely
	// install rancher-wins.
	skipBinaryUpgradeEnvVar = "CATTLE_WINS_SKIP_BINARY_UPGRADE"
)

// getRancherWinsVersionFromBinary executes the wins.exe binary located at 'path' and passes the '--version'
// flag. The release version or commit hash is returned. If the binary returns unexpected output,
// was built with a dirty commit, or does not exist, an error will be returned.
func getRancherWinsVersionFromBinary(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("must specify a path")
	}

	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("provided path (%s) does not exist", path)
		}
		return "", fmt.Errorf("encoutered error stat'ing '%s': %w", path, err)
	}

	out, err := exec.Command(path, "--version").CombinedOutput()
	if err != nil {
		logrus.Errorf("could not invoke '%s --version' to determine installed wins.exe version: %v", path, err)
		return "", fmt.Errorf("failed to invoke '%s --version': %w", path, err)
	}

	logrus.Debugf("'%s --version' output: %s", path, string(out))
	return parseWinsVersion(string(out))
}

func confirmWinsBinaryVersion(desiredVersion string, path string) error {
	installedVersion, err := getRancherWinsVersionFromBinary(path)
	if err != nil {
		return fmt.Errorf("failed to confirm '%s' version: %w", path, err)
	}

	if installedVersion == desiredVersion {
		logrus.Debugf("'%s' returned expected version (%s)", path, desiredVersion)
		return nil
	}

	return fmt.Errorf("'%s' version ('%s') did not match desired version ('%s')", path, installedVersion, desiredVersion)
}

func parseWinsVersion(winsOutput string) (string, error) {
	// Expected output format is 'rancher-wins version v0.x.y[-rc.z]'"
	// A dirty binary will return 'rancher-wins version COMMIT-dirty'
	// A non-tagged version will return 'rancher-wins version COMMIT'
	s := strings.Split(winsOutput, " ")
	if len(s) != 3 {
		return "", fmt.Errorf("'wins.exe --version' did not return expected output length ('%v' was returned)", s)
	}

	verString := strings.Trim(s[2], "\n")
	// We should error out if the binary we're working with is dirty, but
	// if it's simply untagged we should proceed with the upgrade.
	if strings.Contains(verString, "dirty") {
		return "", fmt.Errorf("wins.exe binary returned a dirty version (%s)", verString)
	}

	return verString, nil
}

// copyFile opens the file located at 'source' and creates a new file at 'destination'
// with the same contents. In the event that the 'source' or 'destination' file is being used,
// copyFile will reattempt the operation 5 times over the course of 25 seconds. If the file still cannot
// be moved, an error will be returned. This behavior is beneficial when handling binaries
// that are referenced by services, as the underlying binary used by a service may continue to run
// for a brief time after the service has processed the stop signal.
//
// Note that permission bits on Windows do not function in the same
// way as Linux, the owner bit is always copied to all other bits. The caller of copyFile must
// ensure that the destination is covered by appropriate access control lists.
func copyFile(source, dest string) error {
	var err error
	var b []byte

	_, err = os.Stat(source)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("specified source file '%s' cannot be copied as it does not exist: %w", source, err)
		}
		return fmt.Errorf("failed to stat source file '%s': %w", source, err)
	}

	for i := 0; i < fileOperationAttempts; i++ {
		b, err = os.ReadFile(source)
		if err != nil {
			if strings.Contains(err.Error(), "because it is being used by another process") {
				logrus.Debugf("file copy attempt failed as the source file is in use, waiting %d seconds before reattempting", fileOperationAttemptDelayInSeconds)
				time.Sleep(fileOperationAttemptDelayInSeconds * time.Second)
				continue
			}
			return fmt.Errorf("failed to read from '%s': %w", source, err)
		}

		err = os.WriteFile(dest, b, os.ModePerm)
		if err != nil {
			if strings.Contains(err.Error(), "because it is being used by another process") {
				logrus.Debugf("file copy attempt failed as the destination file is in use, waiting %d seconds before reattempting", fileOperationAttemptDelayInSeconds)
				time.Sleep(fileOperationAttemptDelayInSeconds * time.Second)
				continue
			}
			return fmt.Errorf("failed to write to '%s': %w", dest, err)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to copy '%s' to '%s': %w", source, dest, err)
	}

	return nil
}

func getWinsConfigDir() string {
	customPath := os.Getenv("CATTLE_AGENT_CONFIG_DIR")
	if customPath != "" {
		return customPath
	}
	return defaultConfigDir
}

func getWinsUsrLocalBinBinary() string {
	customPath := os.Getenv("CATTLE_AGENT_BIN_PREFIX")
	if customPath != "" {
		return fmt.Sprintf("%s\\bin\\wins.exe", customPath)
	}
	return defaultWinsUsrLocalBinPath
}
