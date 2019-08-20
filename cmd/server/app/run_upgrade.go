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
	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		return errors.Wrap(err, "could not detect the interactive state")
	}
	if isInteractive {
		logrus.Warn("could not upgrade during windows service interacting")
		return nil
	}

	// watching mode
	if err := enableWatchingMode(ctx, cfg); err != nil {
		return err
	}

	return nil
}

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
			logrus.Errorf("error while watching: %v")
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
