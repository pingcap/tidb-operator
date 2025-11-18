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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
)

func TestPVCNewer(t *testing.T) {
	cases := []struct {
		desc      string
		c         *v1alpha1.Cluster
		obj       *v1alpha1.Scheduler
		enableVAC bool

		patch func(pvc *corev1.PersistentVolumeClaim)
	}{
		{
			desc: "normal",
			c: fake.FakeObj("aaa", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				return obj
			}),
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.Scheduler) *v1alpha1.Scheduler {
				return obj
			}),

			patch: func(_ *corev1.PersistentVolumeClaim) {},
		},
	}
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()
			pvcs := coreutil.PVCs[scope.Scheduler](c.c, c.obj, coreutil.EnableVAC(c.enableVAC))

			for _, pvc := range pvcs {
				c.patch(pvc)
			}

			fg := features.NewFromFeatures(nil)
			if c.enableVAC {
				fg = features.NewFromFeatures([]meta.Feature{
					meta.VolumeAttributesClass,
				})
			}

			actual := PVCNewer().NewPVCs(c.c, c.obj, fg)

			assert.Equal(tt, pvcs, actual)
		})
	}
}
