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

package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestConvertToUnstructured(t *testing.T) {
	cases := []struct {
		desc     string
		gvk      schema.GroupVersionKind
		obj      client.Object
		expected map[string]any
	}{
		{
			desc: "no creationTimestamp",
			obj: &v1alpha1.TiKV{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			expected: map[string]any{
				"apiVersion": "",
				"kind":       "",
				"metadata": map[string]any{
					"name": "test",
				},
			},
		},
		{
			desc: "no initContainers",
			obj: &v1alpha1.TiKV{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1alpha1.TiKVSpec{
					TiKVTemplateSpec: v1alpha1.TiKVTemplateSpec{
						Overlay: &v1alpha1.Overlay{
							Pod: &v1alpha1.PodOverlay{
								Spec: &corev1.PodSpec{},
							},
						},
					},
				},
			},
			expected: map[string]any{
				"apiVersion": "",
				"kind":       "",
				"metadata": map[string]any{
					"name": "test",
				},
				"spec": map[string]any{
					"overlay": map[string]any{
						"pod": map[string]any{
							"spec": map[string]any{},
						},
					},
				},
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			u, err := convertToUnstructured(c.gvk, c.obj)
			require.NoError(tt, err)
			assert.Equal(tt, c.expected, u.Object)
		})
	}
}
