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
	"crypto/rsa"
	"fmt"
	certUtil "github.com/pingcap/tidb-operator/pkg/util"
	certificates "k8s.io/api/certificates/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"time"
)

func CSRNameForService(serviceName, namespace string) string {
	return fmt.Sprintf("%s.%s", serviceName, namespace)
}

func createApprovedCSR(serviceName, namespace string, ownerRefs []metav1.OwnerReference, kubeCli kubernetes.Interface, key *rsa.PrivateKey, timeout time.Duration) (*certificates.CertificateSigningRequest, error) {

	csrName := CSRNameForService(serviceName, namespace)
	err := waitUntilCsrDelete(csrName, kubeCli, timeout)
	if err != nil {
		return nil, err
	}

	klog.Infof("success to delete csr[%s]", csrName)
	dnsNames := []string{
		serviceName,
		fmt.Sprintf("%s.%s", serviceName, namespace),
		fmt.Sprintf("%s.%s.svc", serviceName, namespace),
	}
	requestPem, err := certUtil.GenerateCSRPem(key, serviceName, dnsNames)
	if err != nil {
		return nil, err
	}

	csr := generateCSR(serviceName, namespace, ownerRefs, requestPem)
	createdCSR, err := kubeCli.CertificatesV1beta1().CertificateSigningRequests().Create(csr)
	if err != nil {
		return nil, err
	}
	klog.Infof("success to create csr[%s]", csrName)

	csr, err = approveCSR(createdCSR, kubeCli, timeout)
	if err != nil {
		return nil, err
	}
	err = waitUntilCsrApproved(csrName, kubeCli, timeout)
	if err != nil {
		return nil, err
	}
	klog.Infof("success to approve csr[%s]", csrName)

	approvedCSR, err := kubeCli.CertificatesV1beta1().CertificateSigningRequests().Get(csrName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return approvedCSR, nil
}

func approveCSR(csr *certificates.CertificateSigningRequest, kubeCli kubernetes.Interface, timeout time.Duration) (*certificates.CertificateSigningRequest, error) {
	err := waitUntilCsrExist(csr.Name, kubeCli, timeout)
	if err != nil {
		return nil, err
	}
	csr = generateApproveCSR(csr)
	approvedCSR, err := kubeCli.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(csr)
	if err != nil {
		return nil, err
	}
	return approvedCSR, nil
}

func generateCSR(service, namespace string, ownerRefs []metav1.OwnerReference, requestPem []byte) *certificates.CertificateSigningRequest {
	csr := &certificates.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s.%s", service, namespace),
			OwnerReferences: ownerRefs,
		},
		Spec: certificates.CertificateSigningRequestSpec{
			Request: requestPem,
			Usages: []certificates.KeyUsage{
				certificates.UsageDigitalSignature,
				certificates.UsageServerAuth,
				certificates.UsageKeyEncipherment,
			},
			Groups: []string{
				"system:authenticated",
			},
		},
	}
	return csr
}

func waitUntilCsrDelete(csrName string, kubeCli kubernetes.Interface, timeout time.Duration) error {
	err := kubeCli.CertificatesV1beta1().CertificateSigningRequests().Delete(csrName, &metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return wait.PollImmediate(1*time.Second, timeout, func() (done bool, err error) {
		_, err = kubeCli.CertificatesV1beta1().CertificateSigningRequests().Get(csrName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
}

func waitUntilCsrExist(csrName string, kubeCli kubernetes.Interface, timeout time.Duration) error {

	return wait.PollImmediate(1*time.Second, timeout, func() (done bool, err error) {
		_, err = kubeCli.CertificatesV1beta1().CertificateSigningRequests().Get(csrName, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

func waitUntilCsrApproved(csrName string, kubeCli kubernetes.Interface, timeout time.Duration) error {
	return wait.PollImmediate(1*time.Second, timeout, func() (done bool, err error) {
		csr, err := kubeCli.CertificatesV1beta1().CertificateSigningRequests().Get(csrName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if len(csr.Status.Certificate) > 0 {
			return true, nil
		}
		return false, nil
	})
}

func generateApproveCSR(csr *certificates.CertificateSigningRequest) *certificates.CertificateSigningRequest {
	csr.Status.Conditions = append(csr.Status.Conditions,
		certificates.CertificateSigningRequestCondition{
			Type:           certificates.CertificateApproved,
			LastUpdateTime: metav1.Now(),
		},
	)
	return csr
}
