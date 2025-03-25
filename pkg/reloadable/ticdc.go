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

// CheckTiCDC returns whether changes are reloadable. No change also means reloadable.
// We assume that only when the instance template is changed, the pod spec will change.
// Instance copies the template spec from group directly,
// so we can use the template of instance directly as the last template.
// This function is used in the group controller.
func CheckTiCDC(group *v1alpha1.TiCDCGroup, instance *v1alpha1.TiCDC) bool {
	groupTmpl := &group.Spec.Template
	instanceTmpl := templateFromTiCDC(instance)

	return equalTiCDCTemplate(groupTmpl, instanceTmpl)
}

// CheckTiCDCPod returns whether changes are reloadable. No change also means reloadable.
// Pod records the last template in annotation.
// We use a same way to check whether changes are reloadable.
// This function is used in the instance controller.
func CheckTiCDCPod(instance *v1alpha1.TiCDC, pod *corev1.Pod) bool {
	lastInstanceTemplate := pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate]

	tmpl := &v1alpha1.TiCDCTemplate{}
	if err := json.Unmarshal([]byte(lastInstanceTemplate), tmpl); err != nil {
		// last template is not found or unexpectedly changed
		return true
	}

	instanceTmpl := templateFromTiCDC(instance)

	return equalTiCDCTemplate(instanceTmpl, tmpl)
}

func EncodeLastTiCDCTemplate(instance *v1alpha1.TiCDC, pod *corev1.Pod) error {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	instanceTmpl := templateFromTiCDC(instance)

	data, err := json.Marshal(instanceTmpl)
	if err != nil {
		return err
	}
	pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate] = string(data)

	return nil
}

func MustEncodeLastTiCDCTemplate(instance *v1alpha1.TiCDC, pod *corev1.Pod) {
	if err := EncodeLastTiCDCTemplate(instance, pod); err != nil {
		panic("cannot encode ticdc template: " + err.Error())
	}
}

// TODO: ignore inherited labels and annotations
func templateFromTiCDC(cdc *v1alpha1.TiCDC) *v1alpha1.TiCDCTemplate {
	return &v1alpha1.TiCDCTemplate{
		ObjectMeta: v1alpha1.ObjectMeta{
			Labels:      cdc.Labels,
			Annotations: cdc.Annotations,
		},
		Spec: cdc.Spec.TiCDCTemplateSpec,
	}
}

// convertTiCDCTemplate will ignore some fields
// TODO: set default for some empty fields
func convertTiCDCTemplate(tmpl *v1alpha1.TiCDCTemplate) *v1alpha1.TiCDCTemplate {
	newTmpl := tmpl.DeepCopy()

	newTmpl.Labels = convertLabels(newTmpl.Labels)
	newTmpl.Annotations = convertAnnotations(newTmpl.Annotations)

	if newTmpl.Spec.PreStop != nil {
		newTmpl.Spec.PreStop.Image = nil
	}

	newTmpl.Spec.Volumes = convertVolumes(newTmpl.Spec.Volumes)
	newTmpl.Spec.Overlay = convertOverlay(newTmpl.Spec.Overlay)

	return newTmpl
}

func equalTiCDCTemplate(p, c *v1alpha1.TiCDCTemplate) bool {
	p = convertTiCDCTemplate(p)
	c = convertTiCDCTemplate(c)
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
