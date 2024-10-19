package hutils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

// CertificatePair holds the certificate and private key
type CertificatePair struct {
	Certificate []byte
	PrivateKey  []byte
}

// GenerateCertificatePair generates a self-signed certificate and private key
func GenerateCertificatePair() (*CertificatePair, error) {
	// Generate a new RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	// Create a template for the certificate
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1), // A unique serial number for the certificate
		Subject: pkix.Name{
			Organization: []string{"Hiddify, Inc."},
			CommonName:   "Hiddify",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year

		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},

		BasicConstraintsValid: true,
		IsCA:                  true, // This is a CA certificate (for testing purposes)
	}

	// Self-sign the certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	// Encode the certificate to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})

	// Encode the private key to PEM format
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	return &CertificatePair{
		Certificate: certPEM,
		PrivateKey:  keyPEM,
	}, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err) // returns true if the file exists
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
	err = os.Chmod(keyPath, 0o600)
	if err != nil {
		return err
	}
	return nil
}
