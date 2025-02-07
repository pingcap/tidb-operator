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

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
)

func TestFinalizer(t *testing.T) {
	cases := []struct {
		desc         string
		cluster      *v1alpha1.Cluster
		pdGroup      *v1alpha1.PDGroup
		expected     task.Result
		hasFinalizer bool
	}{
		{
			desc:         "ensured finalizer",
			cluster:      fake.FakeObj[v1alpha1.Cluster]("test"),
			expected:     task.Complete().With("ensured finalizer"),
			hasFinalizer: true,
		},
		{
			desc: "removed finalizer",
			cluster: fake.FakeObj("test",
				fake.DeleteNow[v1alpha1.Cluster](), fake.AddFinalizer[v1alpha1.Cluster]()),
			expected: task.Complete().Break().With("removed finalizer"),
		},
		{
			desc: "deleting components",
			cluster: fake.FakeObj("test",
				fake.DeleteNow[v1alpha1.Cluster](), fake.AddFinalizer[v1alpha1.Cluster]()),
			pdGroup:      fake.FakeObj[v1alpha1.PDGroup]("pd-group"),
			expected:     task.Fail().With("deleting components"),
			hasFinalizer: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := FakeContext(types.NamespacedName{Name: "test"})
			ctx.Cluster = c.cluster
			ctx.PDGroup = c.pdGroup

			fc := client.NewFakeClient(c.cluster)
			tk := NewTaskFinalizer(logr.Discard(), fc)
			res := tk.Sync(ctx)
			assert.Equal(tt, c.expected, res)
			assert.Equal(tt, c.hasFinalizer, controllerutil.ContainsFinalizer(c.cluster, v1alpha1.Finalizer))
		})
	}
}
