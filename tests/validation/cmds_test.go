package validation_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rancher/wins/cmd/server/config"
	"github.com/rancher/wins/pkg/converters"
	"github.com/rancher/wins/pkg/defaults"
)

func writeJsonConfig(cfg *config.Config) string {
	cfgJson, err := converters.ToJson(cfg)
	if err != nil {
		consoleFatal(fmt.Sprintf("Failed to convert config to json: %v", err))
	}

	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		consoleFatal(fmt.Sprintf("Failed to create temp file: %v", err))
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, strings.NewReader(cfgJson))
	if err != nil {
		consoleFatal(fmt.Sprintf("Failed to write json config: %v", err))
	}

	return tmpFile.Name()
}

func writeYamlConfig(cfg *config.Config) string {
	cfgYaml, err := converters.ToYaml(cfg)
	if err != nil {
		consoleFatal(fmt.Sprintf("Failed to convert config to yaml: %v", err))
	}

	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		consoleFatal(fmt.Sprintf("Failed to create temp file: %v", err))
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, strings.NewReader(cfgYaml))
	if err != nil {
		consoleFatal(fmt.Sprintf("Failed to write json config: %v", err))
	}

	return tmpFile.Name()
}

var _ = Describe("cmds", func() {
	Context("server", func() {
		It("default config", func() {
			defCfg := config.DefaultConfig()
			err := config.LoadConfig(defaults.ConfigPath, defCfg)

			Expect(err).NotTo(HaveOccurred())
		})

		It("load json config", func() {
			tmpCfg := config.DefaultConfig()
			tmpCfg.Listen = "rancher_wins2"
			tmpCfgPath := writeJsonConfig(tmpCfg)

			defCfg := config.DefaultConfig()
			err := config.LoadConfig(tmpCfgPath, defCfg)

			Expect(err).NotTo(HaveOccurred())
			Expect(defCfg.Listen).To(Equal(tmpCfg.Listen))
		})

		It("load yaml config", func() {
			tmpCfg := config.DefaultConfig()
			tmpCfg.Listen = "rancher_wins2"
			tmpCfgPath := writeYamlConfig(tmpCfg)

			defCfg := config.DefaultConfig()
			err := config.LoadConfig(tmpCfgPath, defCfg)

			Expect(err).NotTo(HaveOccurred())
			Expect(defCfg.Listen).To(Equal(tmpCfg.Listen))
		})

		It("disable upgrade", func() {
			tmpCfg := config.DefaultConfig()
			tmpCfg.Upgrade.Mode = "none"
			tmpCfgPath := writeYamlConfig(tmpCfg)

			defCfg := config.DefaultConfig()
			err := config.LoadConfig(tmpCfgPath, defCfg)

			Expect(err).NotTo(HaveOccurred())
			Expect(defCfg.Upgrade.Mode).To(Equal("none"))
		})

	})
})
