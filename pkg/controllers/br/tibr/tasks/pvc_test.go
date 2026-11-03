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

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1alpha1br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	task "github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestPatchPVCWithoutOverlay(t *testing.T) {
	for _, exists := range []bool{false, true} {
		t.Run(map[bool]string{false: "waiting for PVC", true: "patch PVC"}[exists], func(t *testing.T) {
			tibr := &v1alpha1br.TiBR{ObjectMeta: metav1.ObjectMeta{Name: "br", Namespace: "test"}, Spec: v1alpha1br.TiBRSpec{Volumes: []v1alpha1.Volume{{Name: "data"}}}}
			sts := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "br", Namespace: "test", UID: "sts-uid"}, Spec: appsv1.StatefulSetSpec{VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "data"}}}}}
			cli := client.NewFakeClient()
			pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data-br-0", Namespace: "test", Labels: TiBRSubResourceLabels(tibr)}}
			if exists {
				require.NoError(t, cli.Create(context.Background(), pvc))
			}
			rtx := NewReconcileContext(context.Background(), types.NamespacedName{Namespace: "test", Name: "br"}, cli)
			rtx.tibr, rtx.sts = tibr, sts
			result, _ := task.RunTask(context.Background(), taskPatchPVCWithOwnerReferences(rtx))
			if !exists {
				require.Equal(t, task.SRetry, result.Status())
				return
			}
			require.Equal(t, task.SComplete, result.Status())
			require.NoError(t, cli.Get(context.Background(), types.NamespacedName{Namespace: "test", Name: pvc.Name}, pvc))
			require.True(t, metav1.IsControlledBy(pvc, sts))
		})
	}
}
