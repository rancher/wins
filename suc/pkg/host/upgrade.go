package host

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rancher/wins/pkg/csiproxy"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/rancher/wins/suc/pkg/service"
	sucConfig "github.com/rancher/wins/suc/pkg/service/config"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc/mgr"
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

	binaryExists, err := confirmWinsBinaryIsInstalled()
	if err != nil {
		return false, err
	}

	if binaryExists {
		currentVersion, err := getRancherWinsVersionFromBinary(defaultWinsPath)
		if err != nil {
			return false, fmt.Errorf("could not determine current wins.exe version: %w", err)
		}

		if currentVersion == desiredVersion {
			logrus.Debugf("wins.exe is up to date (%s)", currentVersion)
			return false, nil
		}
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
	rw, rwExists, err := service.OpenRancherWinsService()
	if err != nil {
		return false, fmt.Errorf("failed to open rancher-wins service while attempting to upgrade binary: %w", err)
	}

	if rwExists {
		// The service needs to be stopped before we can modify the binary it uses
		err = rw.Stop()
		if err != nil {
			return false, fmt.Errorf("failed to stop rancher-wins service while attempting to upgrade binary: %w", err)
		}
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

	err = confirmWinsBinaryVersion(desiredVersion, usrLocalBinPath)
	if err != nil {
		return false, err
	}

	logrus.Infof("Removing %s", updatedBinaryPath)
	err = os.Remove(updatedBinaryPath)
	if err != nil {
		return false, fmt.Errorf("failed to remove temporary wins.exe binary (%s): %w", updatedBinaryPath, err)
	}

	logrus.Infof("Successfully upgraded wins.exe to version %s", desiredVersion)
	return rwExists, nil
}

// UpgradeCSIProxyBinary will attempt to upgrade the csi-proxy binary installed on the host.
// The version to be installed is determined by the CATTLE_CSI_PROXY_VERSION environment variable and
// downloaded from the URL specified by the CATTLE_CSI_PROXY_URL environment variable.
// Upgrades will only be attempted if the CATTLE_SYNC_CSI_PROXY_BINARY environment variable is set to 'true' or '$true',
// and the currently installed version differs from the desired version.
// During an upgrade attempt the csiproxy service will be temporarily stopped.
// A boolean is returned to indicate if an upgrade was successfully performed.
func UpgradeCSIProxyBinary() (bool, error) {
	logrus.Infof("Starting UpgradeCSIProxyBinary process")

	// 1. check if CSIProxyBinary Needs to be updated.
	if strings.ToLower(os.Getenv(syncCSIProxyBinary)) != "true" &&
		strings.ToLower(os.Getenv(syncCSIProxyBinary)) != "$true" {
		logrus.Warnf("environment variable '%s' was not set to true, will not attempt to upgrade binary", syncCSIProxyBinary)
		return false, nil
	}
	if strings.ToLower(os.Getenv(csiProxyBinaryURL)) == "" {
		logrus.Warnf("environment variable '%s' was not set, will not attempt to upgrade binary", csiProxyBinaryURL)
		return false, nil
	}
	desiredVersion := os.Getenv(csiProxyVersion)
	if strings.ToLower(desiredVersion) == "" {
		logrus.Warnf("environment variable '%s' was not set, will not attempt to upgrade binary", csiProxyVersion)
		return false, nil
	}

	logrus.Infof("Environment variables validated. Sync: %s, URL: %s, Desired Version: %s", os.Getenv(syncCSIProxyBinary), os.Getenv(csiProxyBinaryURL), desiredVersion)

	// Retrieve the binary path by checking the csiproxy service details
	logrus.Infof("Connecting to service manager to retrieve csiproxy service details")
	m, err := mgr.Connect()
	if err != nil {
		return false, fmt.Errorf("could not connect to service manager: %w", err)
	}
	defer m.Disconnect()

	logrus.Infof("Opening csiproxy service")
	s, err := m.OpenService("csiproxy")
	if err != nil {
		return false, fmt.Errorf("could not open csiproxy service: %w", err)
	}
	defer s.Close()

	logrus.Infof("Retrieving csiproxy service configuration")
	svcConfig, err := s.Config()
	if err != nil {
		return false, fmt.Errorf("could not retrieve csiproxy service config: %w", err)
	}

	csiProxyBinPath := svcConfig.BinaryPathName

	if csiProxyBinPath == "" {
		return false, fmt.Errorf("could not determine csi-proxy binary path from service config")
	}

	// The BinaryPathName can include arguments (e.g., "C:\path\to\binary.exe" --arg).
	// We need to extract just the file path, removing any quotes or trailing arguments.
	if len(csiProxyBinPath) > 0 {
		csiProxyBinPath = strings.Split(csiProxyBinPath, " ")[0]
	}

	logrus.Infof("Retrieved csiproxy binary path from service config: %s", csiProxyBinPath)

	logrus.Infof("Checking if csi-proxy binary exists at path: %s", csiProxyBinPath)
	binaryExists, err := confirmPathExist(csiProxyBinPath)
	if err != nil {
		return false, fmt.Errorf("could not determine if installed csi-proxy binary exists: %v", err)
	}
	if !binaryExists {
		return false, fmt.Errorf("csi-proxy binary does not exist at path: %s", csiProxyBinPath)
	}

	// 2. read the config to check the current version of csi proxy.
	logrus.Infof("Getting current csi-proxy version from config")
	currentVersion, err := getCSIProxyVersion("")
	if err != nil {
		return false, fmt.Errorf("failed to get current csi-proxy version: %w", err)
	}
	logrus.Infof("Current csi-proxy version: %s", currentVersion)

	// 3. check if the csiProxyVersion drifts from the current version or not
	if currentVersion == desiredVersion {
		logrus.Infof("csi-proxy is up to date (%s), no upgrade required", currentVersion)
		return false, nil
	}

	logrus.Infof("csi-proxy version drift detected. Current: %s, Desired: %s", currentVersion, desiredVersion)

	err = updateCSIProxyBinaries(desiredVersion, os.Getenv(csiProxyBinaryURL), csiProxyBinPath)
	if err != nil {
		return false, fmt.Errorf("failed to update csi-proxy binaries: %w", err)
	}

	return true, nil
}

func updateCSIProxyBinaries(desiredVersion, urlTemplate, csiProxyBinPath string) error {
	logrus.Infof("Downloading updated csi-proxy.exe (version %s)", desiredVersion)

	logrus.Infof("Loading rancher-wins config to prepare for csi-proxy update")
	winsCfg, err := sucConfig.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load config for updating csi-proxy version: %w", err)
	}

	kubeletPath := "c:\\var\\lib\\kubelet\\kubelet.exe"
	if winsCfg.CSIProxy != nil && winsCfg.CSIProxy.KubeletPath != "" {
		kubeletPath = winsCfg.CSIProxy.KubeletPath
	}

	cfg := &csiproxy.Config{
		URL:         urlTemplate,
		Version:     desiredVersion,
		KubeletPath: kubeletPath,
	}

	logrus.Infof("Initializing csi-proxy download with URL: %s, Version: %s, KubeletPath: %s", urlTemplate, desiredVersion, kubeletPath)
	proxy, err := csiproxy.New(cfg, winsCfg.TLSConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize csi proxy instance: %w", err)
	}

	updatedCSIProxyPath := filepath.Join(getWinsConfigDir(), fmt.Sprintf("csi-proxy-%s-%d.exe", desiredVersion, time.Now().UnixMilli()))
	logrus.Infof("Downloading csi-proxy binary to temporary path: %s", updatedCSIProxyPath)
	if err := proxy.Download(updatedCSIProxyPath); err != nil {
		return fmt.Errorf("failed to download csi proxy: %w", err)
	}
	logrus.Infof("Successfully downloaded csi-proxy binary to %s", updatedCSIProxyPath)

	defer func() {
		if err := os.Remove(updatedCSIProxyPath); err != nil && !os.IsNotExist(err) {
			logrus.Warnf("failed to remove temporary csi-proxy binary (%s): %v", updatedCSIProxyPath, err)
		}
	}()

	logrus.Info("Stopping csiproxy service...")
	csiSvc, exists, err := service.Open("csiproxy")
	if err != nil {
		return fmt.Errorf("failed to open csiproxy service: %w", err)
	}
	if !exists {
		return fmt.Errorf("csiproxy service does not exist")
	}
	defer csiSvc.Close()

	err = csiSvc.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop csiproxy service: %w", err)
	}
	logrus.Infof("Successfully stopped csiproxy service")

	processName := filepath.Base(csiProxyBinPath)
	const (
		maxAttempts   = 10
		checkInterval = 5 * time.Second
	)
	totalWait := time.Duration(maxAttempts) * checkInterval

	logrus.Infof("Waiting up to %v for %s process to exit...", totalWait, processName)
	for attempts := 0; attempts < maxAttempts; attempts++ {
		out, _ := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", processName), "/NH").Output()
		if !strings.Contains(string(out), processName) {
			logrus.Infof("Process %s has exited", processName)
			break
		}
		if attempts == maxAttempts-1 {
			logrus.Warnf("Process %s did not exit within %v, proceeding anyway", processName, totalWait)
		}
		time.Sleep(checkInterval)
	}

	logrus.Infof("Copying %s to %s", updatedCSIProxyPath, csiProxyBinPath)
	err = copyFile(updatedCSIProxyPath, csiProxyBinPath)
	if err != nil {
		return fmt.Errorf("failed to replace csi-proxy binary: %w", err)
	}
	logrus.Infof("Successfully copied csi-proxy binary")

	logrus.Info("Starting csiproxy service...")
	err = csiSvc.Start()
	if err != nil {
		return fmt.Errorf("failed to start csiproxy service: %w", err)
	}
	logrus.Infof("Successfully started csiproxy service")

	logrus.Info("Updating rancher-wins config with new csi-proxy version...")
	if winsCfg.CSIProxy == nil {
		winsCfg.CSIProxy = &csiproxy.Config{}
	}
	winsCfg.CSIProxy.URL = urlTemplate
	winsCfg.CSIProxy.Version = desiredVersion
	winsCfg.CSIProxy.KubeletPath = kubeletPath

	err = sucConfig.SaveConfig(winsCfg, "")
	if err != nil {
		return fmt.Errorf("failed to save updated config: %w", err)
	}

	logrus.Infof("Successfully upgraded csi-proxy to version %s", desiredVersion)
	return nil
}
