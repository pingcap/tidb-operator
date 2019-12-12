// Copyright 2019. PingCAP, Inc.
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
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/klog"
)

const (
	// LastAppliedPodTemplate is annotation key of the last applied pod template
	LastAppliedPodTemplate = "pingcap.com/last-applied-podtemplate"

	// LastAppliedConfigAnnotation is annotation key of last applied configuration
	LastAppliedConfigAnnotation = "pingcap.com/last-applied-configuration"
)

// SetDeploymentLastAppliedPodTemplate set last pod template to Deployment's annotation
func SetDeploymentLastAppliedPodTemplate(dep *appsv1.Deployment) error {
	b, err := json.Marshal(dep.Spec.Template.Spec)
	if err != nil {
		return err
	}
	applied := string(b)
	if dep.Annotations == nil {
		dep.Annotations = map[string]string{}
	}
	dep.Annotations[LastAppliedPodTemplate] = applied
	return nil
}

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
func DeploymentPodSpecChanged(new *appsv1.Deployment, old *appsv1.Deployment) bool {
	lastAppliedPodTemplate, err := GetDeploymentLastAppliedPodTemplate(old)
	if err != nil {
		klog.Warningf("error get last-applied-config of deployment %s/%s: %v", old.Namespace, old.Name, err)
		return true
	}
	return !apiequality.Semantic.DeepEqual(new.Spec.Template.Spec, lastAppliedPodTemplate)
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
func ServiceEqual(new, old *corev1.Service) (bool, error) {
	oldSpec := corev1.ServiceSpec{}
	if lastAppliedConfig, ok := old.Annotations[LastAppliedConfigAnnotation]; ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldSpec)
		if err != nil {
			klog.Errorf("unmarshal ServiceSpec: [%s/%s]'s applied config failed,error: %v", old.GetNamespace(), old.GetName(), err)
			return false, err
		}
		return apiequality.Semantic.DeepEqual(oldSpec, new.Spec), nil
	}
	return false, nil
}
