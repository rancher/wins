package concierge

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

type Config struct {
	Args        []string
	Description string
	DisplayName string
	EnvVars     []string
	registryKey string
}

type Concierge struct {
	name string
	path string
	cfg  *Config
}

// New creates a new Concierge for managing a Windows Service.
func New(name, path string, cfg *Config) (*Concierge, error) {
	if name == "" {
		return nil, errors.New("name isn't set and can't be empty")
	}

	if path == "" {
		return nil, errors.New("path isn't set and can't be empty")
	}
	if cfg == nil {
		return nil, errors.New("cfg is nil, please provide at least an empty config")
	}

	cfg.registryKey = fmt.Sprintf(`SYSTEM\CurrentControlSet\Services\%s`, name)

	return &Concierge{
		name: name,
		path: path,
		cfg:  cfg,
	}, nil
}

// Enable will start the Windows Service. If the service doesn't exist it will create it.
func (c *Concierge) Enable() error {
	var service *mgr.Service
	ok, err := c.ServiceExists()
	if err != nil {
		return errors.Wrap(err, "error checking if the service exists")
	}
	if !ok {
		if err = c.CreateService(); err != nil {
			return errors.Wrap(err, "error creating the service")
		}
	}

	if service, err = c.fetchService(); err != nil {
		return errors.Wrap(err, "error fetching the service")
	}
	defer service.Close()

	return service.Start()
}

// Disable will stop the Windows Service.
func (c *Concierge) Disable() error {
	var service *mgr.Service
	ok, err := c.ServiceExists()
	if err != nil {
		return errors.Wrap(err, "error checking if the service exists")
	}
	if !ok {
		return errors.Errorf("service %s not found", c.path)
	}

	if service, err = c.fetchService(); err != nil {
		return errors.Wrap(err, "error fetching the service")
	}
	defer service.Close()

	if _, err := service.Control(svc.Stop); err != nil {
		return errors.Wrap(err, "error stopping the service")
	}
	return nil
}

// CreateService configures the Windows service correctly, returning the service.
func (c *Concierge) CreateService() error {
	m, err := mgr.Connect()
	if err != nil {
		return errors.Wrap(err, "could not open SCM")
	}
	defer m.Disconnect()

	service, err := m.CreateService(c.name, c.path, mgr.Config{
		ServiceType:    windows.SERVICE_WIN32_OWN_PROCESS,
		StartType:      mgr.StartAutomatic,
		ErrorControl:   mgr.ErrorNormal,
		BinaryPathName: c.path,
		Description:    c.cfg.Description,
		DisplayName:    c.cfg.DisplayName,
	}, c.cfg.Args...)

	defer service.Close()

	if err != nil {
		return errors.Wrap(err, "error creating the service")
	}

	recoveryActions := []mgr.RecoveryAction{
		{
			Type:  mgr.ServiceRestart,
			Delay: 10000,
		},
	}
	if err := c.registerEnvVars(); err != nil {
		return err
	}

	return service.SetRecoveryActions(recoveryActions, 0)
}

// Delete removes the service and any registry keys.
func (c *Concierge) Delete() error {
	var service *mgr.Service
	ok, err := c.ServiceExists()
	if err != nil {
		return errors.Wrap(err, "error checking if the service exists")
	}
	if !ok {
		return errors.Errorf("service %s not found", c.path)
	}

	if service, err = c.fetchService(); err != nil {
		return errors.Wrap(err, "error fetching the service")
	}
	defer service.Close()

	return service.Delete()
}

// ServiceExists retrieves the Windows service if exists.
func (c *Concierge) ServiceExists() (bool, error) {
	m, err := mgr.Connect()
	if err != nil {
		return false, errors.Wrap(err, "could not open SCM")
	}
	defer func(m *mgr.Mgr) {
		_ = m.Disconnect()
	}(m)

	services, err := m.ListServices()
	if err != nil {
		return false, errors.Wrap(err, "could not list services")
	}

	for _, service := range services {
		if service == c.name {
			return true, nil
		}
	}

	return false, nil
}

// State gets the state of the service. Examples are stopped, running, etc.
func (c *Concierge) State() (svc.State, error) {
	service, err := c.fetchService()
	if err != nil {
		return svc.State(windows.SERVICE_NO_CHANGE), errors.Wrap(err, "error opening the service")
	}
	defer service.Close()

	status, err := service.Query()
	if err != nil {
		return svc.State(windows.SERVICE_NO_CHANGE), errors.Wrap(err, "error querying the service")
	}
	return status.State, nil
}

// fetchService retrieves the Windows service.
func (c *Concierge) fetchService() (*mgr.Service, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, errors.Wrap(err, "could not open SCM")
	}
	defer m.Disconnect()

	service, err := m.OpenService(c.name)
	if err != nil {
		return nil, errors.Wrap(err, "could not open service")
	}

	return service, nil
}

// registerEnvVars creates a registry key for the service to set environment variables.
func (c *Concierge) registerEnvVars() error {
	if len(c.cfg.EnvVars) == 0 {
		logrus.Infof("skipping environment variable configuration for %s, none are provided", c.name)
		return nil
	}

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, c.cfg.registryKey, registry.WRITE)
	if err != nil {
		return errors.Wrap(err, "error opening registry key")
	}
	defer k.Close()

	return k.SetStringsValue("Environment", c.cfg.EnvVars)
}
