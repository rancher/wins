package profilings

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"github.com/rancher/wins/pkg/converters"
	"github.com/rancher/wins/pkg/defaults"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

// DumpStacks returns up to (1 << 15) bytes of the current processes stack trace as a string
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

// DumpStacksToFile dumps the stack trace of all current goroutines to the file path provided
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

// SetupDumpStacks creates a goroutine that listens for any signals passed to the Win32 event stackdump-{pid}
// that is defined on a Global level; each time a signal is detected to this event, it will dump the a stack
// trace across all goroutines (up to 1 << 15 bytes) to a file within the Windows machine's temp directory.
// By default, this event can only be signaled by built-in administrators and the local system.
func SetupDumpStacks(serviceName string, pid int, cwd string) {
	if serviceName == "" {
		return
	}

	// Windows does not support signals like *nix systems. So instead of
	// trapping on SIGUSR1 to dump stacks, we wait on a Win32 event to be
	// signaled. ACL'd to builtin administrators and local system
	event := fmt.Sprintf("Global\\stackdump-%d", pid)
	ev, _ := windows.UTF16PtrFromString(event)
	sd, err := windows.SecurityDescriptorFromString(defaults.PermissionBuiltinAdministratorsAndLocalSystem)
	if err != nil {
		logrus.Errorf("Failed to get security descriptor for debug stackdump event %s: %v", event, err)
		return
	}
	var sa windows.SecurityAttributes
	sa.Length = uint32(unsafe.Sizeof(sa))
	sa.InheritHandle = 1
	sa.SecurityDescriptor = sd
	h, err := windows.CreateEvent(&sa, 0, 0, ev)
	if h == 0 || err != nil {
		logrus.Errorf("Failed to create debug stackdump event %s: %v", event, err)
		return
	}

	go func() {
		logrus.Infof("[SetupDumpStacks] stackdump feature successfully initialized - waiting for signal at %s", event)
		for {
			windows.WaitForSingleObject(h, windows.INFINITE)
			fileLoc := filepath.Join(cwd, fmt.Sprintf("%s.%d.stacks.log", serviceName, pid))
			logrus.Debugf("SetupStackDumps: stackDump location will be [%s]", fileLoc)
			DumpStacksToFile(fileLoc)
		}
	}()
}
