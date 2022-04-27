package tls

import (
	"crypto/x509"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

const (
	rancherCertPath      = "cacerts"
	localRancherCertDir  = "C:\\etc\\rancher\\agent\\"
	localRancherCertFile = localRancherCertDir + "ranchercert"
	localGenericCertDir  = "C:\\certs\\"
	localGenericCertFile = localGenericCertDir + "customcert"
)

func SetupRancherTLSConfig() (*x509.CertPool, error) {
	if os.Getenv("CATTLE_CA_CHECKSUM") == "" {
		logrus.Info("[SetupRancherTLSConfig] CATTLE_CA_CHECKSUM environment variable is not set, skipping TLS configuration")
		return nil, nil
	}

	// Get the SystemCertPool, continue with an empty pool on error
	systemCerts, err := x509.SystemCertPool()
	if err != nil || systemCerts == nil {
		logrus.Warnf("[SetupRancherTLSConfig] failed to return System Cert Pool, creating new one: %v", err)
		systemCerts = x509.NewCertPool()
	}

	// Get Certificate from Rancher
	// CATTLE_SERVER should always be set when provisioning through Rancher
	rancherURL, err := url.Parse(os.Getenv("CATTLE_SERVER") + rancherCertPath)
	if err != nil {
		return nil, err
	}

	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	resp, err := client.Get(rancherURL.Path)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	f, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("[SetupRancherTLSConfig] failed to read http client response: %v", err)
	}

	err = ioutil.WriteFile(localRancherCertFile, f, 0440)
	if err != nil {
		return nil, fmt.Errorf("[SetupRancherTLSConfig] failed to write file %q: %v", localRancherCertFile, err)
	}

	certs, err := ioutil.ReadFile(localRancherCertFile)
	if err != nil {
		return nil, fmt.Errorf("[SetupRancherTLSConfig] failed to read local cert file %q: %v", localRancherCertFile, err)
	}

	// Append the Rancher cert to the system pool
	if ok := systemCerts.AppendCertsFromPEM(certs); !ok {
		return nil, fmt.Errorf("[SetupRancherTLSConfig] unable to append cert %s to store, using system certs only", localRancherCertFile)
	}

	logrus.Infof("[SetupRancherTLSConfig] successfully loaded %s certificate into system cert store", localRancherCertFile)
	return systemCerts, nil
}

// SetupGenericTLSConfigFromURL SetupGenericTLSConfig returns a x509 system certpool containing the certificate specified in the URL
func SetupGenericTLSConfigFromURL(certURL string) (*x509.CertPool, error) {
	if certURL == "" {
		logrus.Info("[SetupGenericTLSConfigFromURL] specified certificate URL was empty, not modifying system certificate store")
		return nil, nil
	}

	// Get the System certificate store, continue with a new/empty pool on error
	systemCerts, err := x509.SystemCertPool()
	if err != nil || systemCerts == nil {
		logrus.Warnf("[SetupGenericTLSConfigFromURL] failed to return System Cert Pool, creating new one: %v", err)
		systemCerts = x509.NewCertPool()
	}

	// Get Certificate from URL
	externalCertURL, err := url.Parse(certURL)
	if err != nil {
		return nil, err
	}

	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	resp, err := client.Get(externalCertURL.Path)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	f, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("[SetupGenericTLSConfigFromURL] failed to read http client response: %v", err)
	}

	err = ioutil.WriteFile(localGenericCertFile, f, 0440)
	if err != nil {
		return nil, fmt.Errorf("[SetupGenericTLSConfigFromURL] failed to write file %q: %v", localGenericCertFile, err)
	}

	certs, err := ioutil.ReadFile(localGenericCertFile)
	if err != nil {
		return nil, fmt.Errorf("[SetupGenericTLSConfigFromURL] failed to read local cert file %q: %v", localGenericCertFile, err)
	}

	// Append the Rancher cert to the system pool
	if ok := systemCerts.AppendCertsFromPEM(certs); !ok {
		return nil, fmt.Errorf("[SetupGenericTLSConfigFromURL] unable to append cert %s to store, using system certs only", localGenericCertFile)
	}

	logrus.Infof("[SetupGenericTLSConfigFromURL] successfully loaded %s certificate into system cert store", localGenericCertFile)
	return systemCerts, nil
}

// SetupGenericTLSConfigFromFile returns a x509 system certpool containing the specified certificate file
func SetupGenericTLSConfigFromFile(certFile string) (*x509.CertPool, error) {
	if certFile == "" {
		logrus.Info("[SetupGenericTLSConfigFromFile] specified certificate file path is empty, not modifying system certificate store")
		return nil, nil
	}

	// Get the System certificate store, continue with a new/empty pool on error
	systemCerts, err := x509.SystemCertPool()
	if err != nil || systemCerts == nil {
		logrus.Warnf("[SetupGenericTLSConfigFromFile] failed to return System Cert Pool, creating new one: %v", err)
		systemCerts = x509.NewCertPool()
	}

	localCertPath, err := os.Stat(certFile)
	if err != nil {
		return nil, fmt.Errorf("[SetupGenericTLSConfigFromFile] unable to read certificate %s from %s: %v", localCertPath.Name(), certFile, err)
	}

	certs, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("[SetupGenericTLSConfigFromFile] failed to read local cert file %q: %v", certFile, err)
	}

	// Append the specified cert to the system pool
	if ok := systemCerts.AppendCertsFromPEM(certs); !ok {
		return nil, fmt.Errorf("[SetupGenericTLSConfigFromFile] unable to append cert %s to store, using system certs only", certFile)
	}

	logrus.Infof("[SetupGenericTLSConfigFromFile] successfully loaded %s certificate into system cert store", localGenericCertFile)
	return systemCerts, nil
}
