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

// CheckDMWorkerPod returns whether changes are reloadable without pod recreation.
// DM worker has no hot-reload capability, so any config change requires restart.
// No change also means reloadable.
func CheckDMWorkerPod(instance *v1alpha1.DMWorker, pod *corev1.Pod) bool {
	lastInstanceTemplate := pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate]

	tmpl := &v1alpha1.DMWorkerTemplate{}
	if err := json.Unmarshal([]byte(lastInstanceTemplate), tmpl); err != nil {
		// last template not found or corrupted — treat as reloadable to avoid spurious restarts
		return true
	}

	instanceTmpl := templateFromDMWorker(instance)
	return equalDMWorkerTemplate(instanceTmpl, tmpl)
}

func EncodeLastDMWorkerTemplate(instance *v1alpha1.DMWorker, pod *corev1.Pod) error {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	instanceTmpl := templateFromDMWorker(instance)
	data, err := json.Marshal(instanceTmpl)
	if err != nil {
		return err
	}
	pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate] = string(data)

	return nil
}

func MustEncodeLastDMWorkerTemplate(instance *v1alpha1.DMWorker, pod *corev1.Pod) {
	if err := EncodeLastDMWorkerTemplate(instance, pod); err != nil {
		panic("cannot encode dm worker template: " + err.Error())
	}
}

func templateFromDMWorker(dw *v1alpha1.DMWorker) *v1alpha1.DMWorkerTemplate {
	return &v1alpha1.DMWorkerTemplate{
		ObjectMeta: v1alpha1.ObjectMeta{
			Labels:      dw.Labels,
			Annotations: dw.Annotations,
		},
		Spec: dw.Spec.DMWorkerTemplateSpec,
	}
}

func convertDMWorkerTemplate(tmpl *v1alpha1.DMWorkerTemplate) *v1alpha1.DMWorkerTemplate {
	newTmpl := tmpl.DeepCopy()

	newTmpl.Labels = convertLabels(newTmpl.Labels)
	newTmpl.Annotations = convertAnnotations(newTmpl.Annotations)
	newTmpl.Spec.Volumes = convertVolumes(newTmpl.Spec.Volumes)
	newTmpl.Spec.Overlay = convertOverlay(newTmpl.Spec.Overlay)

	return newTmpl
}

// equalDMWorkerTemplate checks whether two DM worker templates are equivalent for pod scheduling.
// DM worker has no hot-reload: any config change always requires pod recreation.
func equalDMWorkerTemplate(c, p *v1alpha1.DMWorkerTemplate) bool {
	p = convertDMWorkerTemplate(p)
	c = convertDMWorkerTemplate(c)

	if p.Spec.Config != c.Spec.Config {
		return false
	}

	p.Spec.Config = ""
	c.Spec.Config = ""
	p.Spec.UpdateStrategy.Config = ""
	c.Spec.UpdateStrategy.Config = ""

	return apiequality.Semantic.DeepEqual(p, c)
}
