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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestCheckTiKVPodWithUserLabelsAndAnnotations(t *testing.T) {
	tikv := &v1alpha1.TiKV{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tikv",
			Namespace: "default",
			Labels: map[string]string{
				"user-label": "value",
			},
			Annotations: map[string]string{
				"user-annotation": "value",
			},
		},
		Spec: v1alpha1.TiKVSpec{
			TiKVTemplateSpec: v1alpha1.TiKVTemplateSpec{},
		},
	}

	// Create a Pod with the same template but with additional user labels/annotations
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tikv-0",
			Namespace: "default",
			Labels: map[string]string{
				"user-label":   "value",        // Same as TiKV
				"manual-label": "manual-value", // Manually added
			},
			Annotations: map[string]string{
				"user-annotation":        "value",        // Same as TiKV
				"manual-annotation":      "manual-value", // Manually added
				"last-instance-template": `{"metadata":{"labels":{"user-label":"value"},"annotations":{"user-annotation":"value"}},"spec":{}}`,
				"features":               "",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "tikv",
					Image: "pingcap/tikv:v6.5.0",
				},
			},
		},
	}

	// This should return true (no restart needed) even with manually added labels/annotations
	result := CheckTiKVPod(tikv, pod)
	assert.True(t, result, "Pod should not be restarted when only user manually added labels/annotations changed")
}
