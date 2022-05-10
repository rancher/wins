package tls

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
)

type Config struct {
	Insecure     *bool  `yaml:"insecure" json:"insecure"`
	CertFilePath string `yaml:"certFilePath" json:"certFilePath"`
}

// SetupGenericTLSConfigFromFile returns a x509 system certificate pool containing the specified certificate file
func (c *Config) SetupGenericTLSConfigFromFile() (*x509.CertPool, error) {
	logrus.Infof("Configuring TLS as insecure: %v and using the following cert: %s", c.Insecure, c.CertFilePath)
	if c.CertFilePath == "" {
		logrus.Info("[SetupGenericTLSConfigFromFile] specified certificate file path is empty, not modifying system certificate store")
		return nil, nil
	}

	// Get the System certificate store, continue with a new/empty pool on error
	systemCerts, err := x509.SystemCertPool()
	if err != nil || systemCerts == nil {
		logrus.Warnf("[SetupGenericTLSConfigFromFile] failed to return System Cert Pool, creating new one: %v", err)
		systemCerts = x509.NewCertPool()
	}

	localCertPath, err := os.Stat(c.CertFilePath)
	if err != nil {
		return nil, fmt.Errorf("[SetupGenericTLSConfigFromFile] unable to read certificate %s from %s: %v", localCertPath.Name(), c.CertFilePath, err)
	}

	certs, err := ioutil.ReadFile(c.CertFilePath)
	if err != nil {
		return nil, fmt.Errorf("[SetupGenericTLSConfigFromFile] failed to read local cert file %q: %v", c.CertFilePath, err)
	}

	// Append the specified cert to the system pool
	if ok := systemCerts.AppendCertsFromPEM(certs); !ok {
		return nil, fmt.Errorf("[SetupGenericTLSConfigFromFile] unable to append cert %s to store, using system certs only", c.CertFilePath)
	}

	logrus.Infof("[SetupGenericTLSConfigFromFile] successfully loaded %s certificate into system cert store", c.CertFilePath)
	return systemCerts, nil
}
