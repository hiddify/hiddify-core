//go:build !windows

package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"math/big"
	"os"
	"time"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err) // returns true if the file exists
}

func GenerateCertificate(certPath, keyPath string, isServer bool, skipIfExist bool) {
	if skipIfExist && fileExists(certPath) && fileExists(keyPath) {
		return
	}
	err := os.MkdirAll("cert", 0o744)
	if err != nil {
		panic(err)
	}
	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		panic(err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	var keyUsage x509.KeyUsage
	var extKeyUsage []x509.ExtKeyUsage

	if isServer {
		keyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	} else {
		keyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Secure data transfer"}},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           extKeyUsage,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		panic(err)
	}

	certFile, err := os.Create(certPath)
	if err != nil {
		panic(err)
	}
	defer certFile.Close()
	certFile.Chmod(0o644)
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyFile, err := os.Create(keyPath)
	if err != nil {
		panic(err)
	}
	defer keyFile.Close()
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		panic(err)
	}
	keyFile.Chmod(0o644)
	pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})
}

func LoadCertificate(certPath, keyPath string) (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certPath, keyPath)
}

func LoadClientCA(certPath string) *x509.CertPool {
	pool := x509.NewCertPool()
	certPEM, err := ioutil.ReadFile(certPath)
	if err != nil {
		panic(err)
	}
	pool.AppendCertsFromPEM(certPEM)
	return pool
}
