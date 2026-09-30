package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
)

// Flags must precede the positional service name, as the flag package stops
// parsing at the first non-flag argument.
var (
	linger       = flag.Duration("linger", 0, "how long the process keeps running after the service reports stopped")
	stopPending  = flag.Duration("stop-pending", 0, "how long the service reports stop pending before reporting stopped")
	startPending = flag.Duration("start-pending", 0, "how long the service reports start pending before reporting running")
)

type handler struct{}

func (h *handler) Execute(
	args []string,
	requests <-chan svc.ChangeRequest,
	status chan<- svc.Status,
) (bool, uint32) {

	if *startPending > 0 {
		// No controls are accepted while start pending, so the SCM rejects them with ERROR_SERVICE_CANNOT_ACCEPT_CTRL.
		status <- svc.Status{State: svc.StartPending, WaitHint: uint32((*startPending + 5*time.Second).Milliseconds())}
		time.Sleep(*startPending)
	}

	const accepted = svc.AcceptStop | svc.AcceptShutdown
	status <- svc.Status{State: svc.Running, Accepts: accepted}

	for req := range requests {
		switch req.Cmd {
		case svc.Interrogate:
			status <- req.CurrentStatus
		case svc.Stop, svc.Shutdown:
			if *stopPending > 0 {
				// The wait hint must exceed the pending duration, otherwise the SCM may treat the service as hung.
				status <- svc.Status{State: svc.StopPending, WaitHint: uint32((*stopPending + 5*time.Second).Milliseconds())}
				time.Sleep(*stopPending)
			}
			status <- svc.Status{State: svc.Stopped}
			return false, 0
		}
	}

	return false, 0
}

func main() {
	flag.Parse()

	isService, err := svc.IsWindowsService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to detect service context: %v\n", err)
		os.Exit(1)
	}

	if isService {
		serviceName := "test"

		if flag.NArg() > 0 {
			serviceName = flag.Arg(0)
		}
		if err := svc.Run(serviceName, &handler{}); err != nil {
			fmt.Fprintf(os.Stderr, "service failed: %v\n", err)
			os.Exit(1)
		}

		// The SCM has already been told the service is stopped, so this keeps the
		// backing process alive past the state transition.
		time.Sleep(*linger)
		return
	}

	time.Sleep(15 * time.Minute)
}
