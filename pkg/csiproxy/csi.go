package csiproxy

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	exeName     = "csi-proxy.exe"
	serviceName = "csiproxy"
)

// Config is the CSI Proxy config settings
type Config struct {
	URL         string `yaml:"url" json:"url"`
	Version     string `yaml:"version" json:"version"`
	KubeletPath string `yaml:"kubeletPath" json:"kubeletPath"`
}

// Validate ensures that the configuration for CSI Proxy is correct if provided.
func (c *Config) validate() error {
	if strings.TrimSpace(c.URL) == "" {
		return errors.New("CSI Proxy URL cannot be empty")
	}

	if strings.TrimSpace(c.Version) == "" {
		return errors.New("CSI Proxy version cannot be empty")
	}

	if strings.TrimSpace(c.KubeletPath) == "" {
		return errors.New("kubelet path cannot be empty")
	}
	return nil
}

// Proxy is for creating and retrieving the Windows Service
type Proxy struct {
	cfg         *Config
	serviceName string
	binaryName  string
	binaryPath  string
}

// New creates a new Proxy struct
func New(cfg *Config) (*Proxy, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return &Proxy{
		cfg:         cfg,
		serviceName: serviceName,
		binaryName:  exeName,
		binaryPath:  filepath.Join(cwd, exeName),
	}, nil
}

func (p *Proxy) Enable() error {
	var service *mgr.Service
	ok, err := p.serviceExists()
	if err != nil {
		return err
	}
	if !ok {
		if err := p.download(); err != nil {
			return err
		}
		if service, err = p.createService(); err != nil {
			return err
		}
		return service.Start()
	}

	if service, err = p.getService(); err != nil {
		return err
	}

	return nil
}

func (p *Proxy) Disable() error {
	var service *mgr.Service
	ok, err := p.serviceExists()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if service, err = p.getService(); err != nil {
		return err
	}

	if _, err := service.Control(svc.Stop); err != nil {
		return err
	}
	return nil
}

// download retrieves the CSI Proxy executable from the config settings.
func (p *Proxy) download() error {
	file, err := os.Create(p.binaryPath)
	if err != nil {
		return err
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}
	resp, err := client.Get(fmt.Sprintf(p.cfg.URL, p.cfg.Version))
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer func(gz *gzip.Reader) {
		_ = gz.Close()
	}(gz)

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if strings.Contains(hdr.Name, p.binaryName) {
			if _, err := io.Copy(file, tr); err != nil {
				return err
			}
		}
	}

	return nil
}

// createService configures the Windows service correctly, returning the service.
func (p *Proxy) createService() (*mgr.Service, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, errors.Wrap(err, "could not open SCM")
	}
	defer func(m *mgr.Mgr) {
		_ = m.Disconnect()
	}(m)

	service, err := m.CreateService(p.serviceName, p.binaryPath, mgr.Config{
		ServiceType:    windows.SERVICE_WIN32_OWN_PROCESS,
		StartType:      mgr.StartAutomatic,
		ErrorControl:   mgr.ErrorNormal,
		BinaryPathName: p.binaryPath,
		DisplayName:    "CSI Proxy",
		Description:    "Manages the Kubernetes CSI Proxy application.",
	}, "-windows-service", "-log_file=\\etc\\rancher\\wins\\csi-proxy.log", "-logtostderr=false")

	if err != nil {
		return nil, err
	}

	recoveryActions := []mgr.RecoveryAction{
		{
			Type:  mgr.ServiceRestart,
			Delay: 10000,
		},
	}

	return service, service.SetRecoveryActions(recoveryActions, 0)
}

// serviceExists retrieves the Windows service if exists.
func (p *Proxy) serviceExists() (bool, error) {
	m, err := mgr.Connect()
	if err != nil {
		return false, errors.Wrap(err, "could not open SCM")
	}
	defer func(m *mgr.Mgr) {
		_ = m.Disconnect()
	}(m)

	services, err := m.ListServices()
	if err != nil {
		return false, errors.Wrap(err, "could list services.")
	}

	for _, service := range services {
		if service == p.serviceName {
			return true, nil
		}
	}

	return false, nil
}

// getService retrieves the Windows service.
func (p *Proxy) getService() (*mgr.Service, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, errors.Wrap(err, "could not open SCM")
	}
	defer func(m *mgr.Mgr) {
		_ = m.Disconnect()
	}(m)

	service, err := m.OpenService(p.serviceName)
	if err != nil {
		return nil, nil
	}

	return service, nil
}
