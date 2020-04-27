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
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/klog"
)

const (
	// LastAppliedPodTemplate is annotation key of the last applied pod template
	LastAppliedPodTemplate = "pingcap.com/last-applied-podtemplate"

	// LastAppliedConfigAnnotation is annotation key of last applied configuration
	LastAppliedConfigAnnotation = "pingcap.com/last-applied-configuration"
)

// GetDeploymentLastAppliedPodTemplate set last applied pod template from Deployment's annotation
func GetDeploymentLastAppliedPodTemplate(dep *appsv1.Deployment) (*corev1.PodSpec, error) {
	applied, ok := dep.Annotations[LastAppliedPodTemplate]
	if !ok {
		return nil, fmt.Errorf("deployment:[%s/%s] not found spec's apply config", dep.GetNamespace(), dep.GetName())
	}
	podSpec := &corev1.PodSpec{}
	err := json.Unmarshal([]byte(applied), podSpec)
	if err != nil {
		return nil, err
	}
	return podSpec, nil
}

// DeploymentPodSpecChanged checks whether the new deployment differs with the old one's last-applied-config
func DeploymentPodSpecChanged(newDep *appsv1.Deployment, oldDep *appsv1.Deployment) bool {
	lastAppliedPodTemplate, err := GetDeploymentLastAppliedPodTemplate(oldDep)
	if err != nil {
		klog.Warningf("error get last-applied-config of deployment %s/%s: %v", oldDep.Namespace, oldDep.Name, err)
		return true
	}
	return !apiequality.Semantic.DeepEqual(newDep.Spec.Template.Spec, lastAppliedPodTemplate)
}

// SetServiceLastAppliedConfigAnnotation set last applied config info to Service's annotation
func SetServiceLastAppliedConfigAnnotation(svc *corev1.Service) error {
	b, err := json.Marshal(svc.Spec)
	if err != nil {
		return err
	}
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	svc.Annotations[LastAppliedConfigAnnotation] = string(b)
	return nil
}

// ServiceEqual compares the new Service's spec with old Service's last applied config
func ServiceEqual(newSvc, oldSvc *corev1.Service) (bool, error) {
	oldSpec := corev1.ServiceSpec{}
	if lastAppliedConfig, ok := oldSvc.Annotations[LastAppliedConfigAnnotation]; ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldSpec)
		if err != nil {
			klog.Errorf("unmarshal ServiceSpec: [%s/%s]'s applied config failed,error: %v", oldSvc.GetNamespace(), oldSvc.GetName(), err)
			return false, err
		}
		return apiequality.Semantic.DeepEqual(oldSpec, newSvc.Spec), nil
	}
	return false, nil
}

func IngressEqual(newIngress, oldIngres *extensionsv1beta1.Ingress) (bool, error) {
	oldIngressSpec := extensionsv1beta1.IngressSpec{}
	if lastAppliedConfig, ok := oldIngres.Annotations[LastAppliedConfigAnnotation]; ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldIngressSpec)
		if err != nil {
			klog.Errorf("unmarshal IngressSpec: [%s/%s]'s applied config failed,error: %v", oldIngres.GetNamespace(), oldIngres.GetName(), err)
			return false, err
		}
		return apiequality.Semantic.DeepEqual(oldIngressSpec, newIngress.Spec), nil
	}
}
