package paths

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func IncludeFiles(path string, filenames ...string) (bool, error) {
	if len(filenames) == 0 {
		return false, errors.New("could not detect empty filename collection")
	}

	if err := EnsureDirectory(path); err != nil {
		return false, errors.Wrapf(err, "failed to ensure directory %s", path)
	}

	for _, f := range filenames {
		fpath := filepath.Join(path, f)
		fs, err := os.Stat(fpath)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, errors.Wrapf(err, "%s could not be touched", fpath)
		} else if fs.IsDir() {
			return false, errors.Errorf("%s is directory", fpath)
		}
	}

	return true, nil
}

func EnsureDirectory(dir string) error {
	d, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	} else if !d.IsDir() {
		return errors.New("it's not a directory")
	}

	return nil
}
