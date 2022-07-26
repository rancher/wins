//go:build mage

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/rancher/wins/magetools"
)

var Default = Build
var g *magetools.Go
var version string
var commit string
var artifactOutput = filepath.Join("artifacts")

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

	dt := os.Getenv("DRONE_TAG")
	isClean, err := magetools.IsGitClean()
	if err != nil {
		return err
	}
	if dt != "" && isClean {
		version = dt
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

	version = commit + "-dirty"
	return nil
}

func Setup() {
	mg.Deps(Version)
	g = magetools.NewGo("amd64", "windows", version, commit, "0")
}

func Dependencies() error {
	mg.Deps(Setup)
	return g.Mod("download")
}

func Validate() error {
	envs := map[string]string{"GOOS": "windows", "ARCH": "amd64", "CGO_ENABLED": "0"}

	log.Printf("[build] Running: golangci-lint \n")
	if err := sh.RunWithV(envs, "golangci-lint", "run"); err != nil {
		return err
	}

	log.Printf("[build] Running: go fmt \n")
	if err := sh.RunWithV(envs, "go", "fmt", "./..."); err != nil {
		return err
	}

	log.Printf("validate has completed successfully \n")
	return nil
}

func Build() error {
	mg.Deps(Clean, Dependencies, Validate)

	winsOutput := filepath.Join("bin", "wins.exe")

	log.Printf("[build] Building wins version %s \n", version)
	log.Printf("[build] Output: %s \n", winsOutput)
	if err := g.Build(flags, "cmd/main.go", winsOutput); err != nil {
		return err
	}
	log.Printf("[build] successfully built wins version version %s \n", version)

	log.Printf("[build] now staging build artifacts \n")
	if err := os.MkdirAll(artifactOutput, os.ModePerm); err != nil {
		return err
	}

	if err := sh.Copy(filepath.Join(artifactOutput, "install.ps1"), "install.ps1"); err != nil {
		return err
	}

	if err := sh.Copy(filepath.Join(artifactOutput, "wins.exe"), winsOutput); err != nil {
		return err
	}

	if err := sh.Copy(filepath.Join(artifactOutput, "run.ps1"), filepath.Join("suc", "run.ps1")); err != nil {
		return err
	}

	log.Printf("[build] all required build artifacts have been staged \n")
	files, err := os.ReadDir(artifactOutput)
	if err != nil {
		return err
	}

	if len(files) != 3 {
		return errors.New("[package] a required build artifact is missing, exiting now \n")
	}

	var artifacts strings.Builder
	for _, file := range files {
		artifacts.WriteString(file.Name() + " ,")
	}

	log.Printf("[build] artifacts copied: %s \n", artifacts.String())

	return nil
}

func Test() error {
	mg.Deps(Build)
	log.Printf("[build] Testing wins version %s \n", version)
	if err := g.Test(flags, "./..."); err != nil {
		return err
	}
	log.Printf("[build] successfully tested wins version version %s \n", version)
	return nil
}

func CI() {
	mg.Deps(Test)
}

func flags(version string, commit string) string {
	return fmt.Sprintf(`-s -w -X github.com/rancher/wins/pkg/defaults.AppVersion=%s -X github.com/rancher/wins/pkg/defaults.AppCommit=%s -extldflags "-static"`, version, commit)
}
