package csiproxy

import (
	"archive/tar"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/rancher/wins/pkg/concierge"
	winstls "github.com/rancher/wins/pkg/tls"
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
	tlsCfg      *winstls.Config
	serviceName string
	binaryName  string
	binaryPath  string
	concierge   *concierge.Concierge
}

// New creates a new Proxy struct
func New(cfg *Config, tlsCfg *winstls.Config) (*Proxy, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	config := concierge.Config{
		Args:        []string{"-windows-service", "-log_file=\\etc\\rancher\\wins\\csi-proxy.log", "-logtostderr=false"},
		Description: "Manages the Kubernetes CSI Proxy application.",
		DisplayName: "CSI Proxy",
		EnvVars:     nil,
	}

	service, err := concierge.New(serviceName, filepath.Join(cwd, exeName), &config)
	if err != nil {
		return nil, err
	}

	return &Proxy{
		cfg:         cfg,
		tlsCfg:      tlsCfg,
		serviceName: serviceName,
		binaryName:  exeName,
		binaryPath:  filepath.Join(cwd, exeName),
		concierge:   service,
	}, nil
}

func (p *Proxy) Enable() error {
	ok, err := p.concierge.ServiceExists()
	if err != nil {
		return err
	}
	if !ok {
		if p.tlsCfg != nil && p.tlsCfg.CertFilePath != "" {
			// CSI Proxy does not need the certpool that is returned
			_, err := p.tlsCfg.SetupGenericTLSConfigFromFile()
			if err != nil {

				return err
			}
		}
		logrus.Infof("CSI Proxy is being downloaded.")
		if err := p.download(); err != nil {
			return err
		}
		logrus.Infof("CSI Proxy is being started.")
		if err := p.concierge.Enable(); err != nil {
			return err
		}
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
		CheckRedirect: func(r *http.Request, _ []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	// default to insecure which matches system-agent functionality
	// if a proxy is set with the proper envvars, we will use it
	// as long as the req does not match an entry in no_proxy env var
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, Proxy: http.ProxyFromEnvironment}

	if p.tlsCfg != nil && !*p.tlsCfg.Insecure && p.tlsCfg.CertFilePath != "" {
		transport.TLSClientConfig.InsecureSkipVerify = false
	}

	client.Transport = transport

	defer client.CloseIdleConnections()

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
