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

// CheckReplicationWorkerPod returns whether changes are reloadable. No change also means reloadable.
// Pod records the last template in annotation.
// We use a same way to check whether changes are reloadable.
// This function is used in the instance controller.
func CheckReplicationWorkerPod(instance *v1alpha1.ReplicationWorker, pod *corev1.Pod) bool {
	lastInstanceTemplate := pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate]
	lastFeatures := pod.Annotations[v1alpha1.AnnoKeyFeatures]

	tmpl := &v1alpha1.ReplicationWorkerTemplate{}
	if err := json.Unmarshal([]byte(lastInstanceTemplate), tmpl); err != nil {
		// last template is not found or unexpectedly changed
		return true
	}

	instanceTmpl := templateFromReplicationWorker(instance)

	return equalReplicationWorkerTemplate(instanceTmpl, tmpl) &&
		features.Reloadable(meta.ComponentReplicationWorker, instance.Spec.Features, decodeFeatures(lastFeatures))
}

func EncodeLastReplicationWorkerTemplate(instance *v1alpha1.ReplicationWorker, pod *corev1.Pod) error {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	instanceTmpl := templateFromReplicationWorker(instance)

	data, err := json.Marshal(instanceTmpl)
	if err != nil {
		return err
	}
	pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate] = string(data)
	pod.Annotations[v1alpha1.AnnoKeyFeatures] = encodeFeatures(instance.Spec.Features)

	return nil
}

func MustEncodeLastReplicationWorkerTemplate(instance *v1alpha1.ReplicationWorker, pod *corev1.Pod) {
	if err := EncodeLastReplicationWorkerTemplate(instance, pod); err != nil {
		panic("cannot encode scheduler template: " + err.Error())
	}
}

// TODO: ignore inherited labels and annotations
func templateFromReplicationWorker(scheduler *v1alpha1.ReplicationWorker) *v1alpha1.ReplicationWorkerTemplate {
	return &v1alpha1.ReplicationWorkerTemplate{
		ObjectMeta: v1alpha1.ObjectMeta{
			Labels:      scheduler.Labels,
			Annotations: scheduler.Annotations,
		},
		Spec: scheduler.Spec.ReplicationWorkerTemplateSpec,
	}
}

// convertReplicationWorkerTemplate will ignore some fields
// TODO: set default for some empty fields
func convertReplicationWorkerTemplate(tmpl *v1alpha1.ReplicationWorkerTemplate) *v1alpha1.ReplicationWorkerTemplate {
	newTmpl := tmpl.DeepCopy()

	newTmpl.Labels = convertLabels(newTmpl.Labels)
	newTmpl.Annotations = convertAnnotations(newTmpl.Annotations)
	newTmpl.Spec.Volumes = convertVolumes(newTmpl.Spec.Volumes)
	newTmpl.Spec.Overlay = convertOverlay(newTmpl.Spec.Overlay)

	return newTmpl
}

func equalReplicationWorkerTemplate(p, c *v1alpha1.ReplicationWorkerTemplate) bool {
	p = convertReplicationWorkerTemplate(p)
	c = convertReplicationWorkerTemplate(c)
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
