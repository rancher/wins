//go:build windows && integration

package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"

	service2 "github.com/rancher/wins/suc/pkg/service"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	testServicePrefix  = "wins-it-"
	testServiceExeName = "wins-it-service.exe"

	serviceRemovalTimeout = 30 * time.Second
	processKillTimeout    = 10 * time.Second
)

var testServiceCount int

// testServiceOpts controls the behavior of the test service, see tests/integration/testservice.
type testServiceOpts struct {
	// Linger keeps the process alive after the service has reported stopped.
	Linger time.Duration
	// StopPending holds the service in the stop pending state before it reports stopped.
	StopPending time.Duration
	// StartPending holds the service in the start pending state before it reports running.
	StartPending time.Duration
}

func (o testServiceOpts) args(name string) []string {
	var args []string
	if o.Linger > 0 {
		args = append(args, "--linger="+o.Linger.String())
	}
	if o.StopPending > 0 {
		args = append(args, "--stop-pending="+o.StopPending.String())
	}
	if o.StartPending > 0 {
		args = append(args, "--start-pending="+o.StartPending.String())
	}
	return append(args, name)
}

// installTestService creates a stopped, manual start service backed by the test service
// binary and returns it opened through Open. The service, and any process it leaves behind,
// is removed when the test completes. Tests must not call Close on the returned Service.
//
// Cleanup kills every test service process, so tests using this must not run in parallel.
func installTestService(t *testing.T, opts testServiceOpts) *service2.Service {
	t.Helper()

	name := testServiceName(t)

	m, err := mgr.Connect()
	if err != nil {
		t.Fatalf("failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	created, err := m.CreateService(name, testServiceExe, mgr.Config{StartType: mgr.StartManual, DisplayName: name}, opts.args(name)...)
	if err != nil {
		t.Fatalf("failed to create service %s: %v", name, err)
	}
	created.Close()

	t.Cleanup(func() {
		if err := killTestServiceProcesses(); err != nil {
			t.Errorf("failed to kill test service processes: %v", err)
		}
		if err := deleteService(name); err != nil {
			t.Errorf("failed to delete service %s: %v", name, err)
		}
	})

	s, exists, err := service2.Open(name)
	if err != nil {
		t.Fatalf("failed to open service %s: %v", name, err)
	}
	if !exists {
		t.Fatalf("service %s does not exist after being created", name)
	}

	// The SCM will not finish deleting the service while this handle is open.
	t.Cleanup(s.Close)

	return s
}

// testServiceName derives a unique, valid service name from the test name. Subtest names contain '/', which service names may not.
func testServiceName(t *testing.T) string {
	testServiceCount++
	sanitized := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, t.Name())
	if len(sanitized) > 200 {
		sanitized = sanitized[:200]
	}
	return testServicePrefix + sanitized + "-" + strconv.Itoa(testServiceCount)
}

// servicePID returns the id of the process currently backing the service, failing the test if there is none.
func servicePID(t *testing.T, s *service2.Service) uint32 {
	t.Helper()
	m, err := mgr.Connect()
	if err != nil {
		t.Fatalf("failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	h, err := m.OpenService(s.Name)
	if err != nil {
		t.Fatalf("failed to open service %s: %v", s.Name, err)
	}
	defer h.Close()

	status, err := h.Query()
	if err != nil {
		t.Fatalf("failed to query service %s: %v", s.Name, err)
	}
	if status.ProcessId == 0 {
		t.Fatalf("service %s has no backing process, state is %d", s.Name, status.State)
	}
	return status.ProcessId
}

// openProcessHandle opens a handle to pid which is closed when the test completes. Holding the
// handle stops the id from being reused, so processExited on it can't be fooled by an
// unrelated process. Open it before stopping the service.
func openProcessHandle(t *testing.T, pid uint32) windows.Handle {
	t.Helper()
	h, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		t.Fatalf("failed to open process %d: %v", pid, err)
	}
	t.Cleanup(func() {
		if err := windows.CloseHandle(h); err != nil {
			t.Errorf("failed to close handle for process %d: %v", pid, err)
		}
	})
	return h
}

// processExited reports whether the process behind h has exited, without blocking.
func processExited(t *testing.T, h windows.Handle) bool {
	t.Helper()
	event, err := windows.WaitForSingleObject(h, 0)
	if err != nil {
		t.Fatalf("failed to wait on process handle: %v", err)
	}
	switch event {
	case windows.WAIT_OBJECT_0:
		return true
	case uint32(windows.WAIT_TIMEOUT):
		return false
	default:
		t.Fatalf("unexpected wait result %d", event)
		return false
	}
}

// setProcessExitDeadline overrides service.ProcessExitDeadline for the duration of the test.
func setProcessExitDeadline(t *testing.T, d time.Duration) {
	t.Helper()
	original := service2.ProcessExitDeadline
	service2.ProcessExitDeadline = d
	t.Cleanup(func() { service2.ProcessExitDeadline = original })
}

// openSCMHandle opens a handle to the service directly through the SCM, bypassing Service,
// so tests can race controls against it. The handle is closed when the test completes.
func openSCMHandle(t *testing.T, name string) *mgr.Service {
	t.Helper()
	m, err := mgr.Connect()
	if err != nil {
		t.Fatalf("failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	h, err := m.OpenService(name)
	if err != nil {
		t.Fatalf("failed to open service %s: %v", name, err)
	}
	t.Cleanup(func() { h.Close() })
	return h
}

// waitForSCMState polls h until the SCM reports want, failing the test after timeout. The
// interval is short so callers land inside narrow windows, such as a lingering process.
func waitForSCMState(t *testing.T, h *mgr.Service, want svc.State, timeout time.Duration) svc.Status {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		status, err := h.Query()
		if err != nil {
			t.Fatalf("failed to query service: %v", err)
		}
		if status.State == want {
			return status
		}
		if time.Now().After(deadline) {
			t.Fatalf("service did not reach state %d within %s, last state was %d", want, timeout, status.State)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// deleteService marks the service for deletion and waits for the SCM to remove it.
func deleteService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	err = s.Delete()
	s.Close()
	if err != nil && !errors.Is(err, windows.ERROR_SERVICE_MARKED_FOR_DELETE) {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	deadline := time.Now().Add(serviceRemovalTimeout)
	for time.Now().Before(deadline) {
		s, err := m.OpenService(name)
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			return nil
		}
		if err == nil {
			s.Close()
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("service was still present %s after deletion, a handle to it may still be open", serviceRemovalTimeout)
}

// removeLeftoverServices deletes any test services left behind by a previous run which did not clean up.
func removeLeftoverServices() error {
	if err := killTestServiceProcesses(); err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	names, err := m.ListServices()
	m.Disconnect()
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	for _, name := range names {
		if !strings.HasPrefix(name, testServicePrefix) {
			continue
		}
		if err := deleteService(name); err != nil {
			return fmt.Errorf("failed to delete leftover service %s: %w", name, err)
		}
	}
	return nil
}

// killTestServiceProcesses terminates every running instance of the test service binary.
// Once a service has reported stopped the SCM no longer reports its process id, so a
// lingering process can only be found by its image name.
func killTestServiceProcesses() error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return fmt.Errorf("failed to snapshot processes: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	var errs []error
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		if !strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), testServiceExeName) {
			continue
		}
		if err := killProcess(entry.ProcessID); err != nil {
			errs = append(errs, err)
		}
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		errs = append(errs, fmt.Errorf("failed to enumerate processes: %w", err))
	}
	return errors.Join(errs...)
}

func killProcess(pid uint32) error {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_TERMINATE, false, pid)
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to open process %d: %w", pid, err)
	}
	defer windows.CloseHandle(h)

	// Terminating a process that is already exiting returns access denied, the wait below covers that case.
	if err := windows.TerminateProcess(h, 1); err != nil && !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return fmt.Errorf("failed to terminate process %d: %w", pid, err)
	}

	event, err := windows.WaitForSingleObject(h, uint32(processKillTimeout.Milliseconds()))
	if err != nil {
		return fmt.Errorf("failed to wait on process %d: %w", pid, err)
	}
	if event != windows.WAIT_OBJECT_0 {
		return fmt.Errorf("process %d did not exit within %s of being terminated", pid, processKillTimeout)
	}
	return nil
}
