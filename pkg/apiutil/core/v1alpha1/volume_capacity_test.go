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
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestVolumeCapacityCondition(t *testing.T) {
	vol := func(name, size string) v1alpha1.Volume {
		return v1alpha1.Volume{Name: name, Storage: resource.MustParse(size)}
	}
	pvc := func(size string) *corev1.PersistentVolumeClaim {
		p := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "pvc-data"}}
		if size != "" {
			p.Status.Capacity = corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(size)}
		}
		return p
	}
	for _, tc := range []struct {
		name   string
		vols   []v1alpha1.Volume
		pvcs   map[string]*corev1.PersistentVolumeClaim
		status metav1.ConditionStatus
		reason string
	}{
		{"no volumes", nil, nil, metav1.ConditionFalse, v1alpha1.ReasonNoVolumes},
		{"missing PVC", []v1alpha1.Volume{vol("data", "50Gi")}, nil, metav1.ConditionUnknown, v1alpha1.ReasonCapacityUnknown},
		{"missing capacity", []v1alpha1.Volume{vol("data", "50Gi")}, map[string]*corev1.PersistentVolumeClaim{"data": pvc("")}, metav1.ConditionUnknown, v1alpha1.ReasonCapacityUnknown},
		{"excess", []v1alpha1.Volume{vol("data", "50Gi")}, map[string]*corev1.PersistentVolumeClaim{"data": pvc("100Gi")}, metav1.ConditionTrue, v1alpha1.ReasonCapacityExceedsRequest},
		{"equal units", []v1alpha1.Volume{vol("data", "1Ti")}, map[string]*corev1.PersistentVolumeClaim{"data": pvc("1024Gi")}, metav1.ConditionFalse, v1alpha1.ReasonCapacityDoesNotExceedRequest},
		{"expanding", []v1alpha1.Volume{vol("data", "200Gi")}, map[string]*corev1.PersistentVolumeClaim{"data": pvc("100Gi")}, metav1.ConditionFalse, v1alpha1.ReasonCapacityDoesNotExceedRequest},
		{"excess and unknown", []v1alpha1.Volume{vol("missing", "50Gi"), vol("data", "50Gi")}, map[string]*corev1.PersistentVolumeClaim{"data": pvc("100Gi")}, metav1.ConditionTrue, v1alpha1.ReasonCapacityExceedsRequest},
		{"equal and unknown", []v1alpha1.Volume{vol("missing", "50Gi"), vol("data", "100Gi")}, map[string]*corev1.PersistentVolumeClaim{"data": pvc("100Gi")}, metav1.ConditionUnknown, v1alpha1.ReasonCapacityUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := VolumeCapacityCondition(tc.vols, tc.pvcs)
			require.Equal(t, v1alpha1.CondVolumeCapacityExceedsRequest, c.Type)
			require.Equal(t, tc.status, c.Status)
			require.Equal(t, tc.reason, c.Reason)
		})
	}
	pvcs := map[string]*corev1.PersistentVolumeClaim{"a": pvc("100Gi"), "b": pvc("100Gi")}
	a, b := vol("a", "50Gi"), vol("b", "25Gi")
	c := VolumeCapacityCondition([]v1alpha1.Volume{b, a}, pvcs)
	require.Equal(t, VolumeCapacityCondition([]v1alpha1.Volume{a, b}, pvcs), c)
	require.Contains(t, c.Message, "capacity 100Gi exceeds request 25Gi")
	require.Contains(t, c.Message, "capacity 100Gi exceeds request 50Gi")
}
