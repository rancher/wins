package service

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rancher/wins/cmd/server/config"
	"github.com/sirupsen/logrus"
)

var (
	ConfigDirEnvVar      = envVar{varName: "wins_config_dir"}
	DebugEnvVar          = envVar{varName: "debug"}
	AgentStringTLSEnvVar = envVar{varName: "strict_verify"}
)

const (
	defaultConfigFile = "C:/etc/rancher/wins/config"
)

type envVar struct {
	varName string
}

// get returns the value of the environment variable
// using os.Getenv. The full name is passed to os.Getenv
// (i.e. CATTLE_WINS_*)
func (e *envVar) get() string {
	return os.Getenv(e.name())
}

// getSimple returns the value of the environment variable
// using os.Getenv. The simple name is passed to os.Getenv
// (i.e. there is no 'CATTLE_WINS_' prefix)
func (e *envVar) getSimple() string {
	return os.Getenv(e.simpleName())
}

// name returns the name of the environment variable, including
// a 'CATTLE_WINS_' prefix
func (e *envVar) name() string {
	return fmt.Sprintf("CATTLE_WINS_%s", strings.ToUpper(e.varName))
}

// simpleName returns the name of the environment variable, without
// including the 'CATTLE_WINS_' prefix
func (e *envVar) simpleName() string {
	return strings.ToUpper(e.varName)
}

// loadConfig utilizes the ConfigDirEnvVar environment variable and path parameter to load
// a config file located on the host. If both are provided, the path parameter will take
// precedence. If neither are provided, then the defaultConfigFile path is used.
func loadConfig(path string) (*config.Config, error) {
	configDirEnv := os.Getenv(ConfigDirEnvVar.get())
	if path == "" && configDirEnv != "" {
		path = configDirEnv
	}
	if path == "" {
		path = defaultConfigFile
	}

	cfg := config.DefaultConfig()
	err := config.LoadConfig(path, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadConfig utilizes the ConfigDirEnvVar environment variable and path parameter to save
// a config file on the host. If both are provided, the path parameter will take
// precedence. If neither are provided, then the defaultConfigFile path is used.
func saveConfig(cfg *config.Config, path string) error {
	configDirEnv := os.Getenv(ConfigDirEnvVar.get())
	if path == "" && configDirEnv != "" {
		path = configDirEnv
	}
	if path == "" {
		path = defaultConfigFile
	}

	return config.SaveConfig(path, cfg)
}

// UpdateConfigFromEnvVars is responsible for updating the rancher-wins config file
// based off of the presence of particular environment variables. The path parameter is used
// to specify the location of the config file on the host. If path is left empty, the defaultConfigFile
// constant is used. The config file will only be updated if a given environment variable is present, and its
// value does not equal the currently set value in the config file. UpdateConfigFromEnvVars returns a boolean
// indicating if the config file has been updated and any errors encountered.
func UpdateConfigFromEnvVars(path string) (bool, error) {
	logrus.Infof("Loading config from host")
	cfg, err := loadConfig(path)
	if err != nil {
		return false, fmt.Errorf("failed to load config: %v", err)
	}

	configNeedsUpdate := false
	logrus.Infof("Checking for %s value. This is a boolean flag, expecting 'true' or 'false'", DebugEnvVar.name())
	if v := DebugEnvVar.get(); v != "" {
		logrus.Infof("Found value '%s' for %s", v, DebugEnvVar.name())
		givenBool := strings.ToLower(DebugEnvVar.get()) == "true"
		if cfg.Debug != givenBool {
			cfg.Debug = givenBool
			configNeedsUpdate = true
		}
	}

	logrus.Infof("Checking for %s value. This is a boolean flag, expecting 'true' or 'false'", AgentStringTLSEnvVar.simpleName())
	if v := AgentStringTLSEnvVar.getSimple(); v != "" {
		logrus.Infof("Found value '%s' for %s", v, AgentStringTLSEnvVar.simpleName())
		tlsMode, err := strconv.ParseBool(strings.ToLower(v))
		if err != nil {
			logrus.Errorf("Error encountered while pasring %s, field will not be updated: %v", AgentStringTLSEnvVar.getSimple(), err)
		} else if cfg.AgentStrictTLSMode != tlsMode {
			cfg.AgentStrictTLSMode = tlsMode
			configNeedsUpdate = true
		}
	}

	// If we haven't made any changes there is no reason to update the config file
	if configNeedsUpdate {
		logrus.Infof("Detected a change in configuration, updating config file")
		err = saveConfig(cfg, path)
		if err != nil {
			return configNeedsUpdate, fmt.Errorf("failed to save config: %v", err)
		}
	} else {
		logrus.Infof("Did not detect a change in configuration")
	}

	return configNeedsUpdate, nil
}
