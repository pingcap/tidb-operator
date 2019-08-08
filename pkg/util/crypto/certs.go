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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	capi "k8s.io/api/certificates/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

func sendCSR(kubeCli kubernetes.Interface, tc *v1alpha1.TidbCluster, csr []byte, suffix string) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	req := &capi.CertificateSigningRequest{
		TypeMeta: types.TypeMeta{Kind: "CertificateSigningRequest"},
		ObjectMeta: types.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", tcName, suffix),
		},
		Spec: capi.CertificateSigningRequestSpec{
			Request: csr,
			Usages: []capi.KeyUsage{
				capi.UsageClientAuth,
				capi.UsageServerAuth,
			},
		},
	}

	resp, err := kubeCli.CertificatesV1beta1().CertificateSigningRequests().Create(req)
	if err != nil && apierrors.IsAlreadyExists(err) {
		glog.Infof("CSR already exist for [%s/%s]: %s-%s, reusing it", ns, tcName, tcName, suffix)
		getOpts := types.GetOptions{TypeMeta: types.TypeMeta{Kind: "CertificateSigningRequest"}}
		resp, err = kubeCli.CertificatesV1beta1().CertificateSigningRequests().Get(req.Name, getOpts)
	}
	if err != nil {
		glog.Errorf("failed to create CSR for [%s/%s]: %s-%s", ns, tcName, tcName, suffix)
		return
	}

	glog.Infof("CSR created for [%s/%s]: %s-%s", ns, tcName, tcName, suffix)
}

func approveCSR() {
	return
}

func GetSignedCert(kubeCli kubernetes.Interface, tc *v1alpha1.TidbCluster, csr []byte, suffix string) {
	sendCSR(kubeCli, tc, csr, suffix)

	return
}

//func RevokeCert() {}
//func RenewCert() {}

func LoadKeyPairFromSecret() {
	return
}

func SaveKeyPairToSecret() {
	return
}
