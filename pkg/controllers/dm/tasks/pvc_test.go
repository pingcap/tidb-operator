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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
)

func TestPVCNewer(t *testing.T) {
	sc := "standard"
	size := resource.MustParse("10Gi")
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
			dm := newTestDM("aaa-0", func(dm *v1alpha1.DM) {
				dm.Spec.DataVolume.StorageClassName = &sc
				dm.Spec.DataVolume.Storage = size
			})

			pvcs := PVCNewer().NewPVCs(cluster, dm, features.NewFromFeatures(nil))
			if assert.Len(t, pvcs, 1) {
				pvc := pvcs[0]
				assert.Equal(t, "data-aaa-dm-master-0", pvc.Name)
				assert.Equal(t, &sc, pvc.Spec.StorageClassName)
				assert.Equal(t, corev1.ReadWriteOnce, pvc.Spec.AccessModes[0])
				assert.Equal(t, size, pvc.Spec.Resources.Requests[corev1.ResourceStorage])
				assert.Equal(t, c.clusterID, pvc.Labels[v1alpha1.LabelKeyClusterID])
			}
		})
	}
}
