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

package volumes

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

// retainPVCRequestIfSufficient retains the current PVC request when either that request
// or the allocated capacity already satisfies the desired size.
func retainPVCRequestIfSufficient() client.ApplyOption {
	return client.Transformers(client.TransformerFunc(func(current, expected client.Object) client.Object {
		if current == nil {
			return expected
		}
		desiredPVC := expected.(*corev1.PersistentVolumeClaim)
		currentPVC := current.(*corev1.PersistentVolumeClaim)
		size := desiredPVC.Spec.Resources.Requests[corev1.ResourceStorage]
		requested := currentPVC.Spec.Resources.Requests[corev1.ResourceStorage]
		capacity := currentPVC.Status.Capacity[corev1.ResourceStorage]
		// Keep the existing request if it or the allocated capacity already covers the
		// desired size. Raising request to capacity can trigger forbidden expansion
		// on a storage class without expansion support, even when no disk resize is needed.
		if size.Cmp(requested) <= 0 || size.Cmp(capacity) <= 0 {
			size = requested
		}
		if desiredPVC.Spec.Resources.Requests == nil {
			desiredPVC.Spec.Resources.Requests = corev1.ResourceList{}
		}
		desiredPVC.Spec.Resources.Requests[corev1.ResourceStorage] = size
		return desiredPVC
	}))
}
