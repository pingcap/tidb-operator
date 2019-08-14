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

package controller

import (
	//"crypto/x509"
	"fmt"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	capi "k8s.io/api/certificates/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CertControlInterface manages certificates used by TiDB clusters
type CertControlInterface interface {
	Sign() error
	LoadFromSecret() error
	SaveToSecret() error
	//RevokeCert() error
	//RenewCert() error
}

type realCertControl struct {
	kubeCli kubernetes.Interface
	csr     []byte
}

// NewRealCertControl creates a new CertControlInterface
func NewRealCertControl(
	kubeCli kubernetes.Interface,
	csr []byte,
) CertControlInterface {
	return &realCertControl{
		kubeCli: kubeCli,
		csr:     csr,
	}
}

func (rcc *realCertControl) Sign() error {
	// TODO: Implement this method
	return nil
}

func (rcc *realCertControl) sendCSR(tc *v1alpha1.TidbCluster, suffix string) (*capi.CertificateSigningRequest, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	req := &capi.CertificateSigningRequest{
		TypeMeta: types.TypeMeta{Kind: "CertificateSigningRequest"},
		ObjectMeta: types.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", tcName, suffix),
		},
		Spec: capi.CertificateSigningRequestSpec{
			Request: rcc.csr,
			Usages: []capi.KeyUsage{
				capi.UsageClientAuth,
				capi.UsageServerAuth,
			},
		},
	}

	resp, err := rcc.kubeCli.CertificatesV1beta1().CertificateSigningRequests().Create(req)
	if err != nil && apierrors.IsAlreadyExists(err) {
		glog.Infof("CSR already exist for [%s/%s]: %s-%s, reusing it", ns, tcName, tcName, suffix)
		getOpts := types.GetOptions{TypeMeta: types.TypeMeta{Kind: "CertificateSigningRequest"}}
		resp, err = rcc.kubeCli.CertificatesV1beta1().CertificateSigningRequests().Get(req.Name, getOpts)
	}
	if err != nil {
		glog.Errorf("failed to create CSR for [%s/%s]: %s-%s", ns, tcName, tcName, suffix)
		return resp, err
	}

	glog.Infof("CSR created for [%s/%s]: %s-%s", ns, tcName, tcName, suffix)
	return resp, nil
}

func (rcc *realCertControl) approveCSR() {
	return
}

/*
func (rcc *realCertControl) RevokeCert() error {
	return nil
}
*/
/*
func (rcc *realCertControl) RenewCert() error {
	return nil
}
*/
func (rcc *realCertControl) LoadFromSecret() error {
	return nil
}

func (rcc *realCertControl) SaveToSecret() error {
	return nil
}

var _ CertControlInterface = &realCertControl{}
