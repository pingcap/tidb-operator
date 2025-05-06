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

// CheckTSO returns whether changes are reloadable. No change also means reloadable.
// We assume that only when the instance template is changed, the pod spec will change.
// Instance copies the template spec from group directly,
// so we can use the template of instance directly as the last template.
// This function is used in the group controller.
func CheckTSO(group *v1alpha1.TSOGroup, instance *v1alpha1.TSO) bool {
	groupTmpl := &group.Spec.Template
	instanceTmpl := templateFromTSO(instance)

	return equalTSOTemplate(groupTmpl, instanceTmpl)
}

// CheckTSOPod returns whether changes are reloadable. No change also means reloadable.
// Pod records the last template in annotation.
// We use a same way to check whether changes are reloadable.
// This function is used in the instance controller.
func CheckTSOPod(instance *v1alpha1.TSO, pod *corev1.Pod) bool {
	lastInstanceTemplate := pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate]

	tmpl := &v1alpha1.TSOTemplate{}
	if err := json.Unmarshal([]byte(lastInstanceTemplate), tmpl); err != nil {
		// last template is not found or unexpectedly changed
		return true
	}

	instanceTmpl := templateFromTSO(instance)

	return equalTSOTemplate(instanceTmpl, tmpl)
}

func EncodeLastTSOTemplate(instance *v1alpha1.TSO, pod *corev1.Pod) error {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	instanceTmpl := templateFromTSO(instance)

	data, err := json.Marshal(instanceTmpl)
	if err != nil {
		return err
	}
	pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate] = string(data)

	return nil
}

func MustEncodeLastTSOTemplate(instance *v1alpha1.TSO, pod *corev1.Pod) {
	if err := EncodeLastTSOTemplate(instance, pod); err != nil {
		panic("cannot encode tso template: " + err.Error())
	}
}

// TODO: ignore inherited labels and annotations
func templateFromTSO(tso *v1alpha1.TSO) *v1alpha1.TSOTemplate {
	return &v1alpha1.TSOTemplate{
		ObjectMeta: v1alpha1.ObjectMeta{
			Labels:      tso.Labels,
			Annotations: tso.Annotations,
		},
		Spec: tso.Spec.TSOTemplateSpec,
	}
}

// convertTSOTemplate will ignore some fields
// TODO: set default for some empty fields
func convertTSOTemplate(tmpl *v1alpha1.TSOTemplate) *v1alpha1.TSOTemplate {
	newTmpl := tmpl.DeepCopy()

	newTmpl.Labels = convertLabels(newTmpl.Labels)
	newTmpl.Annotations = convertAnnotations(newTmpl.Annotations)
	newTmpl.Spec.Volumes = convertVolumes(newTmpl.Spec.Volumes)
	newTmpl.Spec.Overlay = convertOverlay(newTmpl.Spec.Overlay)

	return newTmpl
}

func equalTSOTemplate(p, c *v1alpha1.TSOTemplate) bool {
	p = convertTSOTemplate(p)
	c = convertTSOTemplate(c)
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
