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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestState(t *testing.T) {
	dw := fake.FakeObj("aaa-0", func(obj *v1alpha1.DMWorker) *v1alpha1.DMWorker {
		obj.Spec.Cluster.Name = "cluster"
		obj.Spec.DMGroupRef.Name = "dmg"
		return obj
	})
	cluster := fake.FakeObj[v1alpha1.Cluster]("cluster")
	pod := fake.FakeObj("aaa-dm-worker-0", fake.InstanceOwner[scope.DMWorker, corev1.Pod](dw))

	fc := client.NewFakeClient(dw, cluster, fake.FakeObj[v1alpha1.DMGroup]("dmg"), pod)
	s := NewState(types.NamespacedName{Name: "aaa-0"})

	res, done := task.RunTask(context.Background(), task.Block(
		common.TaskContextObject[scope.DMWorker](s, fc),
		common.TaskContextCluster[scope.DMWorker](s, fc),
		common.TaskContextPod[scope.DMWorker](s, fc),
	))

	assert.Equal(t, task.SComplete, res.Status())
	assert.False(t, done)
	assert.Equal(t, dw, s.DMWorker())
	assert.Equal(t, cluster, s.Cluster())
	assert.Equal(t, pod, s.Pod())
}
