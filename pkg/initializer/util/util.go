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
	certUtil "github.com/pingcap/tidb-operator/pkg/util"
	certificates "k8s.io/api/certificates/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"time"
)

func GenerateCertSecretAndCSRForService(serviceName, namespace, ownerPodName string, kubeCli kubernetes.Interface, timeout time.Duration) (*core.Secret, *certificates.CertificateSigningRequest, error) {

	key, err := certUtil.GenerateRSAKey()
	if err != nil {
		return nil, nil, err
	}
	ownerRefs, err := getOwnerReferences(ownerPodName, namespace, kubeCli)
	if err != nil {
		return nil, nil, err
	}
	secretName := SecretNameForServiceCert(serviceName)
	approvedCSR, err := createApprovedCSR(serviceName, namespace, ownerRefs, kubeCli, key, timeout)
	if err != nil {
		return nil, nil, err
	}
	klog.Infof("success to apply CSR for service[%s/%s]", namespace, serviceName)

	secret, err := GenerateCertSecret(serviceName, secretName, namespace, ownerRefs, approvedCSR, kubeCli, key, timeout)
	if err != nil {
		return nil, nil, err
	}
	klog.Infof("success to apply secret for service[%s/%s]", namespace, serviceName)

	return secret, approvedCSR, nil
}

func getOwnerReferences(podName, namespace string, kubeCli kubernetes.Interface) ([]metav1.OwnerReference, error) {
	pod, err := kubeCli.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pod.OwnerReferences, nil
}
