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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func ConfigMapName(podName string) string {
	return podName
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

// VolumeName returns the real spec.volumes[*].name of pod
// TODO(liubo02): extract to namer pkg
func VolumeName(volName string) string {
	return metav1alpha1.VolNamePrefix + volName
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

func CheckTiKVLeadersEvicted(tikv *v1alpha1.TiKV) error {
	cond := meta.FindStatusCondition(tikv.Status.Conditions, v1alpha1.TiKVCondLeadersEvicted)
	if cond == nil {
		return fmt.Errorf("tikv leaders are not evicting")
	}
	if cond.Status == metav1.ConditionTrue {
		return nil
	}

	return fmt.Errorf("tikv leaders are not evicted: %v, %v", cond.Reason, cond.Message)
}

func CheckTiKVLeadersEvictedOrTimeout(tikv *v1alpha1.TiKV, timeout time.Duration) error {
	cond := meta.FindStatusCondition(tikv.Status.Conditions, v1alpha1.TiKVCondLeadersEvicted)
	if cond == nil {
		return fmt.Errorf("tikv leaders are not evicting")
	}
	// all leaders are evicted
	if cond.Status == metav1.ConditionTrue {
		return nil
	}

	// TODO: this is not a stable way, we should not use reason directly
	if cond.Reason != v1alpha1.ReasonEvicting {
		return fmt.Errorf("tikv leaders are not evicted: %v", cond.Reason)
	}

	t := cond.LastTransitionTime.Time
	if t.Add(timeout).Before(time.Now()) {
		return nil
	}

	return fmt.Errorf("tikv leaders are evicting: %v, %v", cond.Reason, cond.Message)
}
