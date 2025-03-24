package service

import (
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
)

func getStateTransitionAttempts() int {
	env := os.Getenv("CATTLE_WINS_STATE_TRANSITION_ATTEMPTS")
	if env != "" {
		i, err := strconv.Atoi(env)
		if err != nil {
			logrus.Debugf("failed to cast 'CATTLE_WINDOWS_STATE_TRANSITION_ATTEMPTS' (%s) to an integer, returning default value of %d", env, stateTransitionAttempts)
			return stateTransitionAttempts
		}
		return i
	}
	return stateTransitionAttempts
}

func getStateTransitionDelayInSeconds() time.Duration {
	env := os.Getenv("CATTLE_WINS_STATE_TRANSITION_SECONDS")
	if env != "" {
		i, err := strconv.Atoi(env)
		if err != nil {
			logrus.Debugf("failed to cast 'CATTLE_WINDOWS_STATE_TRANSITION_SECONDS' (%s) to an integer, returning default value of %d", env, stateTransitionDelayInSeconds)
			return stateTransitionDelayInSeconds
		}
		return time.Duration(i)
	}
	return stateTransitionDelayInSeconds
}

func UnorderedSlicesEqual[T comparable](s1 []T, s2 []T) bool {
	if len(s1) != len(s2) {
		return false
	}
	for _, e := range s1 {
		found := false
		for _, e2 := range s2 {
			if e == e2 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func removeAllFromSlice[T comparable](x T, s []T) []T {
	var n []T
	for _, e := range s {
		if e == x {
			continue
		}
		n = append(n, e)
	}
	return n
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
