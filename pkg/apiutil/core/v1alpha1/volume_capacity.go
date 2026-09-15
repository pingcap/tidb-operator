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

package coreutil

import (
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

// VolumeCapacityCondition compares declared volume requests with allocated PVC capacity.
// pvcs is keyed by volume name. Missing entries have unknown capacity.
func VolumeCapacityCondition(volumes []v1alpha1.Volume, pvcs map[string]*corev1.PersistentVolumeClaim) metav1.Condition {
	cond := metav1.Condition{
		Type:    v1alpha1.CondVolumeCapacityExceedsRequest,
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.ReasonCapacityDoesNotExceedRequest,
		Message: "No volume capacity exceeds its request.",
	}
	if len(volumes) == 0 {
		cond.Reason = v1alpha1.ReasonNoVolumes
		cond.Message = "The instance declares no volumes."
		return cond
	}
	var excess, unknown []string
	for _, vol := range volumes {
		pvc := pvcs[vol.Name]
		if pvc == nil {
			unknown = append(unknown, fmt.Sprintf("Volume %s: PVC does not exist.", vol.Name))
			continue
		}
		capacity, ok := pvc.Status.Capacity[corev1.ResourceStorage]
		if !ok {
			unknown = append(unknown, fmt.Sprintf("Volume %s (PVC %s): capacity is not reported.", vol.Name, pvc.Name))
		} else if capacity.Cmp(vol.Storage) > 0 {
			excess = append(excess, fmt.Sprintf("Volume %s (PVC %s): capacity %s exceeds request %s.",
				vol.Name, pvc.Name, capacity.String(), vol.Storage.String()))
		}
	}
	if len(excess) > 0 {
		slices.Sort(excess)
		cond.Status = metav1.ConditionTrue
		cond.Reason = v1alpha1.ReasonCapacityExceedsRequest
		cond.Message = strings.Join(excess, " ")
	} else if len(unknown) > 0 {
		slices.Sort(unknown)
		cond.Status = metav1.ConditionUnknown
		cond.Reason = v1alpha1.ReasonCapacityUnknown
		cond.Message = strings.Join(unknown, " ")
	}
	return cond
}
