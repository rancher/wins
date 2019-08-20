package powershells

import (
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"path/filepath"
)

type StdStream func(output interface{})

type IOFormat string
type WindowStyle string
type ExecutionPolicy string

const (
	// refer: https://docs.microsoft.com/en-us/powershell/module/Microsoft.PowerShell.Core/About/about_PowerShell_exe?view=powershell-5.1#-inputformat-text--xml
	IOFormatText IOFormat = "Text"
	IOFormatXML  IOFormat = "XML"

	// refer: https://docs.microsoft.com/en-us/powershell/module/Microsoft.PowerShell.Core/About/about_execution_policies?view=powershell-5.1#powershell-execution-policies
	ExecutionPolicyDefault      ExecutionPolicy = "Default"
	ExecutionPolicyAllSigned    ExecutionPolicy = "AllSigned"
	ExecutionPolicyBypass       ExecutionPolicy = "Bypass"
	ExecutionPolicyRemoteSigned ExecutionPolicy = "RemoteSigned"
	ExecutionPolicyRestricted   ExecutionPolicy = "Restricted"
	ExecutionPolicyUndefined    ExecutionPolicy = "Undefined"
	ExecutionPolicyUnrestricted ExecutionPolicy = "Unrestricted"
)

type Builder struct {
	sta               bool            // starts PowerShell using a single-threaded apartment.
	noProfile         bool            // does not load the PowerShell profile.
	inputFormat       IOFormat        // describes the format of data sent to PowerShell.
	outputFormat      IOFormat        // determines how output from PowerShell is formatted.
	configurationName string          // specifies a configuration endpoint in which PowerShell is run.
	executionPolicy   ExecutionPolicy // sets the default execution policy for the current session and saves it in the `$env:PSExecutionPolicyPreference` environment variable.
	stdout            StdStream
	stderr            StdStream
}

func (b *Builder) Sta() *Builder {
	b.sta = true
	return b
}

func (b *Builder) NoProfile() *Builder {
	b.noProfile = true
	return b
}

func (b *Builder) InputFormat(format IOFormat) *Builder {
	b.inputFormat = format
	return b
}

func (b *Builder) OutputFormat(format IOFormat) *Builder {
	b.outputFormat = format
	return b
}

func (b *Builder) ConfigurationName(name string) *Builder {
	b.configurationName = name
	return b
}

func (b *Builder) ExecutionPolicy(policy ExecutionPolicy) *Builder {
	b.executionPolicy = policy
	return b
}

func (b *Builder) StdOut(stream StdStream) *Builder {
	b.stdout = stream
	return b
}

func (b *Builder) StdErr(stream StdStream) *Builder {
	b.stderr = stream
	return b
}

func (b *Builder) Build() (*PowerShell, error) {
	// refer: https://docs.microsoft.com/en-us/powershell/module/Microsoft.PowerShell.Core/About/about_PowerShell_exe?view=powershell-5.1
	psPath, err := exec.LookPath("powershell.exe")
	if err != nil {
		// compatible with nanoserver, refer: https://docs.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_pwsh?view=powershell-6
		psPath, err = exec.LookPath("pwsh.exe")
	}
	if err != nil {
		return nil, errors.New("cannot get PowerShell executor")
	}

	psPath, err = filepath.Abs(psPath)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(psPath)
	if err != nil {
		return nil, err
	}
	if fi.Mode().IsDir() {
		return nil, errors.Errorf("%s is directory", psPath)
	}

	psArgs := make([]string, 0, 32)
	if b.sta {
		psArgs = append(psArgs, "-Sta")
	}
	if b.noProfile {
		psArgs = append(psArgs, "-NoProfile")
	}
	if len(b.inputFormat) != 0 {
		psArgs = append(psArgs, "-InputFormat", string(b.inputFormat))
	}
	if len(b.outputFormat) != 0 {
		psArgs = append(psArgs, "-OutputFormat", string(b.outputFormat))
	}
	if len(b.configurationName) != 0 {
		psArgs = append(psArgs, "-ConfigurationName", b.configurationName)
	}
	if len(b.executionPolicy) != 0 {
		psArgs = append(psArgs, "-ExecutionPolicy", string(b.executionPolicy))
	}

	ps := &PowerShell{
		path:   psPath,
		args:   psArgs,
		stderr: b.stderr,
		stdout: b.stdout,
	}

	return ps, nil
}

type PowerShell struct {
	path   string
	args   []string
	stdout StdStream
	stderr StdStream
}
