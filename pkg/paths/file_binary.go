package paths

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

func GetBinaryPath(binaryName string) (string, error) {
	// find service abs path
	p, err := exec.LookPath(binaryName)
	if err != nil {
		return "", err
	}
	p, err = filepath.Abs(p)
	if err != nil {
		return "", err
	}

	// detect service is file or not
	fi, err := os.Stat(p)
	if err == nil {
		if !fi.IsDir() {
			return p, nil
		}
		err = errors.Errorf("%s is directory", p)
	}
	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.IsDir() {
				return p, nil
			}
			return "", errors.Errorf("%s is directory", p)
		}
	}

	return "", err
}

func GetBinarySHA1Hash(binaryName string) (string, error) {
	path, err := GetBinaryPath(binaryName)
	if err != nil {
		return "", err
	}
	actualChecksum, err := GetFileSHA1Hash(path)
	if err != nil {
		return "", err
	}

	return actualChecksum, nil
}

func EnsureBinary(binaryName string, expectedChecksum string) error {
	actualChecksum, err := GetBinarySHA1Hash(binaryName)
	if err != nil {
		return errors.Wrapf(err, "could not get checksum")
	}
	if expectedChecksum != actualChecksum {
		return errors.Errorf("could not match (expect checksum %q, but get %q)", expectedChecksum, actualChecksum)
	}

	return nil
}
