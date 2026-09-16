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
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func capacityPVC(request, capacity string) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "default"},
		Spec: corev1.PersistentVolumeClaimSpec{Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(request)}}}}
	if capacity != "" {
		pvc.Status.Capacity = corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(capacity)}
		pvc.Status.Phase = corev1.ClaimBound
	}
	return pvc
}

func TestSyncPreservesCapacity(t *testing.T) {
	for _, tc := range []struct{ name, desired, request, capacity, applied string }{
		{"decrease", "50Gi", "100Gi", "100Gi", "100Gi"},
		{"equal", "100Gi", "100Gi", "100Gi", "100Gi"},
		{"increase", "200Gi", "100Gi", "100Gi", "200Gi"},
		{"expansion in progress", "50Gi", "200Gi", "100Gi", "200Gi"},
		{"excess allocation", "50Gi", "80Gi", "100Gi", "80Gi"},
		{"request within capacity", "90Gi", "80Gi", "100Gi", "80Gi"},
		{"equivalent units", "1Ti", "1024Gi", "1024Gi", "1Ti"},
		{"capacity not reported", "50Gi", "100Gi", "", "100Gi"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			current := capacityPVC(tc.request, tc.capacity)
			cli := client.NewFakeClient(current)
			desired := capacityPVC(tc.desired, "")
			desired.Labels = map[string]string{"changed": "true"}
			require.NoError(t, SyncPVCs(ctx, cli, []*corev1.PersistentVolumeClaim{desired}))
			var actual corev1.PersistentVolumeClaim
			require.NoError(t, cli.Get(ctx, client.ObjectKeyFromObject(current), &actual))
			require.Zero(t, actual.Spec.Resources.Requests.Storage().Cmp(resource.MustParse(tc.applied)))
			require.Equal(t, "true", actual.Labels["changed"])
			require.Zero(t, desired.Spec.Resources.Requests.Storage().Cmp(resource.MustParse(tc.applied)), "apply returns the synced PVC")
		})
	}
	t.Run("new PVC", func(t *testing.T) {
		cli := client.NewFakeClient()
		desired := capacityPVC("50Gi", "")
		require.NoError(t, SyncPVCs(context.Background(), cli, []*corev1.PersistentVolumeClaim{desired}))
		var actual corev1.PersistentVolumeClaim
		require.NoError(t, cli.Get(context.Background(), client.ObjectKeyFromObject(desired), &actual))
		require.Zero(t, actual.Spec.Resources.Requests.Storage().Cmp(resource.MustParse("50Gi")))
	})
}
