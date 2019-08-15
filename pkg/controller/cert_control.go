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
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	capi "k8s.io/api/certificates/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CertControlInterface manages certificates used by TiDB clusters
type CertControlInterface interface {
	Sign(suffix string) ([]byte, error)
	LoadFromSecret() error
	SaveToSecret() error
	//RevokeCert() error
	//RenewCert() error
}

type realCertControl struct {
	kubeCli kubernetes.Interface
	tc      *v1alpha1.TidbCluster
	rawCSR  []byte
}

// NewRealCertControl creates a new CertControlInterface
func NewRealCertControl(
	kubeCli kubernetes.Interface,
	tc *v1alpha1.TidbCluster,
	rawCSR []byte,
) CertControlInterface {
	return &realCertControl{
		kubeCli: kubeCli,
		tc:      tc,
		rawCSR:  rawCSR,
	}
}

func (rcc *realCertControl) Sign(suffix string) ([]byte, error) {
	//ns := rcc.tc.GetNamespace()
	tcName := rcc.tc.GetName()
	csrName := fmt.Sprintf("%s-%s", tcName, suffix)

	csr, err := rcc.sendCSR(suffix)
	if err != nil {
		return nil, err
	}
	err = rcc.approveCSR(csr)
	if err != nil {
		return nil, err
	}

	// wait at most 10min for the cert to be signed
	timeout := time.After(time.Minute * 10)
	tick := time.Tick(time.Millisecond * 500)
	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("fail to get signed certificate for %s after 10min", csrName)
		case <-tick:
			approveCond := csr.Status.Conditions[len(csr.Status.Conditions)-1].Type
			if approveCond == capi.CertificateApproved &&
				csr.Status.Certificate != nil {
				return csr.Status.Certificate, nil
			}
		}
	}
}

func (rcc *realCertControl) sendCSR(suffix string) (*capi.CertificateSigningRequest, error) {
	ns := rcc.tc.GetNamespace()
	tcName := rcc.tc.GetName()

	req := &capi.CertificateSigningRequest{
		TypeMeta: types.TypeMeta{Kind: "CertificateSigningRequest"},
		ObjectMeta: types.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", tcName, suffix),
		},
		Spec: capi.CertificateSigningRequestSpec{
			Request: rcc.rawCSR,
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
		return resp, fmt.Errorf("failed to create CSR for [%s/%s]: %s-%s, error: %v", ns, tcName, tcName, suffix, err)
	}

	glog.Infof("CSR created for [%s/%s]: %s-%s", ns, tcName, tcName, suffix)
	return resp, nil
}

func (rcc *realCertControl) approveCSR(csr *capi.CertificateSigningRequest) error {
	csr.Status.Conditions = append(csr.Status.Conditions, capi.CertificateSigningRequestCondition{
		Type:    capi.CertificateApproved,
		Reason:  "AutoApproved",
		Message: "Auto approved by TiDB Operator",
	})
	_, err := rcc.kubeCli.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(csr)
	if err != nil {
		return fmt.Errorf("error updateing approval for csr: %v", err)
	}
	return nil
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
