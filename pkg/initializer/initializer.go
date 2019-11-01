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
	certUtils "github.com/pingcap/tidb-operator/pkg/util"
	"k8s.io/api/admissionregistration/v1beta1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	glog "k8s.io/klog"
	"os/exec"
)

const (
	CSRName                     = "admission-controller-webhook-csr"
	SecretName                  = "admission-controller-webhook-certs"
	admissionConfigurationName  = "tidb-operator-validation-admission-cfg"
	admissionWebhookServerName  = "admission-controller-webhook"
	admissionWebhookServiceName = "admission-controller-webhook-svc"
	executePath                 = "/etc/initializer/create-cert-script"
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
//  - create or refresh Secret and CSR for ca cert
//  - update webhook server
//  - update validationAdmissionConfiguration
func (initializer *Initializer) Run(namespace string, refreshIntervalHour int) error {

	secret, err := initializer.applyCertAuthority(namespace, refreshIntervalHour)
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
func (initializer *Initializer) applyCertAuthority(namespace string, refreshIntervalHour int) (*core.Secret, error) {

	glog.Info("start to apply Cert Authority.")
	var secret *core.Secret
	oldSecret, err := initializer.kubeCli.CoreV1().Secrets(namespace).Get(SecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			glog.Info("previous CA Authority is not existed.")
			secret, err = initializer.generateSecretAndCSR(namespace, admissionWebhookServiceName)
			if err != nil {
				return nil, err
			}
			return secret, nil
		}
		return nil, err
	}
	secret = oldSecret
	cert, err := certUtils.DecodeCertPem(oldSecret.Data["cert.pem"])
	if err != nil {
		return nil, err
	}
	if certUtils.IsCertificateNeedRefresh(cert, refreshIntervalHour) {
		glog.Info("previous CA Authority need refresh.")
		secret, err = initializer.generateSecretAndCSR(namespace, admissionWebhookServiceName)
		if err != nil {
			return nil, err
		}
	}
	return secret, err
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
		"checksum/config": certUtils.Checksum(secret.Data["cert.pem"]),
	}
	_, err = initializer.kubeCli.ExtensionsV1beta1().Deployments(namespace).Update(server)
	if err != nil {
		return err
	}
	return nil
}

func (initializer *Initializer) generateSecretAndCSR(namespace, serviceName string) (*core.Secret, error) {

	output, err := exec.Command("/bin/sh", executePath, "-n", namespace, "-s", serviceName).CombinedOutput()
	if err != nil {
		glog.Errorf("execute ca cert failed for service[%s/%s],%v", namespace, serviceName, err)
		return nil, err
	}
	glog.Infof("execute output:%s", string(output[:]))
	secret, err := initializer.kubeCli.CoreV1().Secrets(namespace).Get(fmt.Sprintf("%s-cert", serviceName), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret, nil
}
