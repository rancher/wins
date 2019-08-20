package apis

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/rancher/wins/pkg/powershells"
	"github.com/sirupsen/logrus"
)

var ppool sync.Map

type process struct {
	id     int
	name   string
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func (p *process) String() string {
	return fmt.Sprintf("%s(%d)", p.name, p.id)
}

func (p *process) wait() error {
	if p == nil {
		return errors.New("nil process")
	}

	proc, err := os.FindProcess(p.id)
	if err != nil {
		return errors.Wrapf(err, "could not find process %s", p)
	}

	_, err = proc.Wait()
	return err
}

func (p *process) kill(ctx context.Context) error {
	if p == nil {
		return errors.New("nil process")
	}

	// remove firewall route
	psb := &powershells.Builder{}
	ps, err := psb.Build()
	if err != nil {
		return errors.Wrapf(err, "could not build firewall mgmt PowerShell")
	}
	removeFirewallRulePsCommand := fmt.Sprintf(`Get-NetFirewallRule -PolicyStore ActiveStore -Name %s-* | ForEach-Object {Remove-NetFirewallRule -Name $_.Name -PolicyStore ActiveStore -ErrorAction Ignore | Out-Null}`, p.name)
	err = ps.ExecuteCommand(ctx, removeFirewallRulePsCommand)
	if err != nil {
		return errors.Wrapf(err, "could not remove firewall rules")
	}

	// kill task
	taskkill := exec.CommandContext(ctx, "taskkill", "/T", "/F", "/PID", strconv.Itoa(p.id))
	taskkill.Run()

	logrus.Debugf("[Process] Killed process %s", p)
	ppool.Delete(p.name)

	return nil
}

func (s *processService) getFromPool(pname string) (*process, error) {
	po, exist := ppool.Load(pname)
	if exist {
		ret, ok := po.(*process)
		if ok {
			logrus.Debugf("[Process] Got pooled process %s", ret)
			return ret, nil
		}

		ppool.Delete(pname)
	}

	return nil, errors.Errorf("could not find process %s", pname)
}

func (s *processService) getFromHost(pname string) (*process, error) {
	// take windows processes snapshot
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, errors.Wrapf(err, "could not take a snapshot for Windows processes")
	}
	defer syscall.CloseHandle(snapshot)

	// start the iterator
	var procEntry syscall.ProcessEntry32
	procEntry.Size = uint32(unsafe.Sizeof(procEntry))
	if err := syscall.Process32First(snapshot, &procEntry); err != nil {
		return nil, errors.Wrapf(err, "could not start the iterator for Windows processes")
	}

	// process iterating
	logrus.Debug("[Process] Iterating process:")
	var pid int
	for err == nil {
		name := getProcessName(syscall.UTF16ToString(procEntry.ExeFile[:]))
		if pname == name {
			logrus.Debugf("[Process] \tname: %s\t\tpid: %d\tppid: %d", pname, procEntry.ProcessID, procEntry.ParentProcessID)
			if pid == 0 || pid != int(procEntry.ParentProcessID) {
				pid = int(procEntry.ProcessID)
			}
		}

		err = syscall.Process32Next(snapshot, &procEntry)
	}
	if pid == 0 {
		return nil, nil
	}

	p := &process{
		id:   pid,
		name: pname,
	}

	logrus.Debugf("[Process] Got existing process %s", p)

	return p, nil
}

func (s *processService) create(ctx context.Context, path string, dir string, args []string, envs []string, fwrules string) (*process, error) {
	pname := getProcessName(path)

	pInHost, err := s.getFromHost(pname)
	if err != nil {
		return nil, err
	}
	if pInHost != nil {
		// detect the process is running via wins
		if pInPool, _ := s.getFromPool(pname); pInPool != nil {
			return nil, errors.Wrapf(err, "could not run duplicate process")
		}

		// recreate the process to gain the std handler
		logrus.Warnf("[Process] Found stale process %s, try to recreate a new process", pInHost)
		if err := pInHost.kill(ctx); err != nil {
			return nil, errors.Wrapf(err, "could not kill stale process")
		}
	}

	// create firewall rules if needed
	if fwrules != "" {
		psb := &powershells.Builder{}
		ps, err := psb.Build()
		if err != nil {
			return nil, errors.Wrapf(err, "could not build firewall mgmt PowerShell")
		}
		newFirewallRulePsCommand := fmt.Sprintf(`"%s" -split ' ' | ForEach-Object {$ruleMd = $_ -split '-'; $ruleName = "%s-$_"; New-NetFirewallRule -Name $ruleName -DisplayName $ruleName -Action Allow -Protocol $ruleMd[0] -LocalPort $ruleMd[1] -Enabled True -PolicyStore ActiveStore -ErrorAction Ignore | Out-Null}`, fwrules, pname)
		err = ps.ExecuteCommand(ctx, newFirewallRulePsCommand)
		if err != nil {
			return nil, errors.Wrapf(err, "could not create process firewall rules")
		}
	}

	// create command
	c := exec.Command(path, args...)
	c.Dir = dir
	c.Env = append(os.Environ(), envs...)
	c.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x00000010, // CREATE_NEW_CONSOLE: https://docs.microsoft.com/en-us/windows/win32/procthread/process-creation-flags
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, errors.Wrapf(err, "could not create process stdout stream")
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		return nil, errors.Wrapf(err, "could not create process stderr stream")
	}

	if err := c.Start(); err != nil {
		return nil, err
	}

	// pool process
	p := &process{
		id:     c.Process.Pid,
		name:   pname,
		stdout: stdout,
		stderr: stderr,
	}
	ppool.Store(p.name, p)
	logrus.Debugf("[Process] Created process %s", p)

	return p, nil
}

func getProcessName(path string) string {
	return strings.TrimRight(filepath.Base(path), ".exe")
}
