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
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
)

func TestPVCNewer(t *testing.T) {
	sc := "standard"
	size := resource.MustParse("20Gi")
	cases := []struct {
		desc      string
		clusterID string
	}{
		{desc: "basic PVC"},
		{desc: "with cluster ID labels", clusterID: "cluster-id"},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(t *testing.T) {
			t.Parallel()

			cluster := fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				obj.Status.ID = c.clusterID
				return obj
			})
			dw := newTestDMWorker("aaa-0", func(dw *v1alpha1.DMWorker) {
				dw.Spec.RelayVolume.StorageClassName = &sc
				dw.Spec.RelayVolume.Storage = size
			})

			pvcs := PVCNewer().NewPVCs(cluster, dw, features.NewFromFeatures(nil))
			if assert.Len(t, pvcs, 1) {
				pvc := pvcs[0]
				assert.Equal(t, "relay-aaa-dm-worker-0", pvc.Name)
				assert.Equal(t, &sc, pvc.Spec.StorageClassName)
				assert.Equal(t, corev1.ReadWriteOnce, pvc.Spec.AccessModes[0])
				assert.Equal(t, size, pvc.Spec.Resources.Requests[corev1.ResourceStorage])
				assert.Equal(t, c.clusterID, pvc.Labels[v1alpha1.LabelKeyClusterID])
			}
		})
	}
}

func TestPVCNewerAppliesRelayAndAdditionalVolumeOverlays(t *testing.T) {
	cluster := fake.FakeObj[v1alpha1.Cluster]("cluster")
	dw := newTestDMWorker("aaa-0", func(dw *v1alpha1.DMWorker) {
		dw.Spec.Volumes = []v1alpha1.Volume{{
			Name:    "extra",
			Storage: resource.MustParse("5Gi"),
		}}
		dw.Spec.Overlay = &v1alpha1.Overlay{
			PersistentVolumeClaims: []v1alpha1.NamedPersistentVolumeClaimOverlay{
				{
					Name: "relay",
					PersistentVolumeClaim: v1alpha1.PersistentVolumeClaimOverlay{
						ObjectMeta: v1alpha1.ObjectMeta{
							Labels:      map[string]string{"cloud-tag": "dm-worker"},
							Annotations: map[string]string{"owner": "relay"},
						},
					},
				},
				{
					Name: "extra",
					PersistentVolumeClaim: v1alpha1.PersistentVolumeClaimOverlay{
						ObjectMeta: v1alpha1.ObjectMeta{
							Labels: map[string]string{"cloud-tag": "extra"},
						},
					},
				},
			},
		}
	})

	pvcs := PVCNewer().NewPVCs(cluster, dw, features.NewFromFeatures(nil))
	require.Len(t, pvcs, 2)
	assert.Equal(t, "dm-worker", pvcs[0].Labels["cloud-tag"])
	assert.Equal(t, "relay", pvcs[0].Annotations["owner"])
	assert.Equal(t, "extra", pvcs[1].Labels["cloud-tag"])
}
