package service

import (
	"fmt"
	"time"

	"github.com/rancher/wins/pkg/defaults"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func RefreshWinsService() error {
	logrus.Infof("Restarting the %s service", defaults.WindowsServiceDisplayName)
	winSrv, err := openService(defaults.WindowsServiceName)
	if err != nil {
		logrus.Errorf("Cannot restart %s as the service failed to open: %v", defaults.WindowsServiceName, err)
		return fmt.Errorf("failed to refresh the %s service: %w", winSrv.Name, err)
	}
	defer winSrv.Close()

	err = restartService(winSrv)
	if err != nil {
		return fmt.Errorf("failed to refresh the %s service: %w", winSrv.Name, err)
	}

	return nil
}

func getServiceState(s *mgr.Service) (svc.State, error) {
	q, err := s.Query()
	if err != nil {
		return 0, fmt.Errorf("could not query service %s: %w", s.Name, err)
	}

	return q.State, nil
}

func restartService(s *mgr.Service) error {
	currentState, err := getServiceState(s)
	if err != nil {
		return fmt.Errorf("failed to get state of service %s: %w", s.Name, err)
	}

	if currentState == svc.Running {
		if err = stopService(s); err != nil {
			logrus.Errorf("Encountered error attempting to stop the %s service: %v", s.Name, err)
			return fmt.Errorf("encountered error attempting to stop the %s service: %w", s.Name, err)
		}
	}

	if err = s.Start(); err != nil {
		return fmt.Errorf("failed to restart the %s service: %w", s.Name, err)
	}

	return waitForServiceState(s, svc.Running, 5, 5)
}

func openService(name string) (*mgr.Service, error) {
	svcMgr, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer svcMgr.Disconnect()
	return svcMgr.OpenService(name)
}

func stopService(s *mgr.Service) error {
	_, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to send Stop signal to %s: %w", s.Name, err)
	}

	return waitForServiceState(s, svc.Stopped, 5, 5)
}

func waitForServiceState(s *mgr.Service, desiredState svc.State, delayInSeconds time.Duration, maxAttempts int) error {
	transitionedSuccessfully := false
	var err error
	var state svc.State

	for i := 0; i < maxAttempts; i++ {
		state, err = getServiceState(s)
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

// serviceStateToString translates a svc.State to its string representation
func serviceStateToString(state svc.State) string {
	switch state {
	case svc.Running:
		return "Running"
	case svc.Stopped:
		return "Stopped"
	case svc.StopPending:
		return "Stop Pending"
	case svc.StartPending:
		return "Start Pending"
	default:
		return "Unknown State"
	}
}
