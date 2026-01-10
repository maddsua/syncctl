package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

func setupSelfSignedTlsOrDie() *tls.Config {

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(fmt.Errorf("generate tls key: %v", err))
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "s4-server"
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(67),
		Subject: pkix.Name{
			Organization: []string{"localhost"},
			CommonName:   hostname,
		},
		NotBefore: time.Now(),
		//	expire in 5 years
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 5),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, privateKey.Public(), privateKey)
	if err != nil {
		panic(fmt.Errorf("generate tls cert: %v", err))
	}

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		panic(fmt.Errorf("marshal tls cert: %v", err))
	}

	tlsCert, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}))

	if err != nil {
		panic(fmt.Errorf("generate tls cert pair: %v", err))
	}

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}
}
