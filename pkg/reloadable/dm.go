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

// CheckDMPod returns whether changes are reloadable without pod recreation.
// DM (dm-master) has no hot-reload capability, so any config change requires restart.
// No change also means reloadable.
func CheckDMPod(instance *v1alpha1.DM, pod *corev1.Pod) bool {
	lastInstanceTemplate := pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate]

	tmpl := &v1alpha1.DMTemplate{}
	if err := json.Unmarshal([]byte(lastInstanceTemplate), tmpl); err != nil {
		// last template not found or corrupted — treat as reloadable to avoid spurious restarts
		return true
	}

	instanceTmpl := templateFromDM(instance)
	return equalDMTemplate(instanceTmpl, tmpl)
}

func EncodeLastDMTemplate(instance *v1alpha1.DM, pod *corev1.Pod) error {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	instanceTmpl := templateFromDM(instance)
	data, err := json.Marshal(instanceTmpl)
	if err != nil {
		return err
	}
	pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate] = string(data)

	return nil
}

func MustEncodeLastDMTemplate(instance *v1alpha1.DM, pod *corev1.Pod) {
	if err := EncodeLastDMTemplate(instance, pod); err != nil {
		panic("cannot encode dm template: " + err.Error())
	}
}

func templateFromDM(dm *v1alpha1.DM) *v1alpha1.DMTemplate {
	return &v1alpha1.DMTemplate{
		ObjectMeta: v1alpha1.ObjectMeta{
			Labels:      dm.Labels,
			Annotations: dm.Annotations,
		},
		Spec: dm.Spec.DMTemplateSpec,
	}
}

func convertDMTemplate(tmpl *v1alpha1.DMTemplate) *v1alpha1.DMTemplate {
	newTmpl := tmpl.DeepCopy()

	newTmpl.Labels = convertLabels(newTmpl.Labels)
	newTmpl.Annotations = convertAnnotations(newTmpl.Annotations)
	newTmpl.Spec.Volumes = convertVolumes(newTmpl.Spec.Volumes)
	newTmpl.Spec.Overlay = convertOverlay(newTmpl.Spec.Overlay)

	return newTmpl
}

// equalDMTemplate checks whether two DM templates are equivalent for pod scheduling.
// DM has no hot-reload: any config change always requires pod recreation.
func equalDMTemplate(c, p *v1alpha1.DMTemplate) bool {
	p = convertDMTemplate(p)
	c = convertDMTemplate(c)

	if p.Spec.Config != c.Spec.Config {
		return false
	}

	p.Spec.Config = ""
	c.Spec.Config = ""
	p.Spec.UpdateStrategy.Config = ""
	c.Spec.UpdateStrategy.Config = ""

	return apiequality.Semantic.DeepEqual(p, c)
}
