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
	certUtils "github.com/pingcap/tidb-operator/pkg/util"
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
func (initializer *Initializer) webhookResourceInitializer(podName, namespace string, days int) error {

	if !WebhookEnabled {
		return nil
	}
	klog.Info("initializer start to generate resources for webhook")

	err := GenerateSecretAndCSR(AdmissionWebhookName, namespace, days)
	if err != nil {
		return err
	}
	klog.Infof("success to apply CA cert for service[%s/%s]", namespace, AdmissionWebhookName)

	secret, err := initializer.kubeCli.CoreV1().Secrets(namespace).Get(SecretNameForServiceCert(AdmissionWebhookName), metav1.GetOptions{})
	if err != nil {
		return err
	}

	csrName := fmt.Sprintf("%s.%s-csr", AdmissionWebhookName, namespace)

	csr, err := initializer.kubeCli.CertificatesV1beta1().CertificateSigningRequests().Get(csrName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	err = initializer.setOwnerReferences(podName, namespace, secret, csr)
	if err != nil {
		return err
	}

	_, err = initializer.kubeCli.CoreV1().Secrets(namespace).Update(secret)
	if err != nil {
		return err
	}
	_, err = initializer.kubeCli.CertificatesV1beta1().CertificateSigningRequests().Update(csr)
	if err != nil {
		return err
	}

	err = initializer.updateWebhookServer(namespace, secret)
	if err != nil {
		klog.Errorf("failed to update webhook server[%s/%s],%v", namespace, AdmissionWebhookName, err)
		return err
	}
	klog.Infof("success to update webhook server[%s/%s]", namespace, AdmissionWebhookName)

	err = initializer.updateValidationAdmissionConfiguration(secret)
	if err != nil {
		klog.Errorf("failed to update validation admission config[%s/%s],%v", namespace, VACNameForService(namespace, AdmissionWebhookName), err)
		return err
	}
	klog.Infof("success to update validation admission config[%s/%s]", VACNameForService(namespace, AdmissionWebhookName), err)
	return nil
}

// update tidb-operator validation webhook server
func (initializer *Initializer) updateValidationAdmissionConfiguration(secret *core.Secret) error {

	conf, err := initializer.kubeCli.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(AdmissionWebhookName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for id, webhook := range conf.Webhooks {
		webhook.ClientConfig.CABundle = secret.Data[CaCertKey]
		conf.Webhooks[id] = webhook
	}
	_, err = initializer.kubeCli.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Update(conf)
	if err != nil {
		return err
	}
	return nil
}

// update Webhook server to make sure the latest secret would be used.
func (initializer *Initializer) updateWebhookServer(namespace string, secret *core.Secret) error {

	server, err := initializer.kubeCli.ExtensionsV1beta1().Deployments(namespace).Get(AdmissionWebhookName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if server.Spec.Template.Annotations == nil {
		server.Spec.Template.Annotations = map[string]string{
			"checksum/cert": certUtils.Checksum(secret.Data[CaCertKey]),
		}
	} else {
		server.Spec.Template.Annotations[annotation.CACertChecksumKey] = certUtils.Checksum(secret.Data[CaCertKey])
	}

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
