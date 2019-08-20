package validation_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rancher/wins/pkg/converters"
	"golang.org/x/sys/windows/registry"
)

/**
# Background

Based on https://github.com/golang/go/pull/33311, when windows go application catch the following event, go runtime will
treat as the tail signal:

	case CTRL_C_EVENT:        ->  Ctrl+C                       \
	case CTRL_BREAK_EVENT:    ->  Ctrl+Break                    -> syscall.SIGINT
	case CTRL_CLOSE_EVENT:    ->  Closing the console window    \
	case CTRL_LOGOFF_EVENT:   ->  User logs off                \ \
	case CTRL_SHUTDOWN_EVENT: ->  System is shutting down       ->-> syscall.SIGTERM

# Related
- rebase https://github.com/golang/go/pull/33311 into go sources
- guide by https://github.com/moby/moby/issues/25982
*/

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randString() string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, 6)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func writeFile(filePath string, fileContent string) {
	f, err := os.Create(filePath)
	if err != nil {
		consoleFatal(fmt.Sprintf("Failed to create %v: %v", filePath, err))
	}
	defer f.Close()

	_, err = io.Copy(f, strings.NewReader(fileContent))
	if err != nil {
		consoleFatal(fmt.Sprintf("Failed to write %v: %v", filePath, err))
	}
}

func compileExe(goFilePath string, exeFilePath string) {
	cmd := exec.Command("go.exe", "build", "-ldflags", "-s -w -extldflags \"-static\"", "-o", exeFilePath, goFilePath)
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=windows",
		"GOARCH=amd64",
	)
	o, err := cmd.CombinedOutput()
	if err != nil {
		consoleFatal(fmt.Sprintf("Failed to go compile %v: %v", string(o), err))
	}
}

func packageDockerContainer(exeFilePath string, dockerImageName string) {
	dockerfileContent := `
FROM mcr.microsoft.com/windows/servercore:SERVER_VERSION
COPY TEST_APP /Windows/testapp.exe
CMD testapp.exe
`

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		consoleFatal(fmt.Sprintf("Failed to open registry key: %v", err))
	}
	serverVersion := converters.GetStringFromRegistryKey(k, "ReleaseId")
	k.Close()

	dockerfileContent = strings.Replace(dockerfileContent, "SERVER_VERSION", serverVersion, -1)
	dockerfileContent = strings.Replace(dockerfileContent, "TEST_APP", filepath.Base(exeFilePath), -1)

	// write the dockerfile
	dockerfileFilePath := filepath.Join(filepath.Dir(exeFilePath), "Dockerfile."+dockerImageName)
	writeFile(dockerfileFilePath, dockerfileContent)

	// package image
	dockerBuildCmd := exec.Command("docker.exe", "build", "-f", dockerfileFilePath, "-t", dockerImageName, ".")
	dockerBuildCmd.Dir = filepath.Dir(exeFilePath)
	if o, err := dockerBuildCmd.CombinedOutput(); err != nil {
		consoleFatal(fmt.Sprintf("Failed to docker build %v: %v", string(o), err))
	}
}

var _ = Describe("signals", func() {
	var (
		err         error
		testDirPath string
	)

	// prepare temp dir
	BeforeEach(func() {
		testDirPath, err = ioutil.TempDir("", "signals")
		if err != nil {
			consoleFatal(fmt.Sprintf("TempDir failed: %v", err))
		}
	})

	// clean temp dir
	AfterEach(func() {
		os.RemoveAll(testDirPath)
	})

	Context("catch signal", func() {
		var testExeFilePath string

		// compile execution binary
		BeforeEach(func() {
			var goContent = `
package main

import (
    "fmt"
    "os"
    "os/signal"
)

func main() {
    fmt.Println("Starting")
    c := make(chan os.Signal, 1)
    signal.Notify(c)
    s := <-c
    fmt.Printf("Got signal: %s", s)
}

`
			name := filepath.Join(testDirPath, randString())

			// write the content to go
			testGoFilePath := fmt.Sprintf("%s.go", name)
			writeFile(testGoFilePath, goContent)

			// compile the go file
			testExeFilePath = fmt.Sprintf("%s.exe", name)
			compileExe(testGoFilePath, testExeFilePath)
		})

		It("SIGINT", func() {
			// syscall
			modkernel32 := syscall.NewLazyDLL("kernel32.dll")
			procGenerateConsoleCtrlEvent := modkernel32.NewProc("GenerateConsoleCtrlEvent")

			// run the exe
			testExeCmd := exec.Command(testExeFilePath)
			testExeOutput := &bytes.Buffer{}
			testExeCmd.Stdout = testExeOutput
			testExeCmd.Stderr = testExeOutput
			// https://docs.microsoft.com/en-us/windows/console/handlerroutine
			testExeCmd.SysProcAttr = &syscall.SysProcAttr{
				CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
			}
			if err = testExeCmd.Start(); err != nil {
				consoleFatal(fmt.Sprintf("Failed to start %v: %v", testExeFilePath, err))
			}

			// trigger by event
			go func() {
				time.Sleep(1 * time.Second)
				// https://docs.microsoft.com/en-us/windows/console/generateconsolectrlevent
				r1, _, e := procGenerateConsoleCtrlEvent.Call(syscall.CTRL_BREAK_EVENT, uintptr(testExeCmd.Process.Pid))
				if r1 == 0 {
					consoleError(fmt.Sprintf("Failed to call GenerateConsoleCtrlEvent: %v", e))
				}
			}()

			// wait
			if err = testExeCmd.Wait(); err != nil {
				consoleFatal(fmt.Sprintf("Failed to exec, %v: %v", testExeOutput.String(), err))
			}

			consoleInfo(testExeOutput.String())
			Expect(testExeOutput.String()).Should(ContainSubstring("Got signal: interrupt")) // syscall.SIGINT
		})

		// support by https://github.com/golang/go/pull/33311
		Context("SIGTERM", func() {
			var (
				imageName     = "test-sigterm"
				containerName string
				ctx           context.Context
				cancel        context.CancelFunc
				startContainerOutput *bytes.Buffer
			)

			BeforeEach(func() {
				containerName = randString()
				ctx, cancel = context.WithCancel(context.Background())

				// package image
				packageDockerContainer(testExeFilePath, imageName)

				// start container
				startContainerOutput = &bytes.Buffer{}
				go func() {
					defer cancel()

					startContainerCmd := exec.Command("docker.exe", "run", "-t", "--name", containerName, imageName)
					startContainerCmd.Stdout = startContainerOutput
					startContainerCmd.Stderr = startContainerOutput
					if err := startContainerCmd.Run(); err != nil {
						consoleError(fmt.Sprintf("Failed to start container: %v, %v", err, startContainerOutput.String()))
					}
				}()
				time.Sleep(1 * time.Second)
			})

			AfterEach(func() {
				// cleanup container
				exec.Command("docker.exe", "rm", "-f", containerName).CombinedOutput()
			})

			It("docker stop", func() {
				if o, err := exec.CommandContext(ctx, "docker.exe", "stop", containerName).CombinedOutput(); err != nil {
					consoleFatal(fmt.Sprintf("Failed to stop container: %v, %v", err, string(o)))
				}

				<-ctx.Done()

				consoleInfo(startContainerOutput.String())
				Expect(startContainerOutput.String()).Should(ContainSubstring("Got signal: terminated")) // syscall.SIGTERM
			})

		})

	})
})
