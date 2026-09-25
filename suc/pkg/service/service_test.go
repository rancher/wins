//go:build windows

package service

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

func TestStartWithRetry(t *testing.T) {
	t.Run("retries already running error while stopped without sleeping in tests", func(t *testing.T) {
		startCalls := 0
		waitCalls := 0
		sleepCalls := 0
		stateIndex := 0
		states := []svc.State{svc.Stopped, svc.StartPending}

		err := startWithRetryBackoff(
			"rancher-wins",
			func() error {
				startCalls++
				if startCalls == 1 {
					return windows.ERROR_SERVICE_ALREADY_RUNNING
				}
				return nil
			},
			func() (svc.State, error) {
				state := states[stateIndex]
				if stateIndex < len(states)-1 {
					stateIndex++
				}
				return state, nil
			},
			func() error {
				waitCalls++
				return nil
			},
			0,
			2,
			func(time.Duration) {
				sleepCalls++
			},
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if startCalls != 2 {
			t.Fatalf("expected 2 start attempts, got %d", startCalls)
		}
		if waitCalls != 1 {
			t.Fatalf("expected waitForRunning to be called once, got %d", waitCalls)
		}
		if sleepCalls != 1 {
			t.Fatalf("expected 1 sleep call, got %d", sleepCalls)
		}
	})

	t.Run("waits when service is already starting", func(t *testing.T) {
		startCalls := 0
		waitCalls := 0

		err := startWithRetry(
			"rancher-wins",
			func() error {
				startCalls++
				return windows.ERROR_SERVICE_ALREADY_RUNNING
			},
			func() (svc.State, error) {
				return svc.StartPending, nil
			},
			func() error {
				waitCalls++
				return nil
			},
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if startCalls != 1 {
			t.Fatalf("expected 1 start attempt, got %d", startCalls)
		}
		if waitCalls != 1 {
			t.Fatalf("expected waitForRunning to be called once, got %d", waitCalls)
		}
	})

	t.Run("returns non transient start error", func(t *testing.T) {
		expectedErr := errors.New("boom")

		err := startWithRetry(
			"rancher-wins",
			func() error {
				return expectedErr
			},
			func() (svc.State, error) {
				t.Fatal("getState should not be called for non-transient errors")
				return svc.Stopped, nil
			},
			func() error {
				t.Fatal("waitForRunning should not be called for non-transient errors")
				return nil
			},
		)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected wrapped error %v, got %v", expectedErr, err)
		}
	})
}
