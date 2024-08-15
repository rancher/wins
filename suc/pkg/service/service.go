package service

import (
	"fmt"
	"time"

	"github.com/rancher/wins/pkg/defaults"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

type StartType int

func RefreshWinsService() error {
	logrus.Infof("Restarting the %s service", defaults.WindowsServiceDisplayName)
	winSrv, err := openService(defaults.WindowsServiceName)
	if err != nil {
		logrus.Errorf("Cannot restart %s as the service failed to open: %v", defaults.WindowsServiceName, err)
		return err
	}
	defer winSrv.Close()

	state, err := getServiceState(winSrv)
	if err != nil {
		logrus.Errorf("Could not determine the state of the %s service: %v", defaults.WindowsServiceName, err)
		return fmt.Errorf("could not determine the state of the %s service: %v", defaults.WindowsServiceName, err)
	}

	switch state {
	case svc.Running:
		err = restartService(winSrv)
		if err != nil {
			return err
		}
	case svc.Stopped:
		err = winSrv.Start()
		if err != nil {
			return err
		}
	default:
		logrus.Warnf("Unknown service state '%d', will not start or restart service: ", state)
	}

	return nil
}

func getServiceState(s *mgr.Service) (svc.State, error) {
	q, err := s.Query()
	if err != nil {
		return 0, fmt.Errorf("could not query service %s: %v", s.Name, err)
	}

	return q.State, nil
}

func restartService(s *mgr.Service) error {
	currentState, err := getServiceState(s)
	if err != nil {
		return err
	}

	if currentState != svc.Running {
		// can't restart a service that isn't running
		logrus.Warnf("Cannot restart the %s service as it is not yet running", s.Name)
		return fmt.Errorf("cannot restart the %s service as it is not yet running", s.Name)
	}

	_, err = s.Control(mgr.ServiceRestart)
	if err != nil {
		logrus.Errorf("Encountered error attempting to restart the %s service: %v", s.Name, err)
		return fmt.Errorf("encountered error attempting to restart the %s service: %v", s.Name, err)
	}

	return waitForServiceState(s, svc.Running, 5*time.Second, 5)
}

func openService(name string) (*mgr.Service, error) {
	svcMgr, err := mgr.Connect()
	if err != nil {
		return nil, err
	}
	defer svcMgr.Disconnect()
	return svcMgr.OpenService(name)
}

func StopService(s *mgr.Service) error {
	_, err := s.Control(svc.Stop)
	if err != nil {
		return err
	}

	return waitForServiceState(s, svc.Stopped, 5*time.Second, 5)
}

func waitForServiceState(s *mgr.Service, desiredState svc.State, delayInSeconds time.Duration, maxAttempts int) error {
	transitionedSuccessfully := false
	var err error
	var state svc.State

	for i := 0; i < maxAttempts; i++ {
		state, err = getServiceState(s)
		if err != nil {
			return fmt.Errorf("failed to query service %s: %v", s.Name, err)
		}
		if state == desiredState {
			transitionedSuccessfully = true
			break
		}
		time.Sleep(time.Second * delayInSeconds)
	}

	if !transitionedSuccessfully {
		return fmt.Errorf("%s failed to transition to desired state of '%s' within expected timeframe of %d seconds. last known state was '%s'", s.Name, serviceStateToString(desiredState), int(delayInSeconds.Seconds()*float64(maxAttempts)), serviceStateToString(state))
	}

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
		return ""
	}
}
