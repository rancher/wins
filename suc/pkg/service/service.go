package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	stateTransitionAttempts       = 12
	stateTransitionDelayInSeconds = 5
)

var ProcessExitDeadline = 30 * time.Second

// Service is a wrapper around a mgr.Service which simplifies
// common operations and bundles relevant configuration information.
type Service struct {
	Name   string
	svc    *mgr.Service
	Config mgr.Config

	// process is the last process seen backing the service, see query.
	process *processWaiter
}

// processWaiter owns an open handle to the process backing a Windows service.
// Windows will not reuse a process id while a handle to it remains open, so the
// handle continues to refer to the same process even after that process exits.
type processWaiter struct {
	pid    uint32
	handle windows.Handle
}

// Open opens a Windows service and returns a Service containing the relevant mgr.Config.
// If the provided service does not exist, a nil error and a false boolean will be returned.
// The caller of Open is responsible for closing the returned Service (via Service.Close()).
func Open(name string) (service *Service, serviceExists bool, err error) {
	logrus.Debugf("Opening %s service", name)
	svcMgr, err := mgr.Connect()
	if err != nil {
		return nil, false, fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer svcMgr.Disconnect()

	s, err := svcMgr.OpenService(name)
	doesNotExist := errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST)
	if err != nil && !doesNotExist {
		return nil, false, fmt.Errorf("failed to open service %s via service manager: %w", name, err)
	}

	if doesNotExist {
		return nil, false, nil
	}

	if s == nil {
		return nil, false, fmt.Errorf("failed to open service %s, a nil service was returned", name)
	}

	cfg, err := s.Config()
	if err != nil {
		return nil, false, fmt.Errorf("failed to open config for service %s via service manager: %w", name, err)
	}

	service = &Service{
		Name:   name,
		svc:    s,
		Config: cfg,
	}

	// Record the process backing the service now, in case it is stopped by someone else before Stop is called.
	if _, err = service.query(); err != nil {
		logrus.Warnf("Failed to record the process backing the %s service: %v", name, err)
	}

	return service, true, nil
}

// Restart stops and then starts the Service, waiting for each transition to complete.
func (s *Service) Restart() error {
	logrus.Infof("Restarting %s service", s.Name)
	if err := s.Stop(); err != nil {
		return fmt.Errorf("failed to stop the %s service while attempting to restart: %w", s.Name, err)
	}
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start the %s service while attempting to restart: %w", s.Name, err)
	}
	return nil
}

// Start starts the Service and waits for it to enter the svc.Running state. A service which
// is still stopping is started once it has fully stopped, and one which is already starting
// is waited on. If the service was seen running and has since been stopped by someone else,
// it is only started once that process has exited, for the same reasons as Stop.
func (s *Service) Start() error {
	var lastErr error
	err := s.waitForState(svc.Running, func(status svc.Status) error {
		if status.State != svc.Stopped {
			return nil
		}
		if err := s.waitForProcessExit(); err != nil {
			return err
		}
		if lastErr = s.svc.Start(); lastErr != nil {
			logrus.Debugf("Failed to start %s service, retrying: %v", s.Name, lastErr)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to start the %s service: %w", s.Name, errors.Join(err, lastErr))
	}
	return nil
}

// Stop sends a svc.Stop control signal to the Service, waits for it to enter the svc.Stopped
// state, and then waits for the process backing it to exit. We wait on both conditions
// as a lingering process may still hold resources the new one needs, such as ports,
// named pipes, or file locks.
//
// A service which is in a pending state is sent the control once it has settled. A service
// which is svc.StopPending is waited on. A stopped service is not sent the control, but its
// process is still waited on if it was seen running.
func (s *Service) Stop() error {
	status, err := s.query()
	if err != nil {
		return fmt.Errorf("error getting status for %s service while attempting to send stop signal: %w", s.Name, err)
	}

	if status.State != svc.Stopped {
		logrus.Debugf("Stopping %s service", s.Name)
		var lastErr error
		err = s.waitForState(svc.Stopped, func(status svc.Status) error {
			if status.State != svc.Running && status.State != svc.Paused {
				return nil
			}
			if status.Accepts&svc.AcceptStop == 0 {
				return fmt.Errorf("service %s does not accept stop signals while %s", s.Name, serviceStateToString(status.State))
			}
			// The control can be rejected if another control is still being delivered to the service.
			if _, lastErr = s.svc.Control(svc.Stop); lastErr != nil {
				logrus.Debugf("Failed to send stop signal to %s service, retrying: %v", s.Name, lastErr)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to stop the %s service: %w", s.Name, errors.Join(err, lastErr))
		}
	}

	if err = s.waitForProcessExit(); err != nil {
		return err
	}

	logrus.Debugf("Stopped %s service", s.Name)
	return nil
}

// waitForProcessExit waits for the last process seen backing the Service to exit, and then
// releases it. It is a no-op if no process has been seen since it was last called.
func (s *Service) waitForProcessExit() error {
	if s.process == nil {
		return nil
	}
	defer s.closeProcess()
	if err := s.process.wait(ProcessExitDeadline); err != nil {
		return fmt.Errorf("the %s service reported stopped but its process did not exit: %w", s.Name, err)
	}
	return nil
}

// waitForState queries the Service until it enters the target state, or the state transition
// timeout elapses, calling step with each other status it observes. step decides what to do
// from that status alone, so controls are only sent to a service in a state that can accept
// them. Anything step attempts which fails is retried on the next poll against a fresh status,
// and step returns an error only when the target state can never be reached.
func (s *Service) waitForState(target svc.State, step func(svc.Status) error) error {
	delay := getStateTransitionDelay()
	timeout := delay * time.Duration(getStateTransitionAttempts())

	var status svc.Status
	err := wait.PollUntilContextTimeout(context.Background(), delay, timeout, true, func(context.Context) (bool, error) {
		var err error
		if status, err = s.query(); err != nil {
			return false, err
		}
		if status.State == target {
			return true, nil
		}
		logrus.Infof("Waiting for service %s to enter state %s, current state: %s", s.Name, serviceStateToString(target), serviceStateToString(status.State))
		return false, step(status)
	})
	if wait.Interrupted(err) {
		return fmt.Errorf("%s failed to transition to desired state of %s within expected timeframe of %s. last known state was %s", s.Name, serviceStateToString(target), timeout, serviceStateToString(status.State))
	}
	return err
}

// Close closes the Service
func (s *Service) Close() {
	s.closeProcess()
	s.svc.Close()
}

// UpdateConfig commits the stored Service.Config to the registry. Note that
// the config can only be updated a single time after a service has been opened.
// In order to update the config again, the service must be closed and reopened.
func (s *Service) UpdateConfig() error {
	j, err := json.MarshalIndent(s.Config, "", " ")
	if err != nil {
		return fmt.Errorf("error encountered while saving config, could not marshal to json: %w", err)
	}
	logrus.Debugf("Updating config for %s service. Config to be saved:\n%s ", s.Name, string(j))
	return s.svc.UpdateConfig(s.Config)
}

// RefreshConfig updates the Service.Config with the latest config used by the Windows Service.
func (s *Service) RefreshConfig() error {
	cfg, err := s.svc.Config()
	if err != nil {
		return fmt.Errorf("failed to refresh config for service '%s': %w", s.Name, err)
	}
	s.Config = cfg
	return nil
}

// query returns the current status of the Service, and holds a handle to the process backing it
// if one is reported. The SCM stops reporting a process id once the service reports
// svc.Stopped, even if its process is still running, so the handle is kept past that point to
// let Stop wait on a process which was stopped by another caller. Windows will not reuse the id
// of a process while a handle to it is open, so the handle cannot come to refer to another process.
func (s *Service) query() (svc.Status, error) {
	queriedAt := time.Now()
	status, err := s.svc.Query()
	if err != nil {
		return status, fmt.Errorf("failed to query service %s: %w", s.Name, err)
	}

	if status.ProcessId == 0 || (s.process != nil && s.process.pid == status.ProcessId) {
		return status, nil
	}

	handle, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, status.ProcessId)
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		logrus.Debugf("Process %d backing service %s has already exited", status.ProcessId, s.Name)
		return status, nil
	}
	if err != nil {
		return status, fmt.Errorf("failed to open process %d backing service %s: %w", status.ProcessId, s.Name, err)
	}
	process := &processWaiter{pid: status.ProcessId, handle: handle}

	// The process may have exited between the query and OpenProcess, and its id been reused. Ids are
	// only reused once a process has exited, so a process created before the query is the service's own.
	var created, exited, kernel, user windows.Filetime
	if err = windows.GetProcessTimes(handle, &created, &exited, &kernel, &user); err != nil {
		process.closeAndLog(s.Name)
		return status, fmt.Errorf("failed to get the creation time of process %d backing service %s: %w", status.ProcessId, s.Name, err)
	}
	if time.Unix(0, created.Nanoseconds()).After(queriedAt) {
		logrus.Debugf("Process %d backing service %s has already exited and its id was reused", status.ProcessId, s.Name)
		process.closeAndLog(s.Name)
		return status, nil
	}

	s.closeProcess()
	s.process = process
	return status, nil
}

func (s *Service) closeProcess() {
	if s.process != nil {
		s.process.closeAndLog(s.Name)
		s.process = nil
	}
}

// wait blocks until the process exits or the timeout elapses.
func (p *processWaiter) wait(timeout time.Duration) error {
	event, err := windows.WaitForSingleObject(p.handle, uint32(timeout.Milliseconds()))
	if err != nil {
		return fmt.Errorf("failed to wait on process %d: %w", p.pid, err)
	}

	switch event {
	case windows.WAIT_OBJECT_0:
		return nil
	case uint32(windows.WAIT_TIMEOUT):
		return fmt.Errorf("process %d did not exit within %s", p.pid, timeout)
	default:
		return fmt.Errorf("unexpected wait result %d for process %d", event, p.pid)
	}
}

func (p *processWaiter) closeAndLog(serviceName string) {
	if err := windows.Close(p.handle); err != nil {
		logrus.Errorf("Failed to close the handle for process %d backing service %s: %v", p.pid, serviceName, err)
	}
}
