package service

import (
	"fmt"
	"os"
	"strings"

	"github.com/rancher/wins/cmd/server/config"
	"github.com/sirupsen/logrus"
)

var (
	ConfigDirEnvVar      = "CATTLE_WINS_CONFIG_DIR"
	DebugEnvVar          = "CATTLE_WINS_DEBUG"
	AgentStringTLSEnvVar = "STRICT_VERIFY"
)

const (
	defaultConfigFile = "c:/etc/rancher/wins/config"
)

func getConfigPath(path string) string {
	if path != "" {
		return path
	}
	if path = os.Getenv(ConfigDirEnvVar); path != "" {
		return path
	}
	return defaultConfigFile
}

// loadConfig utilizes the ConfigDirEnvVar environment variable and path parameter to load
// a config file located on the host. If both are provided, the path parameter will take
// precedence. If neither are provided, then the defaultConfigFile path is used.
func loadConfig(path string) (*config.Config, error) {
	cfg := config.DefaultConfig()
	err := config.LoadConfig(getConfigPath(path), cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// saveConfig utilizes the ConfigDirEnvVar environment variable and path parameter to save
// a config file on the host. If both are provided, the path parameter will take
// precedence. If neither are provided, then the defaultConfigFile path is used.
func saveConfig(cfg *config.Config, path string) error {
	return config.SaveConfig(getConfigPath(path), cfg)
}

// UpdateConfigFromEnvVars is responsible for updating the rancher-wins config file
// based off of the presence of particular environment variables. The path parameter is used
// to specify the location of the config file on the host. If path is left empty, the defaultConfigFile
// constant is used. The config file will only be updated if a given environment variable is present, and its
// value does not equal the currently set value in the config file. UpdateConfigFromEnvVars returns a boolean
// indicating if the config file has been updated and any errors encountered.
func UpdateConfigFromEnvVars() (bool, error) {
	logrus.Info("Loading config from host")
	path := getConfigPath("")
	cfg, err := loadConfig(path)
	if err != nil {
		return false, fmt.Errorf("failed to load config: %v", err)
	}

	configNeedsUpdate := false
	logrus.Infof("Checking the %s value. This is a boolean flag, expecting 'true' or 'false'", DebugEnvVar)
	if v := os.Getenv(DebugEnvVar); v != "" {
		logrus.Infof("Found value '%s' for %s", v, DebugEnvVar)
		givenBool := strings.ToLower(v) == "true"
		if cfg.Debug != givenBool {
			cfg.Debug = givenBool
			configNeedsUpdate = true
		}
	}

	logrus.Infof("Checking the %s value. This is a boolean flag, expecting 'true' or 'false'", AgentStringTLSEnvVar)
	if v := os.Getenv(AgentStringTLSEnvVar); v != "" {
		logrus.Infof("Found value '%s' for %s", v, AgentStringTLSEnvVar)
		givenBool := strings.ToLower(v) == "true"
		if cfg.AgentStrictTLSMode != givenBool {
			cfg.AgentStrictTLSMode = givenBool
			configNeedsUpdate = true
		}
	}

	// If we haven't made any changes there is no reason to update the config file
	if configNeedsUpdate {
		logrus.Info("Detected a change in configuration, updating config file")
		err = saveConfig(cfg, path)
		if err != nil {
			return configNeedsUpdate, fmt.Errorf("failed to save config: %w", err)
		}
	} else {
		logrus.Info("Did not detect a change in configuration")
	}

	return configNeedsUpdate, nil
}
