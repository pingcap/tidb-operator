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
	"github.com/pingcap/tidb-operator/v2/pkg/features"
)

// CheckResourceManager returns whether changes are reloadable. No change also means reloadable.
// This function is used in the group controller.
func CheckResourceManager(group *v1alpha1.ResourceManagerGroup, instance *v1alpha1.ResourceManager) bool {
	groupTmpl := &group.Spec.Template
	instanceTmpl := templateFromResourceManager(instance)

	return equalResourceManagerTemplate(groupTmpl, instanceTmpl) &&
		features.Reloadable(meta.ComponentResourceManager, group.Spec.Features, instance.Spec.Features)
}

// CheckResourceManagerPod returns whether changes are reloadable. No change also means reloadable.
// This function is used in the instance controller.
func CheckResourceManagerPod(instance *v1alpha1.ResourceManager, pod *corev1.Pod) bool {
	lastInstanceTemplate := pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate]
	lastFeatures := pod.Annotations[v1alpha1.AnnoKeyFeatures]

	tmpl := &v1alpha1.ResourceManagerTemplate{}
	if err := json.Unmarshal([]byte(lastInstanceTemplate), tmpl); err != nil {
		// last template is not found or unexpectedly changed
		return true
	}

	instanceTmpl := templateFromResourceManager(instance)

	return equalResourceManagerTemplate(instanceTmpl, tmpl) &&
		features.Reloadable(meta.ComponentResourceManager, instance.Spec.Features, decodeFeatures(lastFeatures))
}

func EncodeLastResourceManagerTemplate(instance *v1alpha1.ResourceManager, pod *corev1.Pod) error {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	instanceTmpl := templateFromResourceManager(instance)

	data, err := json.Marshal(instanceTmpl)
	if err != nil {
		return err
	}
	pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate] = string(data)
	pod.Annotations[v1alpha1.AnnoKeyFeatures] = encodeFeatures(instance.Spec.Features)

	return nil
}

func MustEncodeLastResourceManagerTemplate(instance *v1alpha1.ResourceManager, pod *corev1.Pod) {
	if err := EncodeLastResourceManagerTemplate(instance, pod); err != nil {
		panic("cannot encode resource manager template: " + err.Error())
	}
}

// TODO: ignore inherited labels and annotations
func templateFromResourceManager(rm *v1alpha1.ResourceManager) *v1alpha1.ResourceManagerTemplate {
	return &v1alpha1.ResourceManagerTemplate{
		ObjectMeta: v1alpha1.ObjectMeta{
			Labels:      rm.Labels,
			Annotations: rm.Annotations,
		},
		Spec: rm.Spec.ResourceManagerTemplateSpec,
	}
}

// convertResourceManagerTemplate will ignore some fields
// TODO: set default for some empty fields
func convertResourceManagerTemplate(tmpl *v1alpha1.ResourceManagerTemplate) *v1alpha1.ResourceManagerTemplate {
	newTmpl := tmpl.DeepCopy()

	newTmpl.Labels = convertLabels(newTmpl.Labels)
	newTmpl.Annotations = convertAnnotations(newTmpl.Annotations)
	newTmpl.Spec.Volumes = convertVolumes(newTmpl.Spec.Volumes)
	newTmpl.Spec.Overlay = convertOverlay(newTmpl.Spec.Overlay)

	return newTmpl
}

func equalResourceManagerTemplate(c, p *v1alpha1.ResourceManagerTemplate) bool {
	p = convertResourceManagerTemplate(p)
	c = convertResourceManagerTemplate(c)
	// not equal only when current strategy is Restart and config is changed
	if c.Spec.UpdateStrategy.Config == v1alpha1.ConfigUpdateStrategyRestart && p.Spec.Config != c.Spec.Config {
		return false
	}

	// ignore these fields
	p.Spec.Config = ""
	c.Spec.Config = ""
	p.Spec.UpdateStrategy.Config = ""
	c.Spec.UpdateStrategy.Config = ""

	return apiequality.Semantic.DeepEqual(p, c)
}
