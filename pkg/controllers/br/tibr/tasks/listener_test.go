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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	task "github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestMigrateDefaultAPIListener(t *testing.T) {
	for _, owned := range []bool{false, true} {
		t.Run(fmt.Sprint(owned), func(t *testing.T) {
			cli := client.NewFakeClient()
			rtx := NewReconcileContext(context.Background(), types.NamespacedName{Namespace: "test", Name: "br"}, cli)
			rtx.tibr = &br.TiBR{ObjectMeta: metav1.ObjectMeta{Name: "br", Namespace: "test", UID: "br-uid"}}
			rtx.cluster = &core.Cluster{}
			sts := assembleSts(rtx)
			sts.Generation = 2
			sts.Status.ObservedGeneration = 1
			sts.Status.ReadyReplicas = 1
			sts.Spec.Template.Spec.Containers[0].Command[4] = fmt.Sprintf("0.0.0.0:%d", APIServerPort)
			sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, corev1.Container{Name: "sidecar", Image: "custom"})
			if !owned {
				sts.OwnerReferences = nil
			}
			require.NoError(t, cli.Create(context.Background(), sts))
			rtx.sts = sts
			result, _ := task.RunTask(context.Background(), taskRestrictAPIListener(rtx))
			if !owned {
				require.Equal(t, task.SFail, result.Status())
				return
			}
			require.Equal(t, task.SComplete, result.Status())
			require.NoError(t, cli.Get(context.Background(), client.ObjectKeyFromObject(sts), sts))
			require.Contains(t, sts.Spec.Template.Spec.Containers[0].Command, fmt.Sprintf("127.0.0.1:%d", APIServerPort))
			require.NotContains(t, sts.Spec.Template.Spec.Containers[0].Command, fmt.Sprintf("0.0.0.0:%d", APIServerPort))
			require.Equal(t, "custom", sts.Spec.Template.Spec.Containers[1].Image)
			rtx.configmap = &corev1.ConfigMap{}
			rtx.headlessSvc = &corev1.Service{}
			require.Contains(t, evaluateSyncedStatus(rtx), "statefulset rollout is not complete")
			sts.Status.ObservedGeneration = sts.Generation
			sts.Status.UpdatedReplicas = 1
			sts.Status.CurrentRevision = "old"
			sts.Status.UpdateRevision = "new"
			rtx.sts = sts
			require.Contains(t, evaluateSyncedStatus(rtx), "statefulset rollout is not complete")
			sts.Status.CurrentRevision = "new"
			sts.Status.UpdatedReplicas = 0
			require.Contains(t, evaluateSyncedStatus(rtx), "statefulset rollout is not complete")
			sts.Status.UpdatedReplicas = 1
			require.Empty(t, evaluateSyncedStatus(rtx))
		})
	}
}
