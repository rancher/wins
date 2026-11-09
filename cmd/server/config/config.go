package config

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/rancher/system-agent/pkg/config"
	"github.com/rancher/wins/pkg/csiproxy"
	wintls "github.com/rancher/wins/pkg/tls"
	"sigs.k8s.io/yaml"
)

func DefaultConfig() *Config {
	return &Config{
		AgentStrictTLSMode: false,
	}
}

type Config struct {
	Debug              bool                `yaml:"debug" json:"debug"`
	SystemAgent        *config.AgentConfig `yaml:"systemagent" json:"systemagent,omitempty"`
	AgentStrictTLSMode bool                `yaml:"agentStrictTLSMode" json:"agentStrictTLSMode"`
	CSIProxy           *csiproxy.Config    `yaml:"csi-proxy" json:"csi-proxy,omitempty"`
	TLSConfig          *wintls.Config      `yaml:"tls-config" json:"tls-config,omitempty"`
}

func (c *Config) Validate() error {
	return nil
}

func LoadConfig(path string, v *Config) error {
	if v == nil {
		return errors.New("config cannot be nil")
	}

	stat, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "could not load config")
		}
		return nil
	} else if stat.IsDir() {
		return errors.New("could not load config from directory")
	}

	if err := DecodeConfig(path, v); err != nil {
		return errors.Wrap(err, "could not decode config")
	}

	return v.Validate()
}

func SaveConfig(path string, v *Config) error {
	if v == nil {
		return errors.New("config cannot be nil")
	}

	yml, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("could not marshal provided config: %w", err)
	}

	return os.WriteFile(path, yml, os.ModePerm)
}

func DecodeConfig(path string, v *Config) error {
	bs, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(bs, v)
}
