//go:build windows && integration

package service

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	service2 "github.com/rancher/wins/suc/pkg/service"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func TestStopWaitsForLingeringProcess(t *testing.T) {
	s := installTestService(t, testServiceOpts{Linger: 5 * time.Second})

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	pid := servicePID(t, s)
	h := openProcessHandle(t, pid)

	if err := s.Stop(); err != nil {
		t.Fatalf("failed to stop service: %v", err)
	}

	if !processExited(t, h) {
		t.Fatalf("Stop returned before process %d exited", pid)
	}

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start service after stop: %v", err)
	}
}

func TestRestartWaitsForLingeringProcess(t *testing.T) {
	s := installTestService(t, testServiceOpts{Linger: 10 * time.Second})

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start service after stop: %v", err)
	}

	oldPid := servicePID(t, s)
	openProcessHandle(t, oldPid)

	if err := s.Restart(); err != nil {
		t.Fatalf("failed to stop service: %v", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		t.Fatalf("failed to connect to service: %v", err)
	}
	defer m.Disconnect()

	srv, err := m.OpenService(s.Name)
	if err != nil {
		t.Fatalf("failed to open service: %v", err)
	}

	defer srv.Close()

	q, err := srv.Query()
	if err != nil {
		t.Fatalf("failed to query service: %v", err)
	}

	if q.State != windows.SERVICE_RUNNING {
		t.Fatalf("service is not running after a restart")
	}

	newPid := servicePID(t, s)
	if oldPid == newPid {
		t.Fatalf("Service post restart did not get assigned a new PID")
	}
}

// TestSCMRejectsStartWhileStopPending confirms the race the restart tests guard against exists.
// The SCM refuses to start a service until it has fully stopped. If this fails, the harness is
// not reproducing the race and the restart tests prove nothing.
func TestSCMRejectsStartWhileStopPending(t *testing.T) {
	s := installTestService(t, testServiceOpts{StopPending: 5 * time.Second})

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	scm := openSCMHandle(t, s.Name)

	if _, err := scm.Control(svc.Stop); err != nil {
		t.Fatalf("failed to send stop control: %v", err)
	}
	waitForSCMState(t, scm, svc.StopPending, 5*time.Second)

	err := scm.Start()
	if !errors.Is(err, windows.ERROR_SERVICE_ALREADY_RUNNING) {
		t.Fatalf("expected ERROR_SERVICE_ALREADY_RUNNING while stop pending, got %v", err)
	}
}

// TestRestartWhileStopPending covers a service stopped by someone else just before Restart.
// Restart previously skipped Stop for any state other than running and failed to start.
func TestRestartWhileStopPending(t *testing.T) {
	s := installTestService(t, testServiceOpts{StopPending: 5 * time.Second})

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	oldPid := servicePID(t, s)
	h := openProcessHandle(t, oldPid)
	scm := openSCMHandle(t, s.Name)

	if _, err := scm.Control(svc.Stop); err != nil {
		t.Fatalf("failed to send stop control: %v", err)
	}
	waitForSCMState(t, scm, svc.StopPending, 5*time.Second)

	if err := s.Restart(); err != nil {
		t.Fatalf("failed to restart service while stop pending: %v", err)
	}

	if !processExited(t, h) {
		t.Fatalf("Restart returned before the old process %d exited", oldPid)
	}
	if newPid := servicePID(t, s); newPid == oldPid {
		t.Fatalf("service did not get a new process after restart")
	}
}

func TestStopWhileStopPending(t *testing.T) {
	s := installTestService(t, testServiceOpts{StopPending: 5 * time.Second, Linger: 3 * time.Second})

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	pid := servicePID(t, s)
	h := openProcessHandle(t, pid)
	scm := openSCMHandle(t, s.Name)

	if _, err := scm.Control(svc.Stop); err != nil {
		t.Fatalf("failed to send stop control: %v", err)
	}
	waitForSCMState(t, scm, svc.StopPending, 5*time.Second)

	if err := s.Stop(); err != nil {
		t.Fatalf("failed to stop service while stop pending: %v", err)
	}

	if !processExited(t, h) {
		t.Fatalf("Stop returned before process %d exited", pid)
	}
}

func TestConcurrentStops(t *testing.T) {
	s := installTestService(t, testServiceOpts{StopPending: 2 * time.Second, Linger: 3 * time.Second})

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	other, exists, err := service2.Open(s.Name)
	if err != nil || !exists {
		t.Fatalf("failed to open second instance of service %s: exists=%t, err=%v", s.Name, exists, err)
	}
	t.Cleanup(other.Close)

	pid := servicePID(t, s)
	h := openProcessHandle(t, pid)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, stopper := range []*service2.Service{s, other} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = stopper.Stop()
		}()
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("stopper %d failed: %v", i, err)
		}
	}

	if !processExited(t, h) {
		t.Fatalf("both stops returned before process %d exited", pid)
	}
}

func TestStopFailsWhenProcessOutlivesDeadline(t *testing.T) {
	setProcessExitDeadline(t, 2*time.Second)
	s := installTestService(t, testServiceOpts{Linger: 10 * time.Second})

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	err := s.Stop()
	if err == nil {
		t.Fatalf("expected Stop to fail when the process outlives the exit deadline")
	}
	if !strings.Contains(err.Error(), "did not exit") {
		t.Fatalf("expected a process exit error, got %v", err)
	}
}

// TestRestartAfterExternalStopWithLingeringProcess covers a service stopped by someone else whose
// process outlives the stop. The SCM does not block a start on the old process, so Restart
// should bring up a second process alongside it.
func TestRestartAfterExternalStopWithLingeringProcess(t *testing.T) {
	s := installTestService(t, testServiceOpts{Linger: 5 * time.Second})

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	oldPid := servicePID(t, s)
	h := openProcessHandle(t, oldPid)
	scm := openSCMHandle(t, s.Name)

	if _, err := scm.Control(svc.Stop); err != nil {
		t.Fatalf("failed to send stop control: %v", err)
	}
	waitForSCMState(t, scm, svc.Stopped, 5*time.Second)

	if processExited(t, h) {
		t.Fatalf("process exited before the restart was attempted, the linger window was missed")
	}

	if err := s.Restart(); err != nil {
		t.Fatalf("failed to restart service: %v", err)
	}

	if newPid := servicePID(t, s); newPid == oldPid {
		t.Fatalf("service did not get a new process after restart")
	}
}

// TestStopWhileStartPending covers a stop requested during startup. The SCM rejects controls
// while the service is start pending, so Stop has to resend the control once it is running.
func TestStopWhileStartPending(t *testing.T) {
	s := installTestService(t, testServiceOpts{StartPending: 5 * time.Second})
	scm := openSCMHandle(t, s.Name)

	if err := scm.Start(); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}
	waitForSCMState(t, scm, svc.StartPending, 5*time.Second)

	if err := s.Stop(); err != nil {
		t.Fatalf("failed to stop service while start pending: %v", err)
	}

	status, err := scm.Query()
	if err != nil {
		t.Fatalf("failed to query service: %v", err)
	}
	if status.State != svc.Stopped {
		t.Fatalf("service is in state %d after Stop, expected stopped", status.State)
	}
}
