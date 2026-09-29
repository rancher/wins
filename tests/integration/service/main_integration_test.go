//go:build windows && integration

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

// testServiceExe is the path to the test service binary built by TestMain.
var testServiceExe string

func TestMain(m *testing.M) {
	os.Exit(runIntegration(m))
}

func runIntegration(m *testing.M) int {
	if !windows.GetCurrentProcessToken().IsElevated() {
		fmt.Fprintln(os.Stderr, "service integration tests must be run from an elevated process")
		return 1
	}

	// Required to terminate lingering test service processes, which run as LocalSystem.
	if err := enableDebugPrivilege(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to enable SeDebugPrivilege: %v\n", err)
		return 1
	}

	dir, err := os.MkdirTemp("", "wins-service-it-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		return 1
	}
	defer func() {
		if err := killTestServiceProcesses(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to kill test service processes: %v\n", err)
		}
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintf(os.Stderr, "failed to remove %s: %v\n", dir, err)
		}
	}()

	testServiceExe = filepath.Join(dir, testServiceExeName)
	build := exec.Command("go", "build", "-o", testServiceExe, "github.com/rancher/wins/tests/integration/testservice")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build test service: %v\n", err)
		return 1
	}

	// Poll the SCM every second rather than every five, keeping the overall budget at one minute.
	for k, v := range map[string]string{
		"CATTLE_WINS_STATE_TRANSITION_SECONDS":  "1",
		"CATTLE_WINS_STATE_TRANSITION_ATTEMPTS": "60",
	} {
		if err := os.Setenv(k, v); err != nil {
			fmt.Fprintf(os.Stderr, "failed to set %s: %v\n", k, err)
			return 1
		}
	}

	if err := removeLeftoverServices(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to remove services left over from a previous run: %v\n", err)
		return 1
	}

	logrus.SetLevel(logrus.DebugLevel)

	return m.Run()
}

func enableDebugPrivilege() error {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token); err != nil {
		return fmt.Errorf("failed to open process token: %w", err)
	}
	defer token.Close()

	var luid windows.LUID
	if err := windows.LookupPrivilegeValue(nil, windows.StringToUTF16Ptr("SeDebugPrivilege"), &luid); err != nil {
		return fmt.Errorf("failed to look up SeDebugPrivilege: %w", err)
	}

	privileges := windows.Tokenprivileges{PrivilegeCount: 1}
	privileges.Privileges[0] = windows.LUIDAndAttributes{Luid: luid, Attributes: windows.SE_PRIVILEGE_ENABLED}
	if err := windows.AdjustTokenPrivileges(token, false, &privileges, 0, nil, nil); err != nil {
		return fmt.Errorf("failed to adjust token privileges: %w", err)
	}
	return nil
}
