package powershells

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/rancher/wins/pkg/panics"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const newline = "\r\n"

// ExecuteCommand executes the `command`, this method will be blocked until finish or error occur,
// returns nil when exit code is 0.
func (ps *PowerShell) ExecuteCommand(ctx context.Context, command string) error {
	if len(command) == 0 {
		return errors.New("can't exec blank command")
	}
	logrus.Debugf("[PowerShell - Stdin]: %s", command)

	// prepare
	args := append(ps.args, "-NoLogo", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", command)
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
		defer panics.Log()
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
		defer panics.Log()
		defer cmdStderr.Close()

		buf := make([]byte, 1<<10)
		for {
			readSize, err := cmdStderr.Read(buf)
			if readSize > 0 {
				ret := string(buf[:readSize])

				if logrus.GetLevel() == logrus.DebugLevel {
					logrus.Errorf("[PowerShell - Stderr]: %s", ret)
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
		defer panics.Log()

		if err := cmd.Run(); err != nil {
			if cmd.ProcessState != nil {
				if cmd.ProcessState.Success() {
					return nil
				}
			}

			return errors.Wrapf(err, "could not execute command %s", command)
		}

		return nil
	})

	return errg.Wait()
}

// Commands holds the input of PowerShell.
func (ps *PowerShell) Commands() (*PowerShellCommands, error) {
	// prepare
	args := append(ps.args, "-NoLogo", "-NonInteractive", "-NoExit", "-WindowStyle", "Hidden", "-Command", "-")

	cmd := exec.Command(ps.path, args...)
	cmdStdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, errors.Wrap(err, "could not take over the PowerShell's stdin stream")
	}
	cmdStdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "could not take over the PowerShell's stdout stream")
	}
	cmdStderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err, "could not take over the PowerShell's stderr stream")
	}
	err = cmd.Start()
	if err != nil {
		return nil, errors.Wrap(err, "could not spawn PowerShell process")
	}

	return &PowerShellCommands{
		cmd:       cmd,
		cmdStdin:  cmdStdin,
		cmdStdout: cmdStdout,
		cmdStderr: cmdStderr,
		stdout:    ps.stdout,
		stderr:    ps.stderr,
	}, nil
}

type PowerShellCommands struct {
	cmd       *exec.Cmd
	cmdStdin  io.WriteCloser
	cmdStdout io.ReadCloser
	cmdStderr io.ReadCloser

	stdout StdStream
	stderr StdStream
}

// Execute allows to input a `command` one by one, returns execution result, stdout info, stderr info and error.
func (psc *PowerShellCommands) Execute(ctx context.Context, command string) (string, string, error) {
	if len(command) == 0 {
		return "", "", errors.New("could not execute blank cmd")
	}
	logrus.Debugf("[PowerShell - Stdin]: %s", command)

	commandStdout := &strings.Builder{}
	commandStderr := &strings.Builder{}
	commandSignal := newSignal()
	errg := &errgroup.Group{}

	commandWrapper := fmt.Sprintf("Try {%s} Catch {[System.Console]::Error.Write($_.Exception.Message)}; [System.Console]::Out.Write(\"%s\"); [System.Console]::Error.Write(\"%s\");%s", command, commandSignal, commandSignal, newline)
	_, err := psc.cmdStdin.Write([]byte(commandWrapper))
	if err != nil {
		return "", "", errors.Errorf("could not input %q command into PowerShell stdin stream", commandWrapper)
	}

	errg.Go(func() error {
		buf := make([]byte, 1<<10)
		for {
			readSize, err := psc.cmdStdout.Read(buf)
			if readSize > 0 {
				ret := strings.TrimSuffix(string(buf[:readSize]), commandSignal)
				if ret != "" {
					if logrus.GetLevel() == logrus.DebugLevel {
						logrus.Infof("[PowerShell - Stdout]: %s", ret)
					}
					commandStdout.WriteString(ret)
					if psc.stderr != nil {
						psc.stdout(ret)
					}
				}

				if len(ret) != readSize {
					break
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
		buf := make([]byte, 1<<10)
		for {
			readSize, err := psc.cmdStderr.Read(buf)
			if readSize > 0 {
				ret := strings.TrimSuffix(string(buf[:readSize]), commandSignal)
				if ret != "" {
					if logrus.GetLevel() == logrus.DebugLevel {
						logrus.Errorf("[PowerShell - Stderr]: %s", ret)
					}
					commandStderr.WriteString(ret)
					if psc.stderr != nil {
						psc.stderr(ret)
					}
				}

				if len(ret) != readSize {
					break
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

	err = errg.Wait()
	if err != nil {
		return "", "", errors.Wrapf(err, "could not execute command %s", command)
	}

	return commandStdout.String(), commandStderr.String(), nil
}

func (psc *PowerShellCommands) Close() error {
	_, err := psc.cmdStdin.Write([]byte("exit" + newline))
	if err != nil {
		return err
	}

	err = psc.cmdStdin.Close()
	if err != nil {
		return err
	}
	err = psc.cmdStdout.Close()
	if err != nil {
		return err
	}
	err = psc.cmdStderr.Close()
	if err != nil {
		return err
	}

	return psc.cmd.Wait()
}

func newSignal() string {
	randArr := make([]byte, 8)
	rand.Read(randArr)

	return fmt.Sprintf("#%s#", hex.EncodeToString(randArr))
}
