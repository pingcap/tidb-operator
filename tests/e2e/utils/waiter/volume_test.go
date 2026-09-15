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
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

func TestAssertPVCListVolumesCapacity(t *testing.T) {
	instance := &v1alpha1.TiKV{ObjectMeta: metav1.ObjectMeta{Name: "tikv-0", Namespace: "test"}}
	volumes := []v1alpha1.Volume{{Name: "data", Storage: resource.MustParse("1Gi")}}
	cases := []struct {
		name          string
		capacity      string
		expectExceeds bool
		wantError     bool
	}{
		{name: "exceeds rejects undersized", capacity: "500Mi", expectExceeds: true, wantError: true},
		{name: "exceeds rejects equal", capacity: "1Gi", expectExceeds: true, wantError: true},
		{name: "exceeds accepts larger", capacity: "2Gi", expectExceeds: true},
		{name: "matching rejects undersized", capacity: "500Mi", wantError: true},
		{name: "matching accepts equal", capacity: "1Gi"},
		{name: "matching rejects larger", capacity: "2Gi", wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capacity := resource.MustParse(tc.capacity)
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Namespace: instance.Namespace, Name: coreutil.PersistentVolumeClaimName[scope.TiKV](instance, "data")},
				Spec: corev1.PersistentVolumeClaimSpec{Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceStorage: capacity},
				}},
				Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound, Capacity: corev1.ResourceList{corev1.ResourceStorage: capacity}},
			}
			err := AssertPVCListVolumes[scope.TiKV]([]*v1alpha1.TiKV{instance}, volumes, tc.expectExceeds)([]*corev1.PersistentVolumeClaim{pvc})
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
