package service

import (
	"golang.org/x/sys/windows/svc"
)

func SlicesMatch(s1 []string, s2 []string) bool {
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

func removeFromSlice(x string, s []string) []string {
	var n []string
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
