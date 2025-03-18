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

package checker

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func NeedRestartInstance(group *v1alpha1.TiDBGroup, instance *v1alpha1.TiDB) bool {
	current := TiDBGroup(group)
	previous := TiDB(instance)

	return !equal(previous, current)
}

func EncodeTiDB(instance *v1alpha1.TiDB) ([]byte, error) {
	spec := TiDB(instance)
	return json.Marshal(spec)
}

func TiDBGroup(group *v1alpha1.TiDBGroup) *v1alpha1.TiDBTemplateSpec {
	return ignore(&group.Spec.Template.Spec)
}

func TiDB(instance *v1alpha1.TiDB) *v1alpha1.TiDBTemplateSpec {
	return ignore(&instance.Spec.TiDBTemplateSpec)
}

// ignore changes of reloadable fields
func ignore(tmpl *v1alpha1.TiDBTemplateSpec) *v1alpha1.TiDBTemplateSpec {
	newTmpl := tmpl.DeepCopy()

	if newTmpl.SlowLog != nil {
		newTmpl.SlowLog.Image = nil
	}

	// ignore overlay labels and annotations
	if newTmpl.Overlay != nil && newTmpl.Overlay.Pod != nil {
		newTmpl.Overlay.Pod.Labels = nil
		newTmpl.Overlay.Pod.Annotations = nil
		if newTmpl.Overlay.Pod.Spec != nil {
			for i := range newTmpl.Overlay.Pod.Spec.Containers {
				c := &newTmpl.Overlay.Pod.Spec.Containers[i]
				// ignore image overlay
				c.Image = ""
			}
		}
	}

	return newTmpl
}

func equal(p, c *v1alpha1.TiDBTemplateSpec) bool {
	// not equal only when current strategy is Restart and config is changed
	switch c.UpdateStrategy.Config {
	case v1alpha1.ConfigUpdateStrategyRestart:
		if p.Config != c.Config {
			return false
		}
	}

	if !equalOverlay(p.Overlay, c.Overlay) {
		return false
	}

	// save state
	pc := p.Config
	cc := c.Config
	psc := p.UpdateStrategy.Config
	csc := c.UpdateStrategy.Config
	po := p.Overlay
	co := c.Overlay

	// restore state
	defer func() {
		p.Config = pc
		c.Config = cc
		p.UpdateStrategy.Config = psc
		c.UpdateStrategy.Config = csc
		p.Overlay = po
		c.Overlay = co
	}()

	// ignore these fields
	p.Config = ""
	c.Config = ""
	p.UpdateStrategy.Config = ""
	c.UpdateStrategy.Config = ""
	p.Overlay = nil
	c.Overlay = nil

	return apiequality.Semantic.DeepEqual(p, c)
}

func equalOverlay(p, c *v1alpha1.Overlay) bool {
	p = covertNilToEmpty(p)
	c = covertNilToEmpty(c)

	return apiequality.Semantic.DeepEqual(p, c)
}

func covertNilToEmpty(o *v1alpha1.Overlay) *v1alpha1.Overlay {
	if o == nil {
		o = &v1alpha1.Overlay{}
	} else {
		o = o.DeepCopy()
	}
	if o.Pod == nil {
		o.Pod = &v1alpha1.PodOverlay{}
	}
	if o.Pod.Spec == nil {
		o.Pod.Spec = &corev1.PodSpec{}
	}
	return o
}
