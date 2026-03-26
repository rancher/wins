package main

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
)

type handler struct{}

// Execute signals SERVICE_RUNNING, then blocks until the SCM sends a stop/shutdown
// control request, at which point it signals SERVICE_STOPPED and returns.
func (h *handler) Execute(
	args []string,
	requests <-chan svc.ChangeRequest,
	status chan<- svc.Status,
) (bool, uint32) {

	const accepted = svc.AcceptStop | svc.AcceptShutdown
	status <- svc.Status{State: svc.Running, Accepts: accepted}

	for req := range requests {
		switch req.Cmd {
		case svc.Stop, svc.Shutdown:
			status <- svc.Status{State: svc.Stopped}
			return false, 0
		}
	}

	return false, 0
}

func main() {
	isService, err := svc.IsWindowsService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to detect service context: %v\n", err)
		os.Exit(1)
	}

	if isService {
		serviceName := "test"

		if len(os.Args) > 1 {
			serviceName = os.Args[1]
		}
		if err := svc.Run(serviceName, &handler{}); err != nil {
			fmt.Fprintf(os.Stderr, "service failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	time.Sleep(15 * time.Minute)
}
