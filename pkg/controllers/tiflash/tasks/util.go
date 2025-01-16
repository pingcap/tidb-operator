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
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

func ConfigMapName(podName string) string {
	return podName
}

func PersistentVolumeClaimName(podName, volName string) string {
	// ref: https://github.com/pingcap/tidb-operator/blob/486cc85c8380efc4f36b3125a1abba9e3146a2c8/pkg/apis/pingcap/v1alpha1/helpers.go#L105
	// NOTE: for v1, volName should be data0, data1, ...
	return fmt.Sprintf("%s-%s", volName, podName)
}

// Real spec.volumes[*].name of pod
// TODO(liubo02): extract to namer pkg
func VolumeName(volName string) string {
	return v1alpha1.VolNamePrefix + volName
}

func VolumeMount(name string, mount *v1alpha1.VolumeMount) *corev1.VolumeMount {
	vm := &corev1.VolumeMount{
		Name:      name,
		MountPath: mount.MountPath,
		SubPath:   mount.SubPath,
	}
	if mount.Type == v1alpha1.VolumeMountTypeTiFlashData {
		if vm.MountPath == "" {
			vm.MountPath = v1alpha1.VolumeMountTiFlashDataDefaultPath
		}
	}

	return vm
}
