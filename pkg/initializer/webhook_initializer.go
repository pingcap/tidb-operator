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

package initializer

import (
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/annotation"
	"github.com/pingcap/tidb-operator/pkg/initializer/util"
	certUtil "github.com/pingcap/tidb-operator/pkg/util"
	certificates "k8s.io/api/certificates/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	CaCertKey = "cert.pem"
)

// Webhook Initializer should do following setups:
//  - create or refresh Secret and CSR for ca cert
//  - update webhook server
//  - update validationAdmissionConfiguration
func (initializer *Initializer) webhookResourceInitializer(namespace string) error {

	if !initializer.config.WebhookEnabled {
		return nil
	}
	admissionWebhookName := initializer.config.AdmissionWebhookName
	ownerPodName := initializer.config.OwnerPodName
	timeout := initializer.config.Timeout

	klog.Info("initializer start to generate resources for webhook")

	secret, _, err := util.GenerateCertSecretAndCSRForService(admissionWebhookName, namespace, ownerPodName, initializer.kubeCli, timeout)
	if err != nil {
		return err
	}

	err = initializer.updateWebhookServer(admissionWebhookName, namespace, secret)
	if err != nil {
		klog.Errorf("failed to update webhook server[%s/%s],%v", namespace, admissionWebhookName, err)
		return err
	}
	klog.Infof("success to update webhook server[%s/%s]", namespace, admissionWebhookName)

	return nil
}

// update Webhook server to make sure the latest secret would be used.
func (initializer *Initializer) updateWebhookServer(serviceName, namespace string, secret *core.Secret) error {

	server, err := initializer.kubeCli.ExtensionsV1beta1().Deployments(namespace).Get(serviceName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if server.Spec.Template.Annotations == nil {
		server.Spec.Template.Annotations = map[string]string{}
	}

	server.Spec.Template.Annotations[annotation.CACertChecksumKey] = certUtil.Checksum(secret.Data[CaCertKey])
	_, err = initializer.kubeCli.ExtensionsV1beta1().Deployments(namespace).Update(server)
	if err != nil {
		return err
	}
	return nil
}

func (initializer *Initializer) setOwnerReferences(podName, namespace string, secret *core.Secret, csr *certificates.CertificateSigningRequest) error {
	pod, err := initializer.kubeCli.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	secret.OwnerReferences = []metav1.OwnerReference{}
	for _, reference := range pod.OwnerReferences {
		if reference.Kind == "Job" {
			secret.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: reference.APIVersion,
					Name:       reference.Name,
					Kind:       reference.Kind,
					UID:        reference.UID,
				},
			}
			csr.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: reference.APIVersion,
					Name:       reference.Name,
					Kind:       reference.Kind,
					UID:        reference.UID,
				},
			}
			return nil
		}
	}
	return fmt.Errorf("pod[%s/%s] has no OwnerReferences", namespace, podName)
}
