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

package tasks

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestVolumeMount(t *testing.T) {
	t.Run("uses default path for data type", func(t *testing.T) {
		mount := VolumeMount("data", &v1alpha1.VolumeMount{
			Type: v1alpha1.VolumeMountTypeTiKVWorkerData,
		})

		assert.Equal(t, "data", mount.Name)
		assert.Equal(t, v1alpha1.VolumeMountTiKVWorkerDataDefaultPath, mount.MountPath)
	})

	t.Run("keeps explicit mount path", func(t *testing.T) {
		mount := VolumeMount("data", &v1alpha1.VolumeMount{
			Type:      v1alpha1.VolumeMountTypeTiKVWorkerData,
			MountPath: "/custom",
		})

		assert.Equal(t, "/custom", mount.MountPath)
	})

	t.Run("does not use data default path when type is empty", func(t *testing.T) {
		mount := VolumeMount("data", &v1alpha1.VolumeMount{})

		assert.Equal(t, "data", mount.Name)
		assert.Empty(t, mount.MountPath)
	})
}
