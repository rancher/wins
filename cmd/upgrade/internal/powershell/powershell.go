package powershell

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Sourced from https://github.com/flannel-io/flannel/blob/d31b0dc85a5a15bda5e606acbbbb9f7089441a87/pkg/powershell/powershell.go

//commandWrapper ensures that exceptions are written to stdout and the powershell process exit code is -1
const commandWrapper = `$ErrorActionPreference="Stop";try { %s } catch { Write-Host $_; os.Exit(-1) }`

// RunCommand executes a given powershell command.
//
// When the command throws a powershell exception, RunCommand will return the exception message as error.
func RunCommand(command string) ([]byte, error) {
	cmd := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", fmt.Sprintf(commandWrapper, command))

	stdout, err := cmd.Output()
	if err != nil {
		if cmd.ProcessState.ExitCode() != 0 {
			message := strings.TrimSpace(string(stdout))
			return []byte{}, errors.New(message)
		}

		return []byte{}, err
	}

	return stdout, nil
}

// RunCommandf executes a given powershell command. Command argument formats according to a format specifier (See fmt.Sprintf).
//
// When the command throws a powershell exception, RunCommandf will return the exception message as error.
func RunCommandf(command string, a ...interface{}) ([]byte, error) {
	return RunCommand(fmt.Sprintf(command, a...))
}

// RunCommandWithJSONResult executes a given powershell command.
// The command will be wrapped with ConvertTo-Json.
//
// You can Wrap your command with @(<cmd>) to ensure that the returned json is an array
//
// When the command throws a powershell exception, RunCommandf will return the exception message as error.
func RunCommandWithJSONResult(command string, v interface{}) error {
	wrappedCommand := fmt.Sprintf(commandWrapper, "ConvertTo-Json (%s)")
	wrappedCommand = fmt.Sprintf(wrappedCommand, command)

	stdout, err := RunCommandf(wrappedCommand)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(stdout, v); err != nil {
		return err
	}

	return nil
}
