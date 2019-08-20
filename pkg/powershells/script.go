package powershells

import (
	"context"
	"io"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// ExecuteScript executes the `scriptPath` script with `scriptArgs`, this method will be blocked until finish or error occur,
// returns nil when exit code is 0
func (ps *PowerShell) ExecuteScript(ctx context.Context, scriptPath string, scriptArgs ...string) error {
	if len(scriptPath) == 0 {
		return errors.New("can't exec blank script")
	}
	logrus.Debugf("[PowerShell - Stdin]: %s, %v", scriptPath, scriptArgs)

	// prepare
	args := append(ps.args, "-NoLogo", "-NonInteractive", "-WindowStyle", "Hidden", "-File", scriptPath)
	args = append(args, scriptArgs...)

	errg, subCtx := errgroup.WithContext(ctx)

	cmd := exec.CommandContext(subCtx, ps.path, args...)
	cmdStdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "could not take over the PowerShell's stdout stream")
	}
	cmdStderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "could not take over the PowerShell's stderr stream")
	}

	errg.Go(func() error {
		defer cmdStdout.Close()

		buf := make([]byte, 1<<10)
		for {
			readSize, err := cmdStdout.Read(buf)
			if readSize > 0 {
				ret := string(buf[:readSize])

				if logrus.GetLevel() == logrus.DebugLevel {
					logrus.Infof("[PowerShell - Stdout]: %s", ret)
				}
				if ps.stdout != nil {
					ps.stdout(ret)
				}
			}

			if err != nil {
				if io.EOF != err && io.ErrClosedPipe != err {
					return err
				}
				break
			}
		}

		return nil
	})

	errg.Go(func() error {
		defer cmdStderr.Close()

		buf := make([]byte, 1<<10)
		for {
			readSize, err := cmdStderr.Read(buf)
			if readSize > 0 {
				ret := string(buf[:readSize])
				if logrus.GetLevel() == logrus.DebugLevel {
					logrus.Errorf("[PowerShell - Stdout]: %s", ret)
				}
				if ps.stderr != nil {
					ps.stderr(ret)
				}
			}

			if err != nil {
				if io.EOF != err && io.ErrClosedPipe != err {
					return err
				}
				break
			}
		}

		return nil
	})

	errg.Go(func() error {
		if err := cmd.Run(); err != nil {
			if cmd.ProcessState.Success() {
				return nil
			}

			return errors.Wrapf(err, "could not execute script %s", scriptPath)
		}

		return nil
	})

	return errg.Wait()
}
