package tls

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
)

type TLSConfig struct {
	Insecure     *bool  `yaml:"insecure" json:"insecure"`
	CertFilePath string `yaml:"CertFilePath" json:"CertFilePath"`
}

// SetupGenericTLSConfigFromFile returns a x509 system certificate pool containing the specified certificate file
func SetupGenericTLSConfigFromFile(config TLSConfig) (*x509.CertPool, error) {
	if config.CertFilePath == "" {
		logrus.Info("[SetupGenericTLSConfigFromFile] specified certificate file path is empty, not modifying system certificate store")
		return nil, nil
	}

	// Get the System certificate store, continue with a new/empty pool on error
	systemCerts, err := x509.SystemCertPool()
	if err != nil || systemCerts == nil {
		logrus.Warnf("[SetupGenericTLSConfigFromFile] failed to return System Cert Pool, creating new one: %v", err)
		systemCerts = x509.NewCertPool()
	}

	localCertPath, err := os.Stat(config.CertFilePath)
	if err != nil {
		return nil, fmt.Errorf("[SetupGenericTLSConfigFromFile] unable to read certificate %s from %s: %v", localCertPath.Name(), config.CertFilePath, err)
	}

	certs, err := ioutil.ReadFile(config.CertFilePath)
	if err != nil {
		return nil, fmt.Errorf("[SetupGenericTLSConfigFromFile] failed to read local cert file %q: %v", config.CertFilePath, err)
	}

	// Append the specified cert to the system pool
	if ok := systemCerts.AppendCertsFromPEM(certs); !ok {
		return nil, fmt.Errorf("[SetupGenericTLSConfigFromFile] unable to append cert %s to store, using system certs only", config.CertFilePath)
	}

	logrus.Infof("[SetupGenericTLSConfigFromFile] successfully loaded %s certificate into system cert store", config.CertFilePath)
	return systemCerts, nil
}
