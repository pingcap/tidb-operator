// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reloadable

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/features"
)

// CheckTiProxy returns whether changes are reloadable. No change also means reloadable.
// We assume that only when the instance template is changed, the pod spec will change.
// Instance copies the template spec from group directly,
// so we can use the template of instance directly as the last template.
// This function is used in the group controller.
func CheckTiProxy(group *v1alpha1.TiProxyGroup, instance *v1alpha1.TiProxy) bool {
	groupTmpl := &group.Spec.Template
	instanceTmpl := templateFromTiProxy(instance)

	return equalTiProxyTemplate(groupTmpl, instanceTmpl) &&
		features.Reloadable(meta.ComponentTiProxy, group.Spec.Features, instance.Spec.Features)
}

// CheckTiProxyPod returns whether changes are reloadable. No change also means reloadable.
// Pod records the last template in annotation.
// We use a same way to check whether changes are reloadable.
// This function is used in the instance controller.
func CheckTiProxyPod(instance *v1alpha1.TiProxy, pod *corev1.Pod) bool {
	lastInstanceTemplate := pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate]
	lastFeatures := pod.Annotations[v1alpha1.AnnoKeyFeatures]

	tmpl := &v1alpha1.TiProxyTemplate{}
	if err := json.Unmarshal([]byte(lastInstanceTemplate), tmpl); err != nil {
		// last template is not found or unexpectedly changed
		return true
	}

	instanceTmpl := templateFromTiProxy(instance)

	return equalTiProxyTemplate(instanceTmpl, tmpl) &&
		features.Reloadable(meta.ComponentTiProxy, instance.Spec.Features, decodeFeatures(lastFeatures))
}

func EncodeLastTiProxyTemplate(instance *v1alpha1.TiProxy, pod *corev1.Pod) error {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	instanceTmpl := templateFromTiProxy(instance)

	data, err := json.Marshal(instanceTmpl)
	if err != nil {
		return err
	}
	pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate] = string(data)
	pod.Annotations[v1alpha1.AnnoKeyFeatures] = encodeFeatures(instance.Spec.Features)

	return nil
}

func MustEncodeLastTiProxyTemplate(instance *v1alpha1.TiProxy, pod *corev1.Pod) {
	if err := EncodeLastTiProxyTemplate(instance, pod); err != nil {
		panic("cannot encode tiproxy template: " + err.Error())
	}
}

// TODO: ignore inherited labels and annotations
func templateFromTiProxy(proxy *v1alpha1.TiProxy) *v1alpha1.TiProxyTemplate {
	return &v1alpha1.TiProxyTemplate{
		ObjectMeta: v1alpha1.ObjectMeta{
			Labels:      proxy.Labels,
			Annotations: proxy.Annotations,
		},
		Spec: proxy.Spec.TiProxyTemplateSpec,
	}
}

// convertTiProxyTemplate will ignore some fields
// TODO: set default for some empty fields
func convertTiProxyTemplate(tmpl *v1alpha1.TiProxyTemplate) *v1alpha1.TiProxyTemplate {
	newTmpl := tmpl.DeepCopy()

	newTmpl.Labels = convertLabels(newTmpl.Labels)
	newTmpl.Annotations = convertAnnotations(newTmpl.Annotations)

	// server labels can be updated dynamically
	newTmpl.Spec.Server.Labels = nil

	newTmpl.Spec.Volumes = convertVolumes(newTmpl.Spec.Volumes)
	newTmpl.Spec.Overlay = convertOverlay(newTmpl.Spec.Overlay)

	return newTmpl
}

func equalTiProxyTemplate(p, c *v1alpha1.TiProxyTemplate) bool {
	p = convertTiProxyTemplate(p)
	c = convertTiProxyTemplate(c)
	// not equal only when current strategy is Restart and config is changed
	switch c.Spec.UpdateStrategy.Config {
	case v1alpha1.ConfigUpdateStrategyRestart:
		if p.Spec.Config != c.Spec.Config {
			return false
		}
	}

	// ignore these fields
	p.Spec.Config = ""
	c.Spec.Config = ""
	p.Spec.UpdateStrategy.Config = ""
	c.Spec.UpdateStrategy.Config = ""

	return apiequality.Semantic.DeepEqual(p, c)
}
