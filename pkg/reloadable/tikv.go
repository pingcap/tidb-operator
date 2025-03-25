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
)

// CheckTiKV returns whether changes are reloadable. No change also means reloadable.
// We assume that only when the instance template is changed, the pod spec will change.
// Instance copies the template spec from group directly,
// so we can use the template of instance directly as the last template.
// This function is used in the group controller.
func CheckTiKV(group *v1alpha1.TiKVGroup, instance *v1alpha1.TiKV) bool {
	groupTmpl := &group.Spec.Template.Spec
	instanceTmpl := &instance.Spec.TiKVTemplateSpec

	return equalTiKVTemplate(groupTmpl, instanceTmpl)
}

// CheckTiKVPod returns whether changes are reloadable. No change also means reloadable.
// Pod records the last template in annotation.
// We use a same way to check whether changes are reloadable.
// This function is used in the instance controller.
func CheckTiKVPod(instance *v1alpha1.TiKV, pod *corev1.Pod) bool {
	lastInstanceTemplate := pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate]

	spec := &v1alpha1.TiKVTemplateSpec{}
	if err := json.Unmarshal([]byte(lastInstanceTemplate), spec); err != nil {
		// last template is not found or unexpectedly changed
		return true
	}

	instanceTmpl := &instance.Spec.TiKVTemplateSpec

	return equalTiKVTemplate(instanceTmpl, spec)
}

func EncodeLastTiKVTemplate(instance *v1alpha1.TiKV, pod *corev1.Pod) error {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	instanceTmpl := &instance.Spec.TiKVTemplateSpec
	data, err := json.Marshal(instanceTmpl)
	if err != nil {
		return err
	}
	pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate] = string(data)

	return nil
}

func MustEncodeLastTiKVTemplate(instance *v1alpha1.TiKV, pod *corev1.Pod) {
	if err := EncodeLastTiKVTemplate(instance, pod); err != nil {
		panic("cannot encode tikv template: " + err.Error())
	}
}

// convertTiKVTemplate will ignore some fields
// TODO: set default for some empty fields
func convertTiKVTemplate(tmpl *v1alpha1.TiKVTemplateSpec) *v1alpha1.TiKVTemplateSpec {
	newTmpl := tmpl.DeepCopy()

	if newTmpl.PreStop != nil {
		newTmpl.PreStop.Image = nil
	}

	newTmpl.Volumes = convertVolumes(newTmpl.Volumes)
	newTmpl.Overlay = convertOverlay(newTmpl.Overlay)

	return newTmpl
}

func equalTiKVTemplate(p, c *v1alpha1.TiKVTemplateSpec) bool {
	p = convertTiKVTemplate(p)
	c = convertTiKVTemplate(c)
	// not equal only when current strategy is Restart and config is changed
	switch c.UpdateStrategy.Config {
	case v1alpha1.ConfigUpdateStrategyRestart:
		if p.Config != c.Config {
			return false
		}
	}

	// ignore these fields
	p.Config = ""
	c.Config = ""
	p.UpdateStrategy.Config = ""
	c.UpdateStrategy.Config = ""

	return apiequality.Semantic.DeepEqual(p, c)
}
