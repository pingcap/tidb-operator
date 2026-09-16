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
	"k8s.io/apimachinery/pkg/api/resource"
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

func TestCheckTiKVCacheTTLSecondsIsReloadable(t *testing.T) {
	group := &v1alpha1.TiKVGroup{
		Spec: v1alpha1.TiKVGroupSpec{
			Template: v1alpha1.TiKVTemplate{
				Spec: v1alpha1.TiKVTemplateSpec{
					Version:         "v1.2.3",
					CacheTTLSeconds: ptr.To[int64](600),
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

func TestCheckTiKVPodCacheTTLSecondsIsReloadable(t *testing.T) {
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
	currentInstance.Spec.CacheTTLSeconds = ptr.To[int64](600)

	assert.True(t, CheckTiKVPod(currentInstance, pod))
}

func TestCheckTiKVMinReadyForLeaderSecondsIsReloadable(t *testing.T) {
	group := &v1alpha1.TiKVGroup{
		Spec: v1alpha1.TiKVGroupSpec{
			Template: v1alpha1.TiKVTemplate{
				Spec: v1alpha1.TiKVTemplateSpec{
					Version:                  "v1.2.3",
					MinReadyForLeaderSeconds: ptr.To[int64](60),
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

func TestCheckTiKVPodMinReadyForLeaderSecondsIsReloadable(t *testing.T) {
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
	currentInstance.Spec.MinReadyForLeaderSeconds = ptr.To[int64](60)

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

func TestVolumeRequestDecreaseIsReloadable(t *testing.T) {
	instance := &v1alpha1.TiKV{}
	instance.Spec.Volumes = []v1alpha1.Volume{{Name: "data", Storage: resource.MustParse("100Gi")}}
	pod := &corev1.Pod{}
	require.NoError(t, EncodeLastTiKVTemplate(instance, pod))
	group := &v1alpha1.TiKVGroup{}
	group.Spec.Template.Spec = *instance.Spec.TiKVTemplateSpec.DeepCopy()
	group.Spec.Template.Spec.Volumes[0].Storage = resource.MustParse("50Gi")
	require.True(t, CheckTiKV(group, instance))
	instance.Spec.TiKVTemplateSpec = *group.Spec.Template.Spec.DeepCopy()
	require.True(t, CheckTiKVPod(instance, pod))
	require.Zero(t, instance.Spec.Volumes[0].Storage.Cmp(resource.MustParse("50Gi")))
}
