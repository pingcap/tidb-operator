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
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestCheckTiKVPlacementIsReloadable(t *testing.T) {
	group := &v1alpha1.TiKVGroup{
		Spec: v1alpha1.TiKVGroupSpec{
			Template: v1alpha1.TiKVTemplate{
				Spec: v1alpha1.TiKVTemplateSpec{
					Version: "v1.2.3",
					Placement: &v1alpha1.TiKVStorePlacement{
						Exclusive: ptr.To(true),
					},
				},
			},
		},
	}
	instance := &v1alpha1.TiKV{
		Spec: v1alpha1.TiKVSpec{
			TiKVTemplateSpec: v1alpha1.TiKVTemplateSpec{
				Version: "v1.2.3",
			},
		},
	}

	assert.True(t, CheckTiKV(group, instance))
}

func TestCheckTiKVPodPlacementIsReloadable(t *testing.T) {
	pod := &corev1.Pod{}
	lastInstance := &v1alpha1.TiKV{
		Spec: v1alpha1.TiKVSpec{
			TiKVTemplateSpec: v1alpha1.TiKVTemplateSpec{
				Version: "v1.2.3",
			},
		},
	}
	require.NoError(t, EncodeLastTiKVTemplate(lastInstance, pod))

	currentInstance := lastInstance.DeepCopy()
	currentInstance.Spec.Placement = &v1alpha1.TiKVStorePlacement{
		Exclusive: ptr.To(true),
	}

	assert.True(t, CheckTiKVPod(currentInstance, pod))
}

func TestCheckTiKVPodVersionChangeIsNotReloadable(t *testing.T) {
	pod := &corev1.Pod{}
	lastInstance := &v1alpha1.TiKV{
		Spec: v1alpha1.TiKVSpec{
			TiKVTemplateSpec: v1alpha1.TiKVTemplateSpec{
				Version: "v1.2.3",
			},
		},
	}
	require.NoError(t, EncodeLastTiKVTemplate(lastInstance, pod))

	currentInstance := lastInstance.DeepCopy()
	currentInstance.Spec.Version = "v1.3.3"

	assert.False(t, CheckTiKVPod(currentInstance, pod))
}
