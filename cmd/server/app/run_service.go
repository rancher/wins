package app

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/rancher/wins/pkg/apis"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/rancher/wins/pkg/logs"
	"github.com/rancher/wins/pkg/paths"
	"github.com/rancher/wins/pkg/profilings"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

func registerService() error {
	// confirm wins binary path
	binaryPath, err := paths.GetBinaryPath(os.Args[0])
	if err != nil {
		return errors.Wrapf(err, "could not get binary")
	}

	// open SCM
	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "could not open SCM")
	}
	defer m.Disconnect()

	// if the service can be opened that means it was registered
	w, err := m.OpenService(defaults.WindowsServiceName)
	if err == nil {
		status, err := w.Query()
		for err == nil {
			if status.State == svc.Stopped {
				break
			}

			logrus.Warnf("Service is not stopped, going to stop it")
			status, err = w.Control(svc.Stop)
			time.Sleep(3 * time.Second)
		}
		if err != nil {
			return errors.Wrap(err, "could not query status")
		}

		// drop this service
		err = w.Delete()
		if err != nil {
			w.Close()
			return errors.Wrap(err, "could not delete")
		}
		w.Close()

		// wait a while
		time.Sleep(3 * time.Second)
	}

	// join server run arguments
	var args []string
	// wins srv/server app run
	for _, arg := range os.Args[1:] {
		if strings.HasSuffix(arg, "register") {
			continue
		}
		args = append(args, arg)
	}

	// create a new service inst
	w, err = m.CreateService(
		defaults.WindowsServiceName,
		binaryPath,
		mgr.Config{
			ServiceType:    windows.SERVICE_WIN32_OWN_PROCESS,
			StartType:      mgr.StartAutomatic,
			ErrorControl:   mgr.ErrorNormal,
			DisplayName:    defaults.WindowsServiceDisplayName,
			BinaryPathName: binaryPath,
		},
		args...,
	)
	if err != nil {
		return errors.Wrap(err, "could not create")
	}
	defer w.Close()

	// see http://stackoverflow.com/questions/35151052/how-do-i-configure-failure-actions-of-a-windows-service-written-in-go
	// using failure action to control the restart after upgrading
	const (
		scActionNone    = 0
		scActionRestart = 1

		serviceConfigFailureActions = 2
	)
	type serviceFailureActions struct {
		ResetPeriod  uint32
		RebootMsg    *uint16
		Command      *uint16
		ActionsCount uint32
		Actions      uintptr
	}
	type scAction struct {
		Type  uint32
		Delay uint32
	}
	t := []scAction{
		{Type: scActionRestart, Delay: uint32(5 * time.Second / time.Millisecond)},
		{Type: scActionRestart, Delay: uint32(10 * time.Second / time.Millisecond)},
		{Type: scActionRestart, Delay: uint32(15 * time.Second / time.Millisecond)},
		{Type: scActionNone,},
	}
	lpInfo := serviceFailureActions{ResetPeriod: uint32(5 * time.Minute / time.Second), ActionsCount: uint32(len(t)), Actions: uintptr(unsafe.Pointer(&t[0]))}
	err = windows.ChangeServiceConfig2(w.Handle, serviceConfigFailureActions, (*byte)(unsafe.Pointer(&lpInfo)))
	if err != nil {
		return errors.Wrap(err, "could not add failure action")
	}

	// create event log
	err = eventlog.InstallAsEventCreate(defaults.WindowsServiceName, eventlog.Info|eventlog.Warning|eventlog.Error)
	if err != nil {
		if strings.HasSuffix(err.Error(), "registry key already exists") {
			return nil
		}
		return errors.Wrap(err, "could not create event log")
	}

	return nil
}

func unregisterService() error {
	// open SCM
	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "could not open SCM")
	}
	defer m.Disconnect()

	w, err := m.OpenService(defaults.WindowsServiceName)
	if err != nil {
		return errors.Wrap(err, "service hasn't been registered")
	}
	defer w.Close()

	// if the service can be opened that means it was registered
	eventlog.Remove(defaults.WindowsServiceName)

	err = w.Delete()
	if err != nil {
		return errors.Wrapf(err, "could not delete")
	}

	return nil
}

func runService(ctx context.Context, server *apis.Server) error {
	// process windows service
	run := debug.Run
	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		return err
	}
	if !isInteractive {
		run = svc.Run

		logrus.SetOutput(ioutil.Discard)

		// ETW tracing
		if hook, err := logs.NewEtwProviderHook(defaults.WindowsServiceName); err != nil {
			return errors.Wrapf(err, "could not create ETW provider logrus hook")
		} else {
			logrus.AddHook(hook)
		}
		if hook, err := logs.NewEventLogHook(defaults.WindowsServiceName); err != nil {
			return errors.Wrapf(err, "could not create eventlog logrus hook")
		} else {
			logrus.AddHook(hook)
		}

		// Stack dump
		profilings.SetupDumpStacks(defaults.WindowsServiceName, os.Getpid())
	}

	h := &serviceHandler{
		ctx:   ctx,
		doneC: make(chan struct{}),
		errC:  make(chan error),
		srv:   server,
	}
	go func() {
		h.errC <- run(defaults.WindowsServiceName, h)
	}()

	for {
		select {
		case err := <-h.errC:
			if err != nil {
				return err
			}
		case <-h.doneC:
			return nil
		}
	}
}

type serviceHandler struct {
	ctx   context.Context
	doneC chan struct{}
	errC  chan error
	srv   *apis.Server
}

func (h *serviceHandler) Execute(_ []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	s <- svc.Status{State: svc.StartPending, Accepts: 0}

	// start wins server
	ctx, cancel := context.WithCancel(h.ctx)
	defer cancel()
	go func() {
		h.errC <- h.srv.Serve(ctx)
	}()

	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown | svc.Accepted(windows.SERVICE_ACCEPT_PARAMCHANGE)}

Loop:
	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			s <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			s <- svc.Status{State: svc.StopPending, Accepts: 0}
			// stop wins server
			h.srv.Close()
			break Loop
		}
	}

	close(h.doneC)
	return false, 0
}
