package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/rancher/wins/cmd/server/config"
	"github.com/rancher/wins/pkg/paths"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
)

func setupUpgrading(ctx context.Context, cfg *config.Config) error {
	isWindowsService, err := svc.IsWindowsService()
	if err != nil {
		return errors.Wrap(err, "could not detect whether the current process is running as a Windows service")
	}
	if !isWindowsService {
		logrus.Warn("cannot upgrade unless the current process is being executed as a Windows service")
		return nil
	}

	// watching mode
	return enableWatchingMode(ctx, cfg)
}

// enableWatchingMode sets up a file system notification-based watcher on the config.Upgrade.WatchingPath directory.
// If a write is detected in to a file in that directory, the watcher computes whether the SHA1 hash of the binary
// within that path differs from the current binary that is being executed.
// If there is a change in the hash, it moves the updated binary to the current binary's path and returns with exit code 1.
// The exit code of 1 triggers the Windows service that is backing wins to execute a failure action (see run.go) after
// a certain number of seconds that will restart the wins binary with the newly updated contents
func enableWatchingMode(ctx context.Context, cfg *config.Config) error {
	if !cfg.Upgrade.IsWatchingMode() {
		return nil
	}

	rawBinPath, err := paths.GetBinaryPath(os.Args[0])
	if err != nil {
		return err
	}
	rawBinChecksum, err := paths.GetFileSHA1Hash(rawBinPath)
	if err != nil {
		return errors.Wrapf(err, "could not get checksum for %q", rawBinPath)
	}

	updateBinPath := cfg.Upgrade.WatchingPath
	updateBinDir := filepath.Dir(updateBinPath)
	updateBinName := filepath.Base(updateBinPath)

	err = paths.EnsureDirectory(updateBinDir)
	if err != nil {
		return errors.Wrapf(err, "could not ensure dir %q", updateBinDir)
	}

	updateHandler := func(watchErr error, watchEvent fsnotify.Event) {
		if watchErr != nil {
			logrus.Errorf("error while watching: %v", watchErr)
			return
		}

		if watchEvent.Op&fsnotify.Write == fsnotify.Write {
			logrus.Debugf("[Upgrade] Catching event %s", watchEvent)
			if filepath.Base(watchEvent.Name) != updateBinName {
				return
			}

			upgradeBinChecksum, err := paths.GetFileSHA1Hash(updateBinPath)
			if err != nil {
				logrus.Errorf("could not get checksum for %q: %v", updateBinPath, err)
				return
			}

			if rawBinChecksum != upgradeBinChecksum {
				logrus.Debugf("[Upgrade] Going to update")
				if err := paths.MoveFile(updateBinPath, rawBinPath); err != nil {
					logrus.Errorf("failed to upgrade %q: %v", rawBinPath, err)
					return
				}

				logrus.Debugf("[Upgrade] Finished updating")
				// have the service failure action to trigger the restart action
				os.Exit(1)
			}
		}
	}
	return paths.Watch(ctx, updateBinDir, updateHandler)
}
