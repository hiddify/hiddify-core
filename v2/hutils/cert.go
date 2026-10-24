package hutils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

// CertificatePair holds the certificate and private key
type CertificatePair struct {
	Certificate []byte
	PrivateKey  []byte
}

// GenerateCertificatePair generates a self-signed certificate and private key
// trusted for local gRPC (127.0.0.1 / localhost).
func GenerateCertificatePair() (*CertificatePair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial: %v", err)
	}

	certTemplate := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"Hiddify, Inc."},
			CommonName:   "127.0.0.1",
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(10 * 365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		DNSNames:              []string{"localhost"},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	return &CertificatePair{
		Certificate: certPEM,
		PrivateKey:  keyPEM,
	}, nil
}

// CertificateHasLoopbackSAN reports whether the PEM certificate can be used for 127.0.0.1 / localhost.
func CertificateHasLoopbackSAN(certPEM []byte) bool {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}
	for _, ip := range cert.IPAddresses {
		if ip != nil && ip.IsLoopback() {
			return true
		}
	}
	for _, name := range cert.DNSNames {
		if name == "localhost" {
			return true
		}
	}
	return cert.Subject.CommonName == "127.0.0.1" || cert.Subject.CommonName == "localhost"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func GenerateCertificateFile(certPath, keyPath string, isServer bool, skipIfExist bool) error {
	if skipIfExist && fileExists(certPath) && fileExists(keyPath) {
		return nil
	}
	err := os.MkdirAll("data/cert", 0o744)
	if err != nil {
		return err
	}
	cers, err := GenerateCertificatePair()
	if err != nil {
		return err
	}
	err = os.WriteFile(certPath, cers.Certificate, 0o644)
	if err != nil {
		return err
	}
	err = os.WriteFile(keyPath, cers.PrivateKey, 0o600)
	if err != nil {
		return err
	}
	return os.Chmod(keyPath, 0o600)
}
