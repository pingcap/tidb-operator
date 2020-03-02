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

	"github.com/pingcap/tidb-operator/pkg/label"
	certutil "github.com/pingcap/tidb-operator/pkg/util/crypto"
	capi "k8s.io/api/certificates/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	certlisters "k8s.io/client-go/listers/certificates/v1beta1"
	"k8s.io/klog"
)

// TiDBClusterCertOptions contains information needed to create new certificates
type TiDBClusterCertOptions struct {
	Namespace  string
	Instance   string
	CommonName string
	HostList   []string
	IPList     []string
	Suffix     string
	Component  string
}

// CertControlInterface manages certificates used by TiDB clusters
type CertControlInterface interface {
	Create(or metav1.OwnerReference, certOpts *TiDBClusterCertOptions) error
	CheckSecret(ns string, secretName string) bool
	//RevokeCert() error
	//RenewCert() error
}

type realCertControl struct {
	kubeCli    kubernetes.Interface
	csrLister  certlisters.CertificateSigningRequestLister
	secControl SecretControlInterface
}

// NewRealCertControl creates a new CertControlInterface
func NewRealCertControl(
	kubeCli kubernetes.Interface,
	csrLister certlisters.CertificateSigningRequestLister,
	secControl SecretControlInterface,
) CertControlInterface {
	return &realCertControl{
		kubeCli:    kubeCli,
		csrLister:  csrLister,
		secControl: secControl,
	}
}

func (rcc *realCertControl) Create(or metav1.OwnerReference, certOpts *TiDBClusterCertOptions) error {
	var csrName string
	if certOpts.Suffix == "" {
		csrName = certOpts.Instance
	} else {
		csrName = fmt.Sprintf("%s-%s", certOpts.Instance, certOpts.Suffix)
	}

	// generate certificate if not exist
	if rcc.secControl.Check(certOpts.Namespace, csrName) {
		klog.Infof("Secret %s already exist, reusing the key pair. TidbCluster: %s/%s", csrName, certOpts.Namespace, csrName)
		return nil
	}

	rawCSR, key, err := certutil.NewCSR(certOpts.CommonName, certOpts.HostList, certOpts.IPList)
	if err != nil {
		return fmt.Errorf("fail to generate new key and certificate for %s/%s, %v", certOpts.Namespace, csrName, err)
	}

	// sign certificate
	csr, err := rcc.sendCSR(or, certOpts.Namespace, certOpts.Instance, rawCSR, csrName)
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

	csrCh, err := rcc.kubeCli.CertificatesV1beta1().CertificateSigningRequests().Watch(watchReq)
	if err != nil {
		klog.Errorf("error watch CSR for [%s/%s]: %s", certOpts.Namespace, certOpts.Instance, csrName)
		return err
	}

	watchCh := csrCh.ResultChan()
	for {
		select {
		case <-tick:
			klog.Infof("CSR still not approved for [%s/%s]: %s, retry later", certOpts.Namespace, certOpts.Instance, csrName)
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
				klog.Infof("signed certificate for [%s/%s]: %s", certOpts.Namespace, certOpts.Instance, csrName)

				// save signed certificate and key to secret
				err = rcc.secControl.Create(or, certOpts, updatedCSR.Status.Certificate, key)
				if err == nil {
					// cleanup the approved csr
					delOpts := &types.DeleteOptions{TypeMeta: types.TypeMeta{Kind: "CertificateSigningRequest"}}
					return rcc.kubeCli.CertificatesV1beta1().CertificateSigningRequests().Delete(csrName, delOpts)
				}
				return err
			}
			continue
		}
	}
}

func (rcc *realCertControl) getCSR(ns string, instance string, csrName string) (*capi.CertificateSigningRequest, error) {
	csr, err := rcc.csrLister.Get(csrName)
	if err != nil && apierrors.IsNotFound(err) {
		// it's supposed to be not found
		return nil, nil
	}
	if err != nil {
		// something else went wrong
		return nil, err
	}

	labelTemp := label.New()
	if csr.Labels[label.NamespaceLabelKey] == ns &&
		csr.Labels[label.ManagedByLabelKey] == labelTemp[label.ManagedByLabelKey] &&
		csr.Labels[label.InstanceLabelKey] == instance {
		return csr, nil
	}
	return nil, fmt.Errorf("CSR %s/%s already exist, but not created by tidb-operator, skip it", ns, csrName)
}

func (rcc *realCertControl) sendCSR(or metav1.OwnerReference, ns, instance string, rawCSR []byte, csrName string) (*capi.CertificateSigningRequest, error) {
	var csr *capi.CertificateSigningRequest

	// check for exist CSR, overwrite if it was created by operator, otherwise block the process
	csr, err := rcc.getCSR(ns, instance, csrName)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSR for [%s/%s]: %s, error: %v", ns, instance, csrName, err)
	}

	if csr != nil {
		klog.Infof("found exist CSR %s/%s created by tidb-operator, overwriting", ns, csrName)
		delOpts := &types.DeleteOptions{TypeMeta: types.TypeMeta{Kind: "CertificateSigningRequest"}}
		err := rcc.kubeCli.CertificatesV1beta1().CertificateSigningRequests().Delete(csrName, delOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to delete exist old CSR for [%s/%s]: %s, error: %v", ns, instance, csrName, err)
		}
		klog.Infof("exist old CSR deleted for [%s/%s]: %s", ns, instance, csrName)
		return rcc.sendCSR(or, ns, instance, rawCSR, csrName)
	}

	csrLabels := label.New().Instance(instance).Labels()
	csr = &capi.CertificateSigningRequest{
		TypeMeta: types.TypeMeta{Kind: "CertificateSigningRequest"},
		ObjectMeta: types.ObjectMeta{
			Name:            csrName,
			Labels:          csrLabels,
			OwnerReferences: []metav1.OwnerReference{or},
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

	resp, err := rcc.kubeCli.CertificatesV1beta1().CertificateSigningRequests().Create(csr)
	if err != nil {
		return resp, fmt.Errorf("failed to create CSR for [%s/%s]: %s, error: %v", ns, instance, csrName, err)
	}
	klog.Infof("CSR created for [%s/%s]: %s", ns, instance, csrName)
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

func (rcc *realCertControl) CheckSecret(ns string, secretName string) bool {
	return rcc.secControl.Check(ns, secretName)
}

var _ CertControlInterface = &realCertControl{}

type FakeCertControl struct {
	realCertControl
}

func NewFakeCertControl(
	kubeCli kubernetes.Interface,
	csrLister certlisters.CertificateSigningRequestLister,
	secControl SecretControlInterface,
) CertControlInterface {
	return &realCertControl{
		kubeCli:    kubeCli,
		csrLister:  csrLister,
		secControl: secControl,
	}
}
