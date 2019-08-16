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
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	certutil "github.com/pingcap/tidb-operator/pkg/util/crypto"
	capi "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

// CertControlInterface manages certificates used by TiDB clusters
type CertControlInterface interface {
	Create(tc *v1alpha1.TidbCluster, commonName string, hostList []string, IPList []string, suffix string) error
	LoadFromSecret(ns string, secretName string) ([]byte, []byte, error)
	SaveToSecret(ns string, secretName string, cert []byte, key []byte) error
	CheckSecret(ns string, secretName string) bool
	//RevokeCert() error
	//RenewCert() error
}

type realCertControl struct {
	kubeCli kubernetes.Interface
}

// NewRealCertControl creates a new CertControlInterface
func NewRealCertControl(
	kubeCli kubernetes.Interface,
) CertControlInterface {
	return &realCertControl{
		kubeCli: kubeCli,
	}
}

func (rcc *realCertControl) Create(tc *v1alpha1.TidbCluster, commonName string,
	hostList []string, IPList []string, suffix string) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	csrName := fmt.Sprintf("%s-%s", tcName, suffix)

	// generate certificate if not exist
	_, key, err := rcc.LoadFromSecret(ns, csrName)
	if !apierrors.IsNotFound(err) {
		return err
	}
	if err == nil {
		glog.Infof("Secret %s/%s already exist, reusing the key", ns, csrName)
		// TODO: validate the cert
		return nil
	}

	rawCSR, key, err := certutil.NewCSR(commonName, hostList, IPList)
	if err != nil {
		return fmt.Errorf("fail to generate new key and certificate for %s/%s, %v", ns, csrName, err)
	}

	// sign certificate
	csr, err := rcc.sendCSR(tc, rawCSR, suffix)
	if err != nil {
		return err
	}
	err = rcc.approveCSR(csr)
	if err != nil {
		return err
	}

	// wait at most 5min for the cert to be signed
	timeout := int64(time.Minute.Seconds() * 5)
	tick := time.After(time.Second * 10)
	watchReq := types.ListOptions{
		Watch:          true,
		TimeoutSeconds: &timeout,
		FieldSelector:  fields.OneTermEqualSelector("metadata.name", csrName).String(),
	}

	csrCh, err := rcc.kubeCli.Certificates().CertificateSigningRequests().Watch(watchReq)
	if err != nil {
		glog.Errorf("error watch CSR for [%s/%s]: %s-%s", ns, tcName, tcName, suffix)
		return err
	}

	watchCh := csrCh.ResultChan()
	for {
		select {
		case <-tick:
			glog.Infof("CSR still not approved for [%s/%s]: %s-%s, retry later", ns, tcName, tcName, suffix)
			continue
		case event, ok := <-watchCh:
			if !ok {
				return fmt.Errorf("fail to get signed certificate for %s", csrName)
			}

			if len(event.Object.(*capi.CertificateSigningRequest).Status.Conditions) == 0 {
				continue
			}

			updatedCSR := event.Object.(*capi.CertificateSigningRequest)
			approveCond := updatedCSR.Status.Conditions[len(csr.Status.Conditions)-1].Type

			if updatedCSR.UID == csr.UID &&
				approveCond == capi.CertificateApproved &&
				updatedCSR.Status.Certificate != nil {
				glog.Infof("signed certificate for [%s/%s]: %s-%s", ns, tcName, tcName, suffix)
				return rcc.SaveToSecret(ns, csrName, updatedCSR.Status.Certificate, key)
			}
			continue
		}
	}

	// TODO: cleanup csr object
}

func (rcc *realCertControl) sendCSR(tc *v1alpha1.TidbCluster, rawCSR []byte, suffix string) (*capi.CertificateSigningRequest, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	req := &capi.CertificateSigningRequest{
		TypeMeta: types.TypeMeta{Kind: "CertificateSigningRequest"},
		ObjectMeta: types.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", tcName, suffix),
		},
		Spec: capi.CertificateSigningRequestSpec{
			Request: pem.EncodeToMemory(&pem.Block{
				Type:    "CERTIFICATE REQUEST",
				Headers: nil,
				Bytes:   rawCSR,
			}),
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
		return fmt.Errorf("error updating approval for csr: %v", err)
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
func (rcc *realCertControl) LoadFromSecret(ns string, secretName string) ([]byte, []byte, error) {
	secret, err := rcc.kubeCli.CoreV1().Secrets(ns).Get(secretName, types.GetOptions{})

	return secret.Data["cert"], secret.Data["key"], err
}

func (rcc *realCertControl) SaveToSecret(ns string, secretName string, cert []byte, key []byte) error {
	secret := &corev1.Secret{
		ObjectMeta: types.ObjectMeta{
			Name: secretName,
		},
		Data: map[string][]byte{
			"cert": cert,
			"key":  key,
		},
	}

	_, err := rcc.kubeCli.CoreV1().Secrets(ns).Create(secret)
	glog.Infof("save cert to secret %s/%s, error: %v", ns, secretName, err)
	return err
}

// CheckSecret returns true if the secret already exist
func (rcc *realCertControl) CheckSecret(ns string, secretName string) bool {
	_, _, err := rcc.LoadFromSecret(ns, secretName)
	if err == nil {
		// TODO: validate the cert
		return true
	}
	return false
}

var _ CertControlInterface = &realCertControl{}
