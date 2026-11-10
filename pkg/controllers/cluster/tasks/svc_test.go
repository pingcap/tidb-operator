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
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func TestPDServiceOwnershipRoundTrip(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "default", UID: "cluster-uid"},
	}
	group := &v1alpha1.PDGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "group", Namespace: cluster.Namespace, UID: "group-uid"},
		Spec:       v1alpha1.PDGroupSpec{Cluster: v1alpha1.ClusterReference{Name: cluster.Name}},
	}
	cluster.Spec.CustomizedPDServiceName = ptr.To("group-pd")
	cluster.Spec.FeatureGates = []metav1alpha1.FeatureGate{
		{Name: metav1alpha1.FeatureModification},
		{Name: metav1alpha1.MultiPDGroup},
	}
	svc := newPDService(cluster)
	svc.UID = "service"
	svc.ResourceVersion = "1"
	svc.Spec.ClusterIP = "10.0.0.1"
	svc.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(group, v1alpha1.SchemeGroupVersion.WithKind("PDGroup")),
	}
	fc := client.NewFakeClient(svc)
	task := NewTaskService(logr.Discard(), fc)
	ctx := &ReconcileContext{Context: t.Context(), Cluster: cluster, PDGroups: []*v1alpha1.PDGroup{group}}
	key := ctrlclient.ObjectKeyFromObject(svc)

	require.False(t, task.Sync(ctx).IsFailed())
	var got corev1.Service
	require.NoError(t, fc.Get(ctx, key, &got))
	require.True(t, metav1.IsControlledBy(&got, cluster))

	cluster.Spec.FeatureGates = cluster.Spec.FeatureGates[:1]
	require.False(t, task.Sync(ctx).IsFailed())
	require.NoError(t, fc.Get(ctx, key, &got))
	require.True(t, metav1.IsControlledBy(&got, group))
	require.Equal(t, svc.UID, got.UID)
	require.Equal(t, svc.Spec.ClusterIP, got.Spec.ClusterIP)
}
