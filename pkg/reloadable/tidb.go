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

// CheckTiDB returns whether changes are reloadable. No change also means reloadable.
// We assume that only when the instance template is changed, the pod spec will change.
// Instance copies the template spec from group directly,
// so we can use the template of instance directly as the last template.
// This function is used in the group controller.
func CheckTiDB(group *v1alpha1.TiDBGroup, instance *v1alpha1.TiDB) bool {
	groupTmpl := &group.Spec.Template
	instanceTmpl := templateFromTiDB(instance)

	return equalTiDBTemplate(groupTmpl, instanceTmpl) &&
		features.Reloadable(meta.ComponentTiDB, group.Spec.Features, instance.Spec.Features)
}

// CheckTiDBPod returns whether changes are reloadable. No change also means reloadable.
// Pod records the last template in annotation.
// We use a same way to check whether changes are reloadable.
// This function is used in the instance controller.
func CheckTiDBPod(instance *v1alpha1.TiDB, pod *corev1.Pod) bool {
	lastInstanceTemplate := pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate]
	lastFeatures := pod.Annotations[v1alpha1.AnnoKeyFeatures]

	tmpl := &v1alpha1.TiDBTemplate{}
	if err := json.Unmarshal([]byte(lastInstanceTemplate), tmpl); err != nil {
		// last template is not found or unexpectedly changed
		return true
	}

	instanceTmpl := templateFromTiDB(instance)

	return equalTiDBTemplate(instanceTmpl, tmpl) &&
		features.Reloadable(meta.ComponentTiDB, instance.Spec.Features, decodeFeatures(lastFeatures))
}

func EncodeLastTiDBTemplate(instance *v1alpha1.TiDB, pod *corev1.Pod) error {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	instanceTmpl := templateFromTiDB(instance)

	data, err := json.Marshal(instanceTmpl)
	if err != nil {
		return err
	}
	pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate] = string(data)
	pod.Annotations[v1alpha1.AnnoKeyFeatures] = encodeFeatures(instance.Spec.Features)

	return nil
}

func MustEncodeLastTiDBTemplate(instance *v1alpha1.TiDB, pod *corev1.Pod) {
	if err := EncodeLastTiDBTemplate(instance, pod); err != nil {
		panic("cannot encode tidb template: " + err.Error())
	}
}

// TODO: ignore inherited labels and annotations
func templateFromTiDB(db *v1alpha1.TiDB) *v1alpha1.TiDBTemplate {
	return &v1alpha1.TiDBTemplate{
		ObjectMeta: v1alpha1.ObjectMeta{
			Labels:      db.Labels,
			Annotations: db.Annotations,
		},
		Spec: db.Spec.TiDBTemplateSpec,
	}
}

// convertTiDBTemplate will ignore some fields
// TODO: set default for some empty fields
func convertTiDBTemplate(tmpl *v1alpha1.TiDBTemplate) *v1alpha1.TiDBTemplate {
	newTmpl := tmpl.DeepCopy()

	newTmpl.Labels = convertLabels(newTmpl.Labels)
	newTmpl.Annotations = convertAnnotations(newTmpl.Annotations)

	if newTmpl.Spec.SlowLog != nil {
		newTmpl.Spec.SlowLog.Image = nil
	}

	if newTmpl.Spec.Mode != v1alpha1.TiDBModeStandBy {
		newTmpl.Spec.Mode = v1alpha1.TiDBModeNormal
	}

	// server labels can be updated dynamically
	newTmpl.Spec.Server.Labels = nil

	newTmpl.Spec.Volumes = convertVolumes(newTmpl.Spec.Volumes)
	newTmpl.Spec.Overlay = convertOverlay(newTmpl.Spec.Overlay)

	return newTmpl
}

func equalTiDBTemplate(p, c *v1alpha1.TiDBTemplate) bool {
	p = convertTiDBTemplate(p)
	c = convertTiDBTemplate(c)
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
