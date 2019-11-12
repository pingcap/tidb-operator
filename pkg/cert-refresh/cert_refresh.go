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

package cert_refresh

import (
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/label"
	certUtil "github.com/pingcap/tidb-operator/pkg/util"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"time"
)

type RefreshConfig struct {
	Image                string
	OwnerVersion         string
	OwnerKind            string
	OwnerName            string
	OwnerUid             string
	RefreshIntervalDays  int
	WebhookAdmissionName string
	Timeout              int
}

type RefreshManager struct {
	kubecli kubernetes.Interface
	config  *RefreshConfig
}

const (
	configMapName  = "tidb-operator-initializer-config"
	serviceAccount = "tidb-operator-initializer-sa"
)

func NewRefreshManager(kubecli kubernetes.Interface, config *RefreshConfig) *RefreshManager {
	return &RefreshManager{
		kubecli: kubecli,
		config:  config,
	}
}

// RefreshManager list all the secrets created by Initializer and check whether it need refresh ca cert
func (rm *RefreshManager) Run(podName, namespace string) error {
	list, err := rm.checkCertsNeedRefresh(namespace, rm.config.RefreshIntervalDays)
	if err != nil {
		return err
	}
	err = rm.setConfigOwnerReferences(podName, namespace)
	if err != nil {
		return err
	}
	for _, component := range list {
		job := newInitializerJob(namespace, component, rm.config)
		_, err := rm.kubecli.BatchV1().Jobs(namespace).Create(job)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rm *RefreshManager) setConfigOwnerReferences(podName, namespace string) error {
	pod, err := rm.kubecli.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	for _, reference := range pod.OwnerReferences {
		if reference.Kind == "Job" {
			rm.config.OwnerName = reference.Name
			rm.config.OwnerVersion = reference.APIVersion
			rm.config.OwnerKind = reference.Kind
			rm.config.OwnerUid = string(reference.UID)
			return nil
		}
	}
	return fmt.Errorf("pod[%s/%s] has no OwnerReferences", namespace, podName)
}

// check certs of each component in secrets created by Tidb-Operator-Initializer
// and return all the component who needs to refresh cert
func (rm *RefreshManager) checkCertsNeedRefresh(namespace string, refreshIntervalDays int) (refreshList []string, err error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			label.ComponentLabelKey: "tidb-operator-initializer",
		},
	})
	if err != nil {
		return nil, err
	}

	secretList, err := rm.kubecli.CoreV1().Secrets(namespace).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	for _, secret := range secretList.Items {
		value, existed := secret.Data["cert.pem"]
		if existed {
			cert, err := certUtil.DecodeCertPem(value)
			if err != nil {
				return nil, err
			}
			if certUtil.IsCertificateNeedRefresh(cert, refreshIntervalDays*24) {
				componentByte, existed := secret.Labels[label.CertServiceKey]
				if existed {
					refreshList = append(refreshList, string(componentByte[:]))
				}
			}
		}
	}
	return refreshList, nil
}

func newInitializerJob(namespace, component string, config *RefreshConfig) *batch.Job {
	now := time.Now()
	year, month, day := now.Date()

	name := fmt.Sprintf("refresh-cert-%s-%d-%d-%d", component, year, month, day)
	job := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: config.OwnerVersion,
					Kind:       config.OwnerKind,
					Name:       config.OwnerName,
					UID:        types.UID(config.OwnerUid),
				},
			},
		},
		Spec: batch.JobSpec{
			BackoffLimit:            func() *int32 { a := int32(10); return &a }(),
			TTLSecondsAfterFinished: func() *int32 { a := int32(60); return &a }(),
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: core.PodSpec{
					ServiceAccountName: serviceAccount,
					RestartPolicy:      core.RestartPolicyOnFailure,
					Containers: []core.Container{
						{
							Name:            "refresh-cert-job",
							Image:           config.Image,
							ImagePullPolicy: core.PullIfNotPresent,
							Command: []string{
								"/usr/local/bin/tidb-operator-initializer",
								fmt.Sprintf("-component=%s", component),
								fmt.Sprintf("-timeout=%d", config.Timeout),
							},
							Env: []core.EnvVar{
								{
									Name: "NAMESPACE",
									ValueFrom: &core.EnvVarSource{
										FieldRef: &core.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name: "POD_NAME",
									ValueFrom: &core.EnvVarSource{
										FieldRef: &core.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	switch component {
	case config.WebhookAdmissionName:
		job.Spec.Template.Spec.Containers[0].Command = append(job.Spec.Template.Spec.Containers[0].Command, "-webhook-enabled=true")
	}
	return job
}
