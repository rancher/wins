package service

import (
	"golang.org/x/sys/windows/svc"
)

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
