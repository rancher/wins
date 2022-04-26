package profilings

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"unsafe"

	"github.com/rancher/wins/pkg/defaults"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"

	"syscall"
)

func StackDump() (err error) {
	err = callStackDump()
	if err != nil {
		logrus.Errorf("[StackDump] failed to call wins stack dump: %v", err)
		return err
	}
	return nil
}

func findWinsProcess() (uint32, error) {
	h, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		panic(err)
	}
	var p windows.ProcessEntry32
	p.Size = uint32(reflect.TypeOf(p).Size())

	for {
		err := windows.Process32Next(h, &p)
		if err != nil {
			logrus.Errorf("[findWinsProcess] error finding next process: %v", err)
			break
		}
		ps := getProcessName(windows.UTF16ToString(p.ExeFile[:]))
		logrus.Debugf("[findWinsProcess] found trimmed process: %v", ps)
		if ps == defaults.WindowsProcessName {
			pid := p.ProcessID
			logrus.Debugf("[findWinsProcess] found process [%s] with pid [%d]", ps, pid)
			return pid, nil
		}
		logrus.Warnf("[findWinsProcess] no process matching [%s] was found", defaults.WindowsProcessName)
	}
	return 0, nil
}

func callStackDump() (err error) {

	winsProcessID, err := findWinsProcess()
	if err != nil {
		return fmt.Errorf("[callStackDump]: error returned when getting wins process id: %v", err)
	}

	event := fmt.Sprintf("Global\\stackdump-%d", winsProcessID)
	ev, _ := windows.UTF16PtrFromString(event)

	// verify that wins is running before trying to send stackdump signal
	if syscall.Signal(syscall.Signal(0)) == 0 {
		logrus.Debugf("[callStackDump] confirmed that wins process %d is running", winsProcessID)
	}

	sd, err := windows.SecurityDescriptorFromString(defaults.PermissionBuiltinAdministratorsAndLocalSystem)
	if err != nil {
		return fmt.Errorf("failed to get security descriptor for debug stackdump event %s: %v", event, err)
	}

	var sa windows.SecurityAttributes
	sa.Length = uint32(unsafe.Sizeof(sa))
	sa.InheritHandle = 1
	sa.SecurityDescriptor = sd

	// attempt to open the existing listen event for wins stack dump
	h, err := windows.OpenEvent(0x1F0003, // EVENT_ALL_ACCESS
		true,
		ev)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)

	if err := windows.SetEvent(h); err != nil {
		return fmt.Errorf("[callStackDump] error setting win32 event: %v", err)
	}

	return windows.ResetEvent(h)
}

func getProcessName(path string) string {
	return strings.TrimRight(filepath.Base(path), ".exe")
}
