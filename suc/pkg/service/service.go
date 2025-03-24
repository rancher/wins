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

// Service is a wrapper around a mgr.Service which simplifies
// common operations and bundles relevant configuration information.
type Service struct {
	Name   string
	svc    *mgr.Service
	Config mgr.Config
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

// GetState queries the Service and returns the svc.State
func (s *Service) GetState() (svc.State, error) {
	q, err := s.svc.Query()
	if err != nil {
		return 0, fmt.Errorf("could not query service %s: %w", s.Name, err)
	}

	return q.State, nil
}

// Restart explicitly stops and then starts the Service.
// Restart blocks for up to 25 seconds, or until the svc.State transitions to svc.Running
func (s *Service) Restart() error {
	logrus.Infof("Restarting %s service", s.Name)
	currentState, err := s.GetState()
	if err != nil {
		return fmt.Errorf("failed to get state of service %s: %w", s.Name, err)
	}

	if currentState == svc.Running {
		if err = s.Stop(); err != nil {
			logrus.Errorf("Encountered error attempting to stop the %s service: %v", s.Name, err)
			return fmt.Errorf("encountered error attempting to stop the %s service: %w", s.Name, err)
		}
	}

	if err = s.svc.Start(); err != nil {
		return fmt.Errorf("failed to start the %s service while attempting to restart: %w", s.Name, err)
	}

	return s.WaitForState(svc.Running, getStateTransitionDelayInSeconds(), getStateTransitionAttempts())
}

// Stop sends a svc.Stop control signal to the Service and waits
// for it to enter the svc.Stopped state
func (s *Service) Stop() error {
	state, err := s.GetState()
	if err != nil {
		return fmt.Errorf("error getting state for %s service while attempting to send stop signal", s.Name)
	}

	if state == windows.SERVICE_STOPPED {
		logrus.Debugf("cannot stop service %s as it is not running", s.Name)
		return nil
	}

	_, err = s.svc.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to send Stop signal to %s: %w", s.Name, err)
	}

	return s.WaitForState(svc.Stopped, getStateTransitionDelayInSeconds(), getStateTransitionAttempts())
}

// Close closes the Service
func (s *Service) Close() {
	s.svc.Close()
}

// WaitForState monitors the current state of the Service and waits for it to transition to the desiredState.
// WaitForState will wait for the state to transition for up to (delayInSeconds * maxAttempts)
func (s *Service) WaitForState(desiredState svc.State, delayInSeconds time.Duration, maxAttempts int) error {
	transitionedSuccessfully := false
	var err error
	var state svc.State

	logrus.Infof("Waiting for service %s to enter state %s", s.Name, serviceStateToString(desiredState))

	for i := 0; i < maxAttempts; i++ {
		state, err = s.GetState()
		if err != nil {
			return fmt.Errorf("failed to query service %s: %w", s.Name, err)
		}
		if state == desiredState {
			transitionedSuccessfully = true
			break
		}
		logrus.Infof("Waiting for service %s to enter state %s, current state: %s", s.Name, serviceStateToString(desiredState), serviceStateToString(state))
		time.Sleep(time.Second * delayInSeconds)
	}

	if !transitionedSuccessfully {
		return fmt.Errorf("%s failed to transition to desired state of %s within expected timeframe of %d seconds. last known state was %s", s.Name, serviceStateToString(desiredState), int(delayInSeconds.Seconds()*float64(maxAttempts)), serviceStateToString(state))
	}

	logrus.Infof("Service %s successfully transitioned to state %s", s.Name, serviceStateToString(desiredState))
	return nil
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
