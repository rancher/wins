package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/rancher/system-agent/pkg/config"
	"github.com/rancher/wins/pkg/csiproxy"
	"github.com/rancher/wins/pkg/defaults"
	wintls "github.com/rancher/wins/pkg/tls"
)

func DefaultConfig() *Config {
	return &Config{
		Listen: defaults.NamedPipeName,
		Proxy:  defaults.ProxyPipeName,
		WhiteList: WhiteListConfig{
			ProcessPaths: []string{},
			ProxyPorts:   []int{},
		},
		Upgrade: UpgradeConfig{
			Mode:         "watching",
			WatchingPath: defaults.UpgradeWatchingPath,
		},
		AgentStrictTLSMode: false,
	}
}

type Config struct {
	Debug              bool                `yaml:"debug" json:"debug"`
	Listen             string              `yaml:"listen" json:"listen"`
	Proxy              string              `yaml:"proxy" json:"proxy"`
	WhiteList          WhiteListConfig     `yaml:"white_list" json:"white_list"`
	Upgrade            UpgradeConfig       `yaml:"upgrade" json:"upgrade"`
	SystemAgent        *config.AgentConfig `yaml:"systemagent" json:"systemagent"`
	AgentStrictTLSMode bool                `yaml:"agentStrictTLSMode" json:"agentStrictTLSMode"`
	CSIProxy           *csiproxy.Config    `yaml:"csi-proxy" json:"csi-proxy"`
	TLSConfig          *wintls.Config      `yaml:"tls-config" json:"tls-config"`
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Listen) == "" {
		return errors.New("[Validate] listen cannot be blank")
	}

	// validate white list field
	if err := c.WhiteList.Validate(); err != nil {
		return errors.Wrap(err, "[Validate] failed to validate white list field")
	}

	// validate upgrade field
	if err := c.Upgrade.Validate(); err != nil {
		return errors.Wrap(err, "[Validate] failed to validate upgrade field")
	}

	return nil
}

type WhiteListConfig struct {
	ProcessPaths []string `yaml:"process_paths" json:"processPaths"`
	ProxyPorts   []int    `yaml:"proxy_ports" json:"proxyPorts"`
}

func (c *WhiteListConfig) Validate() error {
	// process path
	for _, processPath := range c.ProcessPaths {
		if strings.TrimSpace(processPath) == "" {
			return errors.New("could not accept blank path as process white list")
		}
	}
	for _, proxyPort := range c.ProxyPorts {
		if proxyPort < 0 || proxyPort > 0xFFFF {
			return errors.New("could not accept invalid port number in proxy ports")
		}
	}
	return nil
}

type UpgradeConfig struct {
	Mode         string `yaml:"mode" json:"mode"`
	WatchingPath string `yaml:"watching_path" json:"watchingPath"`
}

func (c *UpgradeConfig) Validate() error {
	switch strings.TrimSpace(c.Mode) {
	case "watching":
		if strings.TrimSpace(c.WatchingPath) == "" {
			return errors.New("could not accept blank path as watching path")
		}
	case "none", "":
	default:
		return errors.Errorf("could not accept %q as upgrade mode", c.Mode)
	}

	return nil
}

func (c *UpgradeConfig) IsWatchingMode() bool {
	return c.Mode == "watching"
}

func LoadConfig(path string, v *Config) error {
	if v == nil {
		return errors.New("config cannot not be nil")
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
		return fmt.Errorf("config cannot be nil")
	}

	yml, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("could not marshal provided config")
	}

	return os.WriteFile(path, yml, os.ModePerm)
}

func DecodeConfig(path string, v *Config) error {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(bs, v)
}
