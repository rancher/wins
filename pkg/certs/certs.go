package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"math"
	"math/big"
	"time"

	"github.com/pkg/errors"
)

func GeneratePrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

func GenerateSelfSignedCACert(commonName string, key *rsa.PrivateKey) (*x509.Certificate, error) {
	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(time.Hour * 24 * 365 * 10).UTC(), // 10 years
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDERBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

func GenerateSignedCert(commonName string, extKeyUsages []x509.ExtKeyUsage, key *rsa.PrivateKey, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(commonName) == 0 {
		return nil, errors.New("must specify CommonName")
	}
	if len(extKeyUsages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName: commonName,
		},
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(time.Hour * 24 * 365 * 10).UTC(), // 10 years
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  extKeyUsages,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

func GenerateSelfSignedCACertAndKey(commonName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := GeneratePrivateKey()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate private key")
	}

	cert, err := GenerateSelfSignedCACert(commonName, key)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate cert")
	}

	return cert, key, nil
}

func GenerateCertAndKey(commonName string, extKeyUsages []x509.ExtKeyUsage, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := GeneratePrivateKey()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate private key")
	}

	cert, err := GenerateSignedCert(commonName, extKeyUsages, key, caCert, caKey)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate cert")
	}

	return cert, key, nil
}

func EncodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.EncodeToMemory(&block)
}

func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

func WritePrivateKeyPEM(path string, key *rsa.PrivateKey) error {
	return ioutil.WriteFile(path, EncodePrivateKeyPEM(key), 0600)
}

func WriteCertPEM(path string, cert *x509.Certificate) error {
	return ioutil.WriteFile(path, EncodeCertPEM(cert), 0600)
}
