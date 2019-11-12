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
	"github.com/pingcap/tidb-operator/pkg/label"
	certUtil "github.com/pingcap/tidb-operator/pkg/util"
	certificates "k8s.io/api/certificates/v1beta1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"time"
)

func SecretNameForServiceCert(serviceName string) string {
	return fmt.Sprintf("%s-cert", serviceName)
}

func GenerateCertSecret(serviceName, secretName, namespace string, ownerRefs []metav1.OwnerReference, approvedCSR *certificates.CertificateSigningRequest, kubeCli kubernetes.Interface, key *rsa.PrivateKey, timeout time.Duration) (*core.Secret, error) {

	certPem := approvedCSR.Status.Certificate
	err := waitUtilSecretDelete(secretName, namespace, kubeCli, timeout)
	if err != nil {
		return nil, err
	}
	keyPem := certUtil.GenerateKeyPEM(key)
	secret := generateCertSecret(serviceName, secretName, ownerRefs, certPem, keyPem)
	createdSecret, err := kubeCli.CoreV1().Secrets(namespace).Create(secret)
	if err != nil {
		return nil, err
	}
	return createdSecret, nil
}

func generateCertSecret(serviceName, secretName string, ownerRefs []metav1.OwnerReference, certPem []byte, keyPem []byte) *core.Secret {
	secret := core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            secretName,
			OwnerReferences: ownerRefs,
			Labels: map[string]string{
				label.ManagedByLabelKey: "tidb-operator",
				label.ComponentLabelKey: "tidb-operator-initializer",
				label.CertServiceKey:    serviceName,
			},
		},
		Data: map[string][]byte{
			"cert.pem": certPem,
			"key.pem":  keyPem,
		},
	}
	return &secret
}

func waitUtilSecretDelete(secretName, namespace string, kubeCli kubernetes.Interface, timeout time.Duration) error {
	err := kubeCli.CoreV1().Secrets(namespace).Delete(secretName, &metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("delete secret[%s/%s] failed", namespace, secretName)
	}
	return wait.PollImmediate(1*time.Second, timeout, func() (done bool, err error) {
		_, err = kubeCli.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
}
