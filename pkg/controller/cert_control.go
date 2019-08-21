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
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/label"
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
	Create(ns string, instance string, commonName string, hostList []string, IPList []string, suffix string) error
	LoadFromSecret(ns string, secretName string) ([]byte, []byte, error)
	SaveToSecret(ns string, instance string, component string, cert []byte, key []byte) error
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

func (rcc *realCertControl) Create(ns string, instance string, commonName string,
	hostList []string, IPList []string, suffix string) error {
	csrName := fmt.Sprintf("%s-%s", instance, suffix)

	// generate certificate if not exist
	_, _, err := rcc.LoadFromSecret(ns, csrName)
	if !apierrors.IsNotFound(err) {
		return err
	}
	if err == nil {
		// TODO: validate the cert and key
		glog.Infof("Secret %s already exist, reusing the key pair. TidbCluster: %s/%s", csrName, ns, csrName)
		return nil
	}

	rawCSR, key, err := certutil.NewCSR(commonName, hostList, IPList)
	if err != nil {
		return fmt.Errorf("fail to generate new key and certificate for %s/%s, %v", ns, csrName, err)
	}

	// sign certificate
	csr, err := rcc.sendCSR(ns, instance, rawCSR, suffix)
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
		glog.Errorf("error watch CSR for [%s/%s]: %s", ns, instance, csrName, suffix)
		return err
	}

	watchCh := csrCh.ResultChan()
	for {
		select {
		case <-tick:
			glog.Infof("CSR still not approved for [%s/%s]: %s, retry later", ns, instance, csrName, suffix)
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
				glog.Infof("signed certificate for [%s/%s]: %s", ns, instance, csrName, suffix)

				// save signed certificate and key to secret
				err = rcc.SaveToSecret(ns, instance, suffix, updatedCSR.Status.Certificate, key)
				if err == nil {
					// cleanup the approved csr
					delOpts := &types.DeleteOptions{TypeMeta: types.TypeMeta{Kind: "CertificateSigningRequest"}}
					return rcc.kubeCli.Certificates().CertificateSigningRequests().Delete(csrName, delOpts)
				}
				return err
			}
			continue
		}
	}
}

func (rcc *realCertControl) getCSR(ns string, instance string, csrName string) (*capi.CertificateSigningRequest, error) {
	getOpts := types.GetOptions{TypeMeta: types.TypeMeta{Kind: "CertificateSigningRequest"}}
	csr, err := rcc.kubeCli.CertificatesV1beta1().CertificateSigningRequests().Get(csrName, getOpts)
	if err != nil && apierrors.IsNotFound(err) {
		// it's supposed to be not found
		return nil, nil
	}
	if err != nil {
		// something else went wrong
		return nil, err
	}

	// check for exist CSR, overwirte if it was created by operator, otherwise block the process
	labelTemp := label.New()
	if csr.Labels[label.NamespaceLabelKey] == ns &&
		csr.Labels[label.ManagedByLabelKey] == labelTemp[label.ManagedByLabelKey] &&
		csr.Labels[label.InstanceLabelKey] == instance {
		glog.Infof("found exist CSR %s/%s created by tidb-operator, overwriting", ns, csrName)
		return csr, nil
	}
	return nil, fmt.Errorf("CSR %s/%s already exist, but not created by tidb-operator, skip it", ns, csrName)
}

func (rcc *realCertControl) sendCSR(ns string, instance string, rawCSR []byte, suffix string) (*capi.CertificateSigningRequest, error) {
	csrName := fmt.Sprintf("%s-%s", instance, suffix)

	var csr *capi.CertificateSigningRequest

	csr, err := rcc.getCSR(ns, instance, csrName)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSR for [%s/%s]: %s, error: %v", ns, instance, csrName, err)
	}

	if csr != nil {
		delOpts := &types.DeleteOptions{TypeMeta: types.TypeMeta{Kind: "CertificateSigningRequest"}}
		err := rcc.kubeCli.Certificates().CertificateSigningRequests().Delete(csrName, delOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to delete exist old CSR for [%s/%s]: %s, error: %v", ns, instance, csrName, err)
		}
		glog.Infof("exist old CSR deleted for [%s/%s]: %s", ns, instance, csrName)
		return rcc.sendCSR(ns, instance, rawCSR, suffix)
	}

	csr = &capi.CertificateSigningRequest{
		TypeMeta: types.TypeMeta{Kind: "CertificateSigningRequest"},
		ObjectMeta: types.ObjectMeta{
			Name:   csrName,
			Labels: make(map[string]string),
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

	labelTemp := label.New()
	csr.Labels[label.NamespaceLabelKey] = ns
	csr.Labels[label.ManagedByLabelKey] = labelTemp[label.ManagedByLabelKey]
	csr.Labels[label.InstanceLabelKey] = instance

	resp, err := rcc.kubeCli.CertificatesV1beta1().CertificateSigningRequests().Create(csr)
	if err != nil {
		return resp, fmt.Errorf("failed to create CSR for [%s/%s]: %s, error: %v", ns, instance, csrName, err)
	}
	glog.Infof("CSR created for [%s/%s]: %s", ns, instance, csrName)
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

// LoadFromSecret loads cert and key from Secret matching the name
func (rcc *realCertControl) LoadFromSecret(ns string, secretName string) ([]byte, []byte, error) {
	secret, err := rcc.kubeCli.CoreV1().Secrets(ns).Get(secretName, types.GetOptions{})

	return secret.Data["cert"], secret.Data["key"], err
}

func (rcc *realCertControl) SaveToSecret(ns string, instance string, component string, cert []byte, key []byte) error {
	secretName := fmt.Sprintf("%s-%s", instance, component)
	// the client cert is part of tidb component
	componentName := strings.TrimSuffix(component, "-client")

	secret := &corev1.Secret{
		ObjectMeta: types.ObjectMeta{
			Name:   secretName,
			Labels: make(map[string]string),
		},
		Data: map[string][]byte{
			"cert": cert,
			"key":  key,
		},
	}

	labelTemp := label.New()
	secret.Labels[label.NamespaceLabelKey] = ns
	secret.Labels[label.ManagedByLabelKey] = labelTemp[label.ManagedByLabelKey]
	secret.Labels[label.InstanceLabelKey] = instance
	secret.Labels[label.ComponentLabelKey] = componentName

	_, err := rcc.kubeCli.CoreV1().Secrets(ns).Create(secret)
	glog.Infof("save cert to secret %s/%s, error: %v", ns, secretName, err)
	return err
}

// CheckSecret returns true if the secret already exist
func (rcc *realCertControl) CheckSecret(ns string, secretName string) bool {
	_, _, err := rcc.LoadFromSecret(ns, secretName)
	if err == nil {
		// TODO: validate the cert and key
		return true
	}
	return false
}

var _ CertControlInterface = &realCertControl{}
