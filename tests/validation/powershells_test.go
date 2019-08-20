package validation_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/rancher/wins/pkg/powershells"
)

func storeAsFile(content string) string {
	tempFile, _ := ioutil.TempFile(os.TempDir(), "")

	fileName := tempFile.Name()
	psScriptName := fmt.Sprintf("%s.ps1", fileName)

	err := ioutil.WriteFile(fileName, []byte(content), os.ModePerm)
	if err != nil {
		consoleFatal(err)
	}

	err = tempFile.Close()
	if err != nil {
		consoleFatal(err)
	}

	err = os.Rename(fileName, psScriptName)
	if err != nil {
		consoleFatal(err)
	}

	return psScriptName
}

var _ = Describe("powershells", func() {
	It("execute command directly", func() {
		var psOutput interface{}
		scriptStdout := func(output interface{}) {
			psOutput = output
			consoleInfo(output)
		}

		psb := &powershells.Builder{}
		ps, err := psb.StdOut(scriptStdout).Build()
		Expect(err).NotTo(HaveOccurred())

		psErr := ps.ExecuteCommand(context.TODO(), `[System.Console]::Out.Write("test-val")`)
		Expect(psErr).NotTo(HaveOccurred())
		Expect(psOutput).To(Equal("test-val"))
	})

	It("execute commands", func() {
		psb := &powershells.Builder{}
		ps, err := psb.StdOut(consoleInfo).StdErr(consoleError).Build()
		Expect(err).NotTo(HaveOccurred())

		psc, err := ps.Commands()
		Expect(err).NotTo(HaveOccurred())
		defer psc.Close()

		s, _, err := psc.Execute(context.TODO(), `[System.Console]::Out.Write("test-val")`)
		Expect(err).NotTo(HaveOccurred())
		Expect(s).To(Equal("test-val"))

		_, s, err = psc.Execute(context.TODO(), `[System.Console]::Error.Write("test-val")`)
		Expect(err).NotTo(HaveOccurred())
		Expect(s).To(Equal("test-val"))
	})

	Context("execute script", func() {
		var (
			psOutput interface{}
			ps       *powershells.PowerShell
			err      error
		)

		BeforeEach(func() {
			scriptStdout := func(output interface{}) {
				psOutput = output
				consoleInfo(output)
			}

			psb := &powershells.Builder{}
			ps, err = psb.StdOut(scriptStdout).StdErr(consoleError).Build()
			Expect(err).NotTo(HaveOccurred())
		})

		It("without args", func() {
			scriptContent := `
#Requires -Version 5.0

param ()

function print {
  [System.Console]::Out.Write($args[0])
  Start-Sleep -Milliseconds 100
}

function set-env-var {
  param(
      [parameter(Mandatory = $true)] [string]$Key,
      [parameter(Mandatory = $false)] [string]$Value = ""
  )

  [Environment]::SetEnvironmentVariable($Key, $Value, [EnvironmentVariableTarget]::Process)
}

function get-env-var {
  param(
      [parameter(Mandatory = $true)] [string]$Key
  )

  return [Environment]::GetEnvironmentVariable($Key, [EnvironmentVariableTarget]::Process)
}

set-env-var -Key test-key1 -Value test-val1
set-env-var -Key test-key2 -Value test-val

$val2 = get-env-var -Key test-key2

print $val2
`
			scriptPath := storeAsFile(scriptContent)
			defer func() {
				os.Remove(scriptPath)
			}()
			err := ps.ExecuteScript(context.TODO(), scriptPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(psOutput).To(Equal("test-val"))
		})

		It("with args", func() {
			scriptContent := `
#Requires -Version 5.0

param (
  [parameter(Mandatory = $true)] [string]$Val
)

function print {
  [System.Console]::Out.Write($args[0])
  Start-Sleep -Milliseconds 100
}

print $Val
`
			scriptPath := storeAsFile(scriptContent)
			defer func() {
				os.Remove(scriptPath)
			}()
			err := ps.ExecuteScript(context.TODO(), scriptPath, "-Val", "test-val")
			Expect(err).NotTo(HaveOccurred())

			Expect(psOutput).To(Equal("test-val"))
		})
	})
})
