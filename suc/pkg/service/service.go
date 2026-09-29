package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
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

	return &Service{
		Name:   name,
		svc:    s,
		Config: cfg,
	}, true, nil
}

// Restart explicitly stops and then starts the Service.
// Restart blocks for up to 60 seconds, or until the svc.State transitions to svc.Running
func (s *Service) Restart() error {
	logrus.Infof("Restarting %s service", s.Name)
	serviceQuery, err := s.svc.Query()
	if err != nil {
		return fmt.Errorf("failed to get state of service %s: %w", s.Name, err)
	}

	// If we're restarting a service that is just coming up, we must wait for it
	// to finish starting before we can send more controls.
	if serviceQuery.State == svc.StartPending {
		serviceQuery, err = s.WaitForState(svc.Running, getStateTransitionDelay(), getStateTransitionAttempts())
		if err != nil {
			return fmt.Errorf("failed to restart service %s, service pending start did not transition before deadline: %w", s.Name, err)
		}
	}

	if serviceQuery.State == svc.Running || serviceQuery.State == svc.StopPending {
		if err = s.Stop(); err != nil {
			logrus.Errorf("Encountered error attempting to stop the %s service: %v", s.Name, err)
			return fmt.Errorf("encountered error attempting to stop the %s service: %w", s.Name, err)
		}
	}

	if err = s.Start(); err != nil {
		return fmt.Errorf("failed to start the %s service while attempting to restart: %w", s.Name, err)
	}

	// service state transition monitoring is handled by s.Start()
	return nil
}

func (s *Service) Start() error {
	serviceQuery, err := s.svc.Query()
	if err != nil {
		return fmt.Errorf("failed to get status of service %s: %w", s.Name, err)
	}

	if serviceQuery.State == windows.SERVICE_START_PENDING {
		_, err = s.WaitForState(svc.Running, getStateTransitionDelay(), getStateTransitionAttempts())
		if err != nil {
			return fmt.Errorf("failed to start service %s, service pending start did not transition after deadline: %w", s.Name, err)
		}
		return nil
	}

	err = s.svc.Start()
	if err != nil {
		if !errors.Is(err, windows.ERROR_SERVICE_ALREADY_RUNNING) {
			return fmt.Errorf("failed to start service %s: %w", s.Name, err)
		}
	}

	_, err = s.WaitForState(svc.Running, getStateTransitionDelay(), getStateTransitionAttempts())
	return err
}

// Stop sends a svc.Stop control signal to the Service, waits for it to enter the
// svc.Stopped state, and then waits for the process backing it to exit. If the service
// is already stopping, no control signal is sent and only the transition is awaited.
// If the service is already stopped, this function is a no-op.
func (s *Service) Stop() error {
	serviceQuery, err := s.svc.Query()
	if err != nil {
		return fmt.Errorf("error getting status for %s service while attempting to send stop signal: %w", s.Name, err)
	}

	if serviceQuery.State == windows.SERVICE_STOPPED {
		logrus.Debugf("cannot stop service %s as it is not running", s.Name)
		return nil
	}

	// The SCM considers a service stopped as soon as it reports svc.Stopped, even if the process
	// backing it is still running, and will start a new process alongside it. A lingering process
	// may still hold resources the new one needs, such as ports, named pipes, or file locks. To
	// avoid this, we not only wait on the SCM state transition but also on the exit of the process.
	waiter, err := s.acquireProcessWaiter(serviceQuery.ProcessId)
	if err != nil {
		return fmt.Errorf("failed to open the process backing the %s service: %w", s.Name, err)
	}

	if waiter != nil {
		defer waiter.closeAndLog(s.Name)
	}

	logrus.Debugf("Stopping %s service", s.Name)
	if serviceQuery.State == windows.SERVICE_STOP_PENDING {
		logrus.Debugf("service %s is already stopping, waiting", s.Name)
	} else if _, err = s.svc.Control(svc.Stop); err != nil {
		// The service may have stopped on its own and reaped the underlying process between the query above and this control.
		// All of these mean it is already on its way down, so we can just wait for it to fully stop.
		if !errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE) && !errors.Is(err, windows.ERROR_SERVICE_CANNOT_ACCEPT_CTRL) && !errors.Is(err, windows.ERROR_BROKEN_PIPE) {
			return fmt.Errorf("failed to send Stop signal to %s: %w", s.Name, err)
		}
		logrus.Debugf("service %s could not accept a stop control (%v), waiting for it to stop", s.Name, err)
	}

	if _, err = s.WaitForState(svc.Stopped, getStateTransitionDelay(), getStateTransitionAttempts()); err != nil {
		return fmt.Errorf("failed to stop the %s service: %w", s.Name, err)
	}

	if waiter != nil {
		if err = waiter.wait(ProcessExitDeadline); err != nil {
			return fmt.Errorf("the %s service reported stopped but its process did not exit: %w", s.Name, err)
		}
	}

	logrus.Debugf("Stopped %s service", s.Name)
	return nil
}

// Close closes the Service
func (s *Service) Close() {
	s.svc.Close()
}

// WaitForState monitors the current state of the Service and waits for it to transition to the desiredState.
// WaitForState will wait for the state to transition for up to (delay * maxAttempts)
func (s *Service) WaitForState(desiredState svc.State, delay time.Duration, maxAttempts int) (svc.Status, error) {
	transitionedSuccessfully := false
	var err error
	var state svc.State
	var serviceQuery svc.Status

	logrus.Infof("Waiting for service %s to enter state %s", s.Name, serviceStateToString(desiredState))

	for i := 0; i < maxAttempts; i++ {
		serviceQuery, err = s.svc.Query()
		if err != nil {
			return serviceQuery, fmt.Errorf("failed to query service %s: %w", s.Name, err)
		}
		state = serviceQuery.State
		if state == desiredState {
			transitionedSuccessfully = true
			break
		}
		logrus.Infof("Waiting for service %s to enter state %s, current state: %s", s.Name, serviceStateToString(desiredState), serviceStateToString(state))
		time.Sleep(delay)
	}

	if !transitionedSuccessfully {
		return serviceQuery, fmt.Errorf("%s failed to transition to desired state of %s within expected timeframe of %s. last known state was %s", s.Name, serviceStateToString(desiredState), delay*time.Duration(maxAttempts), serviceStateToString(state))
	}

	logrus.Infof("Service %s successfully transitioned to state %s", s.Name, serviceStateToString(desiredState))
	return serviceQuery, nil
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

// acquireProcessWaiter opens a handle to the process backing the Service so that its
// exit can be observed later. It must be called while the service is still running.
// Holding a handle reserves the process id, but resolving an id after the service has
// stopped may open an unrelated process which has since reused it. A nil waiter and a
// nil error are returned when there is no process left to wait on.
func (s *Service) acquireProcessWaiter(pid uint32) (*processWaiter, error) {
	if pid == 0 {
		logrus.Debugf("Service %s reports no process id, nothing to wait on", s.Name)
		return nil, nil
	}

	logrus.Debugf("Opening process %d backing service %s", pid, s.Name)
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err != nil {
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			logrus.Debugf("Process %d backing service %s has already exited, nothing to wait on", pid, s.Name)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open process %d backing service %s: %w", pid, s.Name, err)
	}

	waiter := &processWaiter{pid: pid, handle: handle}

	// The process may have exited between the query which reported pid and the OpenProcess
	// call above, leaving us holding a handle to an unrelated process which reused the id.
	// Because the handle reserves the id, the service still reporting it is proof that the
	// handle refers to the service's own process.
	confirm, err := s.svc.Query()
	if err != nil {
		waiter.closeAndLog(s.Name)
		return nil, fmt.Errorf("failed to confirm process %d backing service %s: %w", pid, s.Name, err)
	}

	if confirm.State == svc.Stopped || confirm.ProcessId != pid {
		logrus.Debugf("Service %s no longer reports process %d, nothing to wait on", s.Name, pid)
		waiter.closeAndLog(s.Name)
		return nil, nil
	}

	return waiter, nil
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
