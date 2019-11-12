// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"time"
)

// BasicConstraints CSR information RFC 5280, 4.2.1.9
type basicConstraints struct {
	IsCA       bool `asn1:"optional"`
	MaxPathLen int  `asn1:"optional,default:-1"`
}

func Checksum(src []byte) string {
	h := sha256.New()
	h.Write(src)
	out := h.Sum(nil)
	return hex.EncodeToString(out)
}

func DecodeCertPem(certByte []byte) (*x509.Certificate, error) {

	block, _ := pem.Decode(certByte)
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

// check if cert would be expired after refreshIntervalHour hours
func IsCertificateNeedRefresh(cert *x509.Certificate, refreshIntervalHour int) bool {
	now := time.Now()
	expireDate := cert.NotAfter
	internal := expireDate.Sub(now)
	return internal.Hours() <= float64(refreshIntervalHour)
}

func GenerateRSAKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

func GenerateKeyPEM(key *rsa.PrivateKey) []byte {
	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.EncodeToMemory(pemBlock)
}

func GenerateCSRPem(key *rsa.PrivateKey, commonName string, dnsNames []string) ([]byte, error) {
	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: commonName,
		},
		DNSNames: dnsNames,
	}
	caExt, err := createCAExtension()
	if err != nil {
		return nil, fmt.Errorf("failed to create CA Extension")
	}

	template.Extensions = []pkix.Extension{caExt}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &template, key)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSR: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr}), nil
}

func createCAExtension() (pkix.Extension, error) {
	val, err := asn1.Marshal(basicConstraints{false, 0})
	if err != nil {
		return pkix.Extension{}, fmt.Errorf("failed to marshal basic constraints: %v", err)
	}

	return pkix.Extension{
		Id:       asn1.ObjectIdentifier{2, 5, 29, 19},
		Value:    val,
		Critical: true,
	}, nil
}

//func GenerateRSAKeyPair(namespace, serviceName string, privateKey *rsa.PrivateKey) ([]byte, []byte, error) {
//	ca := &x509.Certificate{
//		SerialNumber: big.NewInt(1653),
//		Subject: pkix.Name{
//			Organization: []string{"Acme Co"},
//		},
//		NotBefore:             time.Now(),
//		NotAfter:              time.Now().AddDate(1, 0, 0),
//		SubjectKeyId:          []byte{1, 2, 3, 4, 5},
//		BasicConstraintsValid: true,
//		IsCA:                  true,
//		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
//		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
//	}
//	server := &x509.Certificate{
//		SerialNumber: big.NewInt(1658),
//		Subject: pkix.Name{
//			Organization: []string{"SERVER"},
//		},
//		NotBefore:    time.Now(),
//		NotAfter:     time.Now().AddDate(10, 0, 0),
//		SubjectKeyId: []byte{1, 2, 3, 4, 6},
//		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
//		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
//	}
//	hosts := []string{serviceName + "." + namespace + ".svc"}
//	for _, h := range hosts {
//		server.DNSNames = append(server.DNSNames, h)
//	}
//	privSer, _ := rsa.GenerateKey(rand.Reader, 2048)
//	return createCertificateFile(server, privSer, ca, privateKey)
//}
//
//func createCertificateFile(cert *x509.Certificate, key *rsa.PrivateKey, caCert *x509.Certificate, caKey *rsa.PrivateKey) ([]byte, []byte, error) {
//	priv := key
//	pub := &priv.PublicKey
//	privPm := priv
//	if caKey != nil {
//		privPm = caKey
//	}
//	ca_b, err := x509.CreateCertificate(rand.Reader, cert, caCert, pub, privPm)
//	if err != nil {
//		return nil, nil, err
//	}
//	var certificate = &pem.Block{Type: "CERTIFICATE",
//		Headers: map[string]string{},
//		Bytes:   ca_b}
//	ca_b64 := pem.EncodeToMemory(certificate)
//	priv_b := x509.MarshalPKCS1PrivateKey(priv)
//	var privateKey = &pem.Block{Type: "PRIVATE KEY",
//		Headers: map[string]string{},
//		Bytes:   priv_b}
//	priv_b64 := pem.EncodeToMemory(privateKey)
//	return ca_b64, priv_b64, nil
//}
