//go:build mage

package main

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/rancher/wins/magetools"
)

var Default = BuildAll
var g *magetools.Go
var version string
var commit string
var artifactOutput = filepath.Join("artifacts")
var integrationBin = filepath.Join("tests/integration/bin")

const requiredFilesCount = 4

func Clean() error {
	if err := sh.Rm(artifactOutput); err != nil {
		return err
	}
	return sh.Rm("bin")
}

func Version() error {
	c, err := magetools.GetCommit()
	if err != nil {
		return err
	}
	commit = c

	ght := os.Getenv("TAG")
	isClean, err := magetools.IsGitClean()
	if err != nil {
		return err
	}
	if ght != "" && isClean {
		version = ght
		return nil
	}

	tag, err := magetools.GetLatestTag()
	if err != nil {
		return err
	}
	if tag != "" && isClean {
		version = tag
		return nil
	}

	version = commit
	if !isClean {
		version = commit + "-dirty"
		log.Printf("[Version] dirty version encountered: %s \n", version)
	}
	// check if this is a release version and fail if the version contains `dirty`
	if strings.Contains(version, "dirty") && ght != "" || tag != "" {
		return fmt.Errorf("[Version] releases require a non-dirty tag: %s", version)
	}
	log.Printf("[Version] version: %s \n", version)

	return nil
}

func Setup() {
	mg.Deps(Version)
	g = magetools.NewGo("amd64", "windows", version, commit, "0", "1")
}

func Dependencies() error {
	mg.Deps(Setup)
	return g.Mod("download")
}

func Validate() error {
	envs := map[string]string{"GOOS": "windows", "ARCH": "amd64", "CGO_ENABLED": "0", "MAGEFILE_VERBOSE": "1"}

	log.Printf("[Validate] Running: golangci-lint \n")
	if err := sh.RunWithV(envs, "golangci-lint", "run", "--timeout", "10m"); err != nil {
		return err
	}

	log.Printf("[Validate] Running: go fmt \n")
	if err := sh.RunWithV(envs, "go", "fmt", "./..."); err != nil {
		return err
	}

	log.Printf("validate has completed successfully \n")
	return nil
}

func BuildAll() error {
	mg.SerialDeps(Build, BuildSUC, Validate)
	return nil
}

func Build() error {
	mg.Deps(Clean, Dependencies)
	winsOutput := filepath.Join("bin", "wins.exe")

	log.Printf("[Build] Building wins version: %s \n", version)
	log.Printf("[Build] Output: %s \n", winsOutput)
	if err := g.Build(flags, "cmd/main.go", winsOutput); err != nil {
		return err
	}
	log.Printf("[Build] successfully built wins version %s \n", version)

	log.Printf("[Build] now staging build artifacts \n")
	if err := os.MkdirAll(artifactOutput, os.ModePerm); err != nil {
		return err
	}

	if err := sh.Copy(filepath.Join(artifactOutput, "install.ps1"), "install.ps1"); err != nil {
		return err
	}

	if err := sh.Copy(filepath.Join(artifactOutput, "wins.exe"), winsOutput); err != nil {
		return err
	}

	// create sha256 and sha512 artifacts for wins.exe
	exe, err := os.Open(filepath.Join(artifactOutput, "wins.exe"))
	if err != nil {
		return err
	}

	h256 := sha256.New()
	h512 := sha512.New()
	if _, err = io.Copy(h256, exe); err != nil {
		return err
	}

	if _, err = io.Copy(h512, exe); err != nil {
		return err
	}

	if err = os.WriteFile(filepath.Join(artifactOutput, "sha256.txt"), h256.Sum(nil), os.ModePerm); err != nil {
		return err
	}

	if err = os.WriteFile(filepath.Join(artifactOutput, "sha512.txt"), h512.Sum(nil), os.ModePerm); err != nil {
		return err
	}

	log.Printf("[Build] all required build artifacts have been staged \n")
	files, err := os.ReadDir(artifactOutput)
	if err != nil {
		return err
	}

	if len(files) != requiredFilesCount {
		return fmt.Errorf("[Build] a required build artifact is missing, expected %d artifacts and only got %d, exiting now \n", requiredFilesCount, len(files))
	}

	var artifacts strings.Builder
	for _, file := range files {
		artifacts.WriteString(file.Name() + " ,")
	}

	log.Printf("[Build] artifacts copied: %s \n", artifacts.String())

	return nil
}

func BuildSUC() error {
	log.Printf("[Build] Building wins SUC version: %s \n", version)
	// move wins.exe into the suc package so that it can be embedded
	err := sh.Copy(filepath.Join("suc/pkg/host/wins.exe"), filepath.Join(artifactOutput, "wins.exe"))
	if err != nil {
		log.Printf("failed to copy wins.exe to suc/pkg/host")
		return err
	}
	winsSucOutput := filepath.Join("bin", "wins-suc.exe")
	if err := g.Build(flags, "suc/main.go", winsSucOutput); err != nil {
		return err
	}
	if err := sh.Copy(filepath.Join(artifactOutput, "wins-suc.exe"), winsSucOutput); err != nil {
		return err
	}
	return nil
}

func Test() error {
	mg.Deps(BuildAll)
	log.Printf("[Test] Testing wins version %s \n", version)
	if err := g.Test(flags, "./..."); err != nil {
		return err
	}
	log.Printf("[Test] successfully tested wins version %s \n", version)
	return nil
}

// Integration target must be run on a wins system
// with Containers feature / docker installed
func Integration() error {
	mg.Deps(BuildAll)
	log.Printf("[Integration] Starting Integration Test for wins version %s \n", version)

	// make sure the docker files have access to the exe
	if err := os.MkdirAll(integrationBin, os.ModePerm); err != nil {
		return err
	}

	// move exe to right location
	if err := sh.Copy(filepath.Join("tests", "integration", "docker", "wins.exe"), filepath.Join(artifactOutput, "wins.exe")); err != nil {
		return err
	}

	// run test suite
	if err := sh.RunV("powershell.exe", filepath.Join("tests/integration/integration_suite_test.ps1")); err != nil {
		return err
	}

	log.Printf("[Test] successfully ran integration tests on wins version %s \n", version)
	return nil
}

func TestAll() error {
	mg.SerialDeps(Test, Integration)
	return nil
}

func CI() {
	mg.Deps(TestAll)
}

func flags(version string, commit string) string {
	return fmt.Sprintf(`-s -w -X github.com/rancher/wins/pkg/defaults.AppVersion=%s -X github.com/rancher/wins/pkg/defaults.AppCommit=%s -extldflags "-static"`, version, commit)
}
