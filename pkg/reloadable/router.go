// Copyright 2026 PingCAP, Inc.
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

// CheckRouter returns whether changes are reloadable. No change also means reloadable.
// This function is used in the group controller.
func CheckRouter(group *v1alpha1.RouterGroup, instance *v1alpha1.Router) bool {
	groupTmpl := &group.Spec.Template
	instanceTmpl := templateFromRouter(instance)

	return equalRouterTemplate(groupTmpl, instanceTmpl) &&
		features.Reloadable(meta.ComponentRouter, group.Spec.Features, instance.Spec.Features)
}

// CheckRouterPod returns whether changes are reloadable. No change also means reloadable.
// This function is used in the instance controller.
func CheckRouterPod(instance *v1alpha1.Router, pod *corev1.Pod) bool {
	lastInstanceTemplate := pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate]
	lastFeatures := pod.Annotations[v1alpha1.AnnoKeyFeatures]

	tmpl := &v1alpha1.RouterTemplate{}
	if err := json.Unmarshal([]byte(lastInstanceTemplate), tmpl); err != nil {
		// last template is not found or unexpectedly changed
		return true
	}

	instanceTmpl := templateFromRouter(instance)

	return equalRouterTemplate(instanceTmpl, tmpl) &&
		features.Reloadable(meta.ComponentRouter, instance.Spec.Features, decodeFeatures(lastFeatures))
}

func EncodeLastRouterTemplate(instance *v1alpha1.Router, pod *corev1.Pod) error {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	instanceTmpl := templateFromRouter(instance)

	data, err := json.Marshal(instanceTmpl)
	if err != nil {
		return err
	}
	pod.Annotations[v1alpha1.AnnoKeyLastInstanceTemplate] = string(data)
	pod.Annotations[v1alpha1.AnnoKeyFeatures] = encodeFeatures(instance.Spec.Features)

	return nil
}

func MustEncodeLastRouterTemplate(instance *v1alpha1.Router, pod *corev1.Pod) {
	if err := EncodeLastRouterTemplate(instance, pod); err != nil {
		panic("cannot encode router template: " + err.Error())
	}
}

// TODO: ignore inherited labels and annotations
func templateFromRouter(r *v1alpha1.Router) *v1alpha1.RouterTemplate {
	return &v1alpha1.RouterTemplate{
		ObjectMeta: v1alpha1.ObjectMeta{
			Labels:      r.Labels,
			Annotations: r.Annotations,
		},
		Spec: r.Spec.RouterTemplateSpec,
	}
}

// convertRouterTemplate will ignore some fields
// TODO: set default for some empty fields
func convertRouterTemplate(tmpl *v1alpha1.RouterTemplate) *v1alpha1.RouterTemplate {
	newTmpl := tmpl.DeepCopy()

	newTmpl.Labels = convertLabels(newTmpl.Labels)
	newTmpl.Annotations = convertAnnotations(newTmpl.Annotations)
	newTmpl.Spec.Volumes = convertVolumes(newTmpl.Spec.Volumes)
	newTmpl.Spec.Overlay = convertOverlay(newTmpl.Spec.Overlay)

	return newTmpl
}

func equalRouterTemplate(c, p *v1alpha1.RouterTemplate) bool {
	p = convertRouterTemplate(p)
	c = convertRouterTemplate(c)
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
