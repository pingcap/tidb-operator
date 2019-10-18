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
	"k8s.io/api/admissionregistration/v1beta1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	glog "k8s.io/klog"
)

const (
	SecretName                  = "admission-controller-webhook-certs"
	admissionConfigurationName  = "tidb-operator-validation-admission-cfg"
	admissionWebhookServerName  = "admission-controller-webhook"
	admissionWebhookServiceName = "admission-controller-webhook-svc"
)

type Initializer struct {
	kubeCli kubernetes.Interface
}

func NewInitializer(kubecli kubernetes.Interface) *Initializer {
	return &Initializer{
		kubeCli: kubecli,
	}
}

// Initializer should do following setups:
//  - create or retain Secret for ca cert
//  - update webhook server
//  - update validationAdmissionConfiguration
func (initializer *Initializer) Run(namespace string) error {

	secret, err := initializer.applySecret(namespace)
	if err != nil {
		glog.Errorf("failed to apply Secret,%v", err)
		return err
	}
	glog.Info("success to apply cert secret")

	err = initializer.updateWebhookServer(namespace, secret)
	if err != nil {
		glog.Errorf("failed to update webhook server,%v", err)
		return err
	}
	glog.Info("success to update webhook server ")

	err = initializer.updateValidationAdmissionConfiguration(namespace, secret)
	if err != nil {
		glog.Errorf("failed to update webhook config,%v", err)
		return err
	}
	glog.Info("success to update webhook validationadmissionconfiguration ")

	return nil
}

// create secret if not existed for ca cert
func (initializer *Initializer) applySecret(namespace string) (*core.Secret, error) {

	oldSecret, err := initializer.kubeCli.CoreV1().Secrets(namespace).Get(SecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			newSecret, err := initializer.GenerateNewSecretForInitializer(namespace)
			if err != nil {
				return nil, err
			}
			newSecret, err = initializer.kubeCli.CoreV1().Secrets(namespace).Create(newSecret)
			if err != nil {
				return nil, err
			}
			return newSecret, nil
		}
		return nil, err
	}
	return oldSecret, nil
}

// update tidb-operator validation webhook server
func (initializer *Initializer) updateValidationAdmissionConfiguration(namespace string, secret *core.Secret) error {
	conf, err := initializer.kubeCli.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(admissionConfigurationName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	f := v1beta1.Fail
	for id, webhook := range conf.Webhooks {
		webhook.ClientConfig.CABundle = secret.Data["cert.pem"]
		webhook.FailurePolicy = &f
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

	server, err := initializer.kubeCli.ExtensionsV1beta1().Deployments(namespace).Get(admissionWebhookServerName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	server.Spec.Template.Annotations = map[string]string{
		"checksum/config": Checksum(secret.Data["cert.pem"]),
	}
	_, err = initializer.kubeCli.ExtensionsV1beta1().Deployments(namespace).Update(server)
	if err != nil {
		return err
	}
	return nil
}
