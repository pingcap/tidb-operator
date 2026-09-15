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

package waiter

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

// WaitForPVCList waits for the group's PVC list to satisfy cond. The list may
// include PVCs retained after scale-in, so the callback determines the expected count.
func WaitForPVCList[S scope.Group[F, T], F client.Object, T runtime.Group](
	ctx context.Context, c client.Client, g F, cond func([]*corev1.PersistentVolumeClaim) error, timeout time.Duration,
) error {
	group := scope.From[S](g)
	list := &corev1.PersistentVolumeClaimList{}
	return WaitForList(ctx, c, list, func() error {
		items := make([]*corev1.PersistentVolumeClaim, len(list.Items))
		for i := range list.Items {
			items[i] = &list.Items[i]
		}
		return cond(items)
	}, timeout, client.InNamespace(g.GetNamespace()), client.MatchingLabels{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyCluster:   group.Cluster(),
		v1alpha1.LabelKeyGroup:     g.GetName(),
		v1alpha1.LabelKeyComponent: group.Component(),
	})
}

// WaitForVolumeCapacityExceedsRequest first checks instance requests and conditions, then their PVCs.
// exceeds is the expected number of instances with VolumeCapacityExceedsRequest=True.
// Retained PVCs without an active instance are ignored.
func WaitForVolumeCapacityExceedsRequest[
	GS scope.GroupInstance[GF, GT, IS],
	IS scope.InstanceList[IF, IT, IL],
	GF client.Object,
	GT runtime.Group,
	IF client.Object,
	IT runtime.Instance,
	IL client.ObjectList,
](ctx context.Context, c client.Client, group GF, exceeds int, timeout time.Duration) error {
	volumes := coreutil.Volumes[GS](group)
	var old, current []IF
	instanceErr := WaitForInstanceList[GS](ctx, c, group, func(items []IF) error {
		old, current = nil, nil
		for _, instance := range items {
			cond := coreutil.FindStatusCondition[IS](instance, v1alpha1.CondVolumeCapacityExceedsRequest)
			if cond != nil && cond.Status == metav1.ConditionTrue {
				old = append(old, instance)
			} else {
				current = append(current, instance)
			}
		}
		errList := []error{
			AssertInstanceListCondition[IS](v1alpha1.CondVolumeCapacityExceedsRequest, metav1.ConditionTrue)(old),
			AssertInstanceListCondition[IS](v1alpha1.CondVolumeCapacityExceedsRequest, metav1.ConditionFalse)(current),
			AssertInstanceListVolumes[IS](volumes)(items),
		}
		if len(old) != exceeds {
			errList = append(errList, fmt.Errorf("expected %d instances with VolumeCapacityExceedsRequest, found %d", exceeds, len(old)))
		}
		return errors.NewAggregate(errList)
	}, timeout)

	pvcErr := WaitForPVCList[GS](ctx, c, group, func(items []*corev1.PersistentVolumeClaim) error {
		return errors.NewAggregate([]error{
			AssertPVCListVolumes[IS](old, volumes, true)(items),
			AssertPVCListVolumes[IS](current, volumes, false)(items),
		})
	}, timeout)
	return errors.NewAggregate([]error{instanceErr, pvcErr})
}

// AssertInstanceListVolumes checks that active instances request the expected volume storage.
func AssertInstanceListVolumes[S scope.Instance[F, T], F client.Object, T runtime.Instance](
	volumes []v1alpha1.Volume,
) func([]F) error {
	return func(items []F) error {
		var errList []error
		for _, instance := range items {
			if !instance.GetDeletionTimestamp().IsZero() || coreutil.IsOffline[S](instance) {
				errList = append(errList, fmt.Errorf("instance %s is deleting or offline", instance.GetName()))
			}
			for _, expected := range volumes {
				found := false
				for _, volume := range coreutil.Volumes[S](instance) {
					if volume.Name != expected.Name {
						continue
					}
					found = true
					if volume.Storage.Cmp(expected.Storage) != 0 {
						errList = append(errList, fmt.Errorf("instance %s volume %s requests %s, expected %s", instance.GetName(), volume.Name, volume.Storage.String(), expected.Storage.String()))
					}
				}
				if !found {
					errList = append(errList, fmt.Errorf("instance %s has no volume %s", instance.GetName(), expected.Name))
				}
			}
		}
		return errors.NewAggregate(errList)
	}
}

// AssertPVCListVolumes checks PVCs for the given instances. When expectExceeds
// is true, each instance must have at least one PVC whose storage exceeds
// the group request; otherwise all PVCs must match. Every PVC must have at least
// the requested capacity. Unrelated PVCs are ignored.
func AssertPVCListVolumes[S scope.Instance[F, T], F client.Object, T runtime.Instance](
	instances []F, volumes []v1alpha1.Volume, expectExceeds bool,
) func([]*corev1.PersistentVolumeClaim) error {
	return func(items []*corev1.PersistentVolumeClaim) error {
		var errList []error
		pvcs := make(map[client.ObjectKey]*corev1.PersistentVolumeClaim, len(items))
		for _, pvc := range items {
			pvcs[client.ObjectKeyFromObject(pvc)] = pvc
		}
		for _, instance := range instances {
			exceeds, complete := false, true
			for _, expected := range volumes {
				key := client.ObjectKey{Namespace: instance.GetNamespace(), Name: coreutil.PersistentVolumeClaimName[S](instance, expected.Name)}
				pvc, ok := pvcs[key]
				if !ok {
					errList = append(errList, fmt.Errorf("PVC %s not found", key))
					complete = false
					continue
				}
				if pvc.Status.Phase != corev1.ClaimBound {
					errList = append(errList, fmt.Errorf("PVC %s is %s, expected Bound", key, pvc.Status.Phase))
				}
				capacity := pvc.Status.Capacity.Storage()
				if request := pvc.Spec.Resources.Requests.Storage(); request.Cmp(*capacity) != 0 {
					errList = append(errList, fmt.Errorf("PVC %s requests %s, capacity is %s", key, request.String(), capacity.String()))
				}
				comparison := capacity.Cmp(expected.Storage)
				if comparison > 0 {
					exceeds = true
				}
				if comparison < 0 || (!expectExceeds && comparison > 0) {
					errList = append(errList, fmt.Errorf("PVC %s capacity is %s, expected %s", key, capacity.String(), expected.Storage.String()))
				}
			}
			if expectExceeds && complete && !exceeds {
				errList = append(errList, fmt.Errorf("instance %s has VolumeCapacityExceedsRequest but no PVC storage exceeds the group request", instance.GetName()))
			}
		}
		return errors.NewAggregate(errList)
	}
}
