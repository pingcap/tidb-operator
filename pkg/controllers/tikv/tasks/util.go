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
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
)

func ConfigMapName(podName string) string {
	return podName
}

func PersistentVolumeClaimName(podName, volName string) string {
	// ref: https://github.com/pingcap/tidb-operator/blob/v1.6.0/pkg/apis/pingcap/v1alpha1/helpers.go#L92
	// NOTE: for v1, should use component as volName of data, e.g. tikv
	return fmt.Sprintf("%s-%s", volName, podName)
}

func DeletePodWithGracePeriod(ctx context.Context, c client.Client, pod *corev1.Pod, regionCount int) error {
	if pod == nil {
		return nil
	}
	sec := pod.GetDeletionGracePeriodSeconds()
	gracePeriod := CalcGracePeriod(regionCount)

	if sec == nil || *sec > gracePeriod {
		if err := c.Delete(ctx, pod, client.GracePeriodSeconds(gracePeriod)); err != nil {
			return err
		}
	}

	return nil
}

func CalcGracePeriod(regionCount int) int64 {
	gracePeriod := int64(regionCount/RegionsPerSecond + 1)
	if gracePeriod < MinGracePeriodSeconds {
		gracePeriod = MinGracePeriodSeconds
	}

	return gracePeriod
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
	if mount.Type == v1alpha1.VolumeMountTypeTiKVData {
		if vm.MountPath == "" {
			vm.MountPath = v1alpha1.VolumeMountTiKVDataDefaultPath
		}
	}

	return vm
}
