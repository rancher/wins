package profilings

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"github.com/Microsoft/go-winio"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/rancher/wins/pkg/converters"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

func DumpStacks() string {
	var (
		buf       []byte
		stackSize int
		bufferLen = 1 << 15
	)
	for stackSize == len(buf) {
		buf = make([]byte, bufferLen)
		stackSize = runtime.Stack(buf, true)
		bufferLen *= 2
	}
	buf = buf[:stackSize]
	return converters.UnsafeBytesToString(buf)
}

func DumpStacksToFile(filename string) {
	if filename == "" {
		return
	}

	stacksdump := DumpStacks()

	f, err := os.Create(filename)
	if err != nil {
		logrus.Errorf("Failed to dump stacks to %s", filename)
		return
	}
	defer f.Close()
	f.WriteString(stacksdump)
}

func SetupDumpStacks(serviceName string, pid int) {
	if serviceName == "" {
		return
	}

	// Windows does not support signals like *nix systems. So instead of
	// trapping on SIGUSR1 to dump stacks, we wait on a Win32 event to be
	// signaled. ACL'd to builtin administrators and local system
	event := fmt.Sprintf("Global\\stackdump-%d", pid)
	ev, _ := windows.UTF16PtrFromString(event)
	sd, err := winio.SddlToSecurityDescriptor(defaults.PermissionBuiltinAdministratorsAndLocalSystem)
	if err != nil {
		logrus.Errorf("Failed to get security descriptor for debug stackdump event %s: %v", event, err)
		return
	}
	var sa windows.SecurityAttributes
	sa.Length = uint32(unsafe.Sizeof(sa))
	sa.InheritHandle = 1
	sa.SecurityDescriptor = uintptr(unsafe.Pointer(&sd[0]))
	h, err := windows.CreateEvent(&sa, 0, 0, ev)
	if h == 0 || err != nil {
		logrus.Errorf("Failed to create debug stackdump event %s: %v", event, err)
		return
	}
	go func() {
		logrus.Infof("Stackdump - waiting signal at %s", event)
		for {
			windows.WaitForSingleObject(h, windows.INFINITE)
			DumpStacksToFile(filepath.Join(os.TempDir(), fmt.Sprintf("%s.%d.stacks.log", serviceName, pid)))
		}
	}()
}
