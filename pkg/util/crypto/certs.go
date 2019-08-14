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
// limitations under the License.package spec

package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/golang/glog"
)

const (
	rsaKeySize = 2048
)

// generate a new private key
func newPrivateKey(size int) (*rsa.PrivateKey, error) {
	// TODO: support more key types
	privateKey, err := rsa.GenerateKey(rand.Reader, *keySize)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

// convert private key to PEM format
func convertToPEM(blockType string, dataBytes *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory{
		&pem.Block{
			Type:    blockType,
			Headers: nil,
			Bytes:   x509.MarshalPKCS1PrivateKey(dataBytes),
		},
	}
}

func NewCSR(commonName string, hostList []string, IPList []string) ([]byte, error) {
	// TODO: option to use an exist private key
	privKey, err := newPrivateKey(rsaKeySize)
	if err != nil {
		return nil, err
	}
	pemKey := convertToPEM("RSA PRIVATE KEY", privKey)

	// set CSR attributes
	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{
			Organization:       []string{"PingCAP"},
			OrganizationalUnit: []string{"TiDB Operator"},
			CommonName:         commonName,
		},
		DNSNames:    hostList,
		IPAddresses: IPList,
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, pemKey)
	if err != nil {
		return nil, err
	}
	return csr, nil
}
