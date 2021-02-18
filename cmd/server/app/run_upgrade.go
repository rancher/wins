package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/rancher/wins/cmd/server/config"
	"github.com/rancher/wins/pkg/paths"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
)

func setupUpgrading(ctx context.Context, cfg *config.Config, cfgPath string) error {
	isWindowsService, err := svc.IsWindowsService()
	if err != nil {
		return errors.Wrap(err, "could not detect whether the current process is running as a Windows service")
	}
	if !isWindowsService {
		logrus.Warn("cannot upgrade unless the current process is being executed as a Windows service")
		return nil
	}

	// watching mode
	if err := enableWatchingMode(ctx, cfg, cfgPath); err != nil {
		return err
	}

	return nil
}

// enableWatchingMode sets up a file system notification-based watcher on the cfg.Upgrade.WatchingPath and cfgPath.
// If a write is detected in to a file in either directory, the watcher computes whether the SHA1 hash of the file
// within that path differs from the current binary that is being executed or the current config being used.
// If there is a change in the hash of the binary, it moves the updated binary to the current binary's path and returns with exit code 1.
// If there is a change in the hash of the config, it returns with exit code 1.
// The exit code of 1 triggers the Windows service that is backing wins to execute a failure action (see run.go) after
// a certain number of seconds that will restart the wins binary with the newly updated contents
func enableWatchingMode(ctx context.Context, cfg *config.Config, cfgPath string) error {
	if !cfg.Upgrade.IsWatchingMode() {
		return nil
	}

	// Initialize fields for onChange functions
	rawBinPath, err := paths.GetBinaryPath(os.Args[0])
	if err != nil {
		return err
	}
	onBinChange := func() {
		logrus.Debugf("[Upgrade] Going to update")
		if err := paths.MoveFile(cfg.Upgrade.WatchingPath, rawBinPath); err != nil {
			logrus.Errorf("failed to upgrade %q: %v", rawBinPath, err)
			return
		}
		logrus.Debugf("[Upgrade] Finished updating")
	}
	onCfgChange := func() {
		logrus.Debugf("[Upgrade] Restarting wins with new config")
	}

	// Create filesystem watchers
	watchBinConfig, err := newWatchConfig(rawBinPath, onBinChange)
	if err != nil {
		return err
	}
	watchCfgConfig, err := newWatchConfig(cfgPath, onCfgChange)
	if err != nil {
		return err
	}

	return multierror.Append(watchBinConfig.watch(ctx), watchCfgConfig.watch(ctx))
}

type watchConfig struct {
	path     string
	dir      string
	name     string
	checksum string
	onChange func()
}

func newWatchConfig(path string, onChange func()) (w *watchConfig, err error) {
	w.path = path
	w.onChange = onChange
	w.dir = filepath.Dir(path)
	w.name = filepath.Base(path)
	w.checksum, err = paths.GetFileSHA1Hash(path)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get checksum for %q", path)
	}
	err = paths.EnsureDirectory(w.dir)
	if err != nil {
		return nil, errors.Wrapf(err, "could not ensure dir %q", w.dir)
	}
	return w, nil
}

func (w *watchConfig) watch(ctx context.Context) error {
	return paths.Watch(ctx, w.dir, w.handle)
}

func (w *watchConfig) handle(watchErr error, watchEvent fsnotify.Event) {
	if watchErr != nil {
		logrus.Errorf("error while watching: %v")
		return
	}

	if watchEvent.Op&fsnotify.Write == fsnotify.Write {
		logrus.Debugf("[Upgrade] Catching event %s", watchEvent)
		if filepath.Base(watchEvent.Name) != w.name {
			return
		}

		checksum, err := paths.GetFileSHA1Hash(w.path)
		if err != nil {
			logrus.Errorf("could not get checksum for %q: %v", w.path, err)
			return
		}

		if checksum != w.checksum {
			// trigger the function that should be called on a change and have the
			// service failure action to trigger the restart action
			w.onChange()
			os.Exit(1)
		}
	}
}
