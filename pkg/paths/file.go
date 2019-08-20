package paths

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

func GetFileSHA1Hash(path string) (string, error) {
	h := sha1.New()

	fs, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fs.Close()

	s, err := fs.Stat()
	if err != nil {
		return "", err
	}
	if s.IsDir() {
		return "", errors.Errorf("%s is not a file", path)
	}

	_, err = io.Copy(h, fs)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func MoveFile(srcPath, targetPath string) error {
	dir := filepath.Dir(targetPath)
	d, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrapf(err, "could not detect the directory of target file")
		}

		err = os.Mkdir(dir, os.ModePerm)
		if err != nil {
			return errors.Wrapf(err, "could not create the directory of target file")
		}
	} else if !d.IsDir() {
		return errors.Errorf("%s is not a directory", dir)
	}

	// if target path is already existing
	if _, err := os.Stat(targetPath); err == nil {
		// don't remove binary directly
		tempTargetPath := filepath.Join(os.TempDir(), filepath.Base(targetPath))
		err = os.Rename(targetPath, tempTargetPath)
		if err != nil {
			return errors.Wrapf(err, "could not backup the existing target file")
		}
	}

	err = os.Rename(srcPath, targetPath)
	if err != nil {
		return errors.Wrapf(err, "could not move the source file to the target")
	}

	return nil
}

type WatchHandle func(watchErr error, watchEvent fsnotify.Event)

func Watch(ctx context.Context, path string, handle WatchHandle) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Wrap(err, "could not start fsnotify watcher")
	}

	go func() {
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if handle != nil {
					handle(nil, event)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				if handle != nil {
					handle(err, fsnotify.Event{})
				}
			}
		}
	}()

	err = watcher.Add(path)
	if err != nil {
		return errors.Wrapf(err, "could not add watching for %s", path)
	}

	return nil
}
