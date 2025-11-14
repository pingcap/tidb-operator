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

package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
)

func TestEnsureGroupSubResourceDeleted(t *testing.T) {
	//nolint:goconst // test
	ns := "ns1"
	gpName := "basic"
	cli := client.NewFakeClient(
		fake.FakeObj("svc1", fake.SetNamespace[corev1.Service](ns),
			fake.Label[corev1.Service](v1alpha1.LabelKeyGroup, gpName),
			fake.Label[corev1.Service](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator)),
		fake.FakeObj("svc2", fake.SetNamespace[corev1.Service](ns),
			fake.Label[corev1.Service](v1alpha1.LabelKeyGroup, gpName),
			fake.Label[corev1.Service](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator)))
	// list to ensure the services are created
	svcs := &corev1.ServiceList{}
	err := cli.List(context.Background(), svcs, client.InNamespace(ns))
	require.NoError(t, err)
	require.Len(t, svcs.Items, 2)

	// return an error with "wait for all sub resources of ns1/basic being removed"
	err = EnsureGroupSubResourceDeleted(context.Background(), cli, ns, gpName)
	require.Error(t, err)
	require.Contains(t, err.Error(), "wait for all sub resources of ns1/basic being removed")
	err = EnsureGroupSubResourceDeleted(context.Background(), cli, ns, gpName)
	require.NoError(t, err)

	// list to ensure the services are deleted
	svcs = &corev1.ServiceList{}
	err = cli.List(context.Background(), svcs, client.InNamespace(ns))
	require.NoError(t, err)
}

func TestDeleteInstanceSubresources(t *testing.T) {
	ns := "ns1"
	pd := &v1alpha1.PD{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "basic",
		},
		Spec: v1alpha1.PDSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: "basic",
			},
		},
	}
	pod := fake.FakeObj("pod", fake.SetNamespace[corev1.Pod](ns),
		fake.Label[corev1.Pod](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
		fake.Label[corev1.Pod](v1alpha1.LabelKeyInstance, pd.GetName()),
		fake.Label[corev1.Pod](v1alpha1.LabelKeyCluster, pd.Spec.Cluster.Name),
		fake.Label[corev1.Pod](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
	)
	cli := client.NewFakeClient(pd, pod)
	// list to ensure the pod is created
	pods := &corev1.PodList{}
	err := cli.List(context.Background(), pods, client.InNamespace(ns))
	require.NoError(t, err)
	require.Len(t, pods.Items, 1)

	// should wait for the pod to be deleted
	wait, err := DeleteInstanceSubresource(context.Background(), cli, runtime.FromPD(pd), &corev1.PodList{})
	require.NoError(t, err)
	require.True(t, wait)

	// list to ensure the pod is deleted
	pods = &corev1.PodList{}
	err = cli.List(context.Background(), pods, client.InNamespace(ns))
	require.NoError(t, err)
	require.Empty(t, pods.Items)

	// should not wait for the pod to be deleted
	wait, err = DeleteInstanceSubresource(context.Background(), cli, runtime.FromPD(pd), &corev1.PodList{})
	require.NoError(t, err)
	require.False(t, wait)
}

func TestDeleteGroupSubresources(t *testing.T) {
	ns := "ns1"
	pdg := &v1alpha1.PDGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "basic",
		},
		Spec: v1alpha1.PDGroupSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: "basic",
			},
		},
	}
	svc := fake.FakeObj("svc", fake.SetNamespace[corev1.Service](ns),
		fake.Label[corev1.Service](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
		fake.Label[corev1.Service](v1alpha1.LabelKeyGroup, pdg.GetName()),
		fake.Label[corev1.Service](v1alpha1.LabelKeyCluster, pdg.Spec.Cluster.Name),
		fake.Label[corev1.Service](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
	)
	cli := client.NewFakeClient(pdg, svc)
	// list to ensure the service is created
	svcs := &corev1.ServiceList{}
	err := cli.List(context.Background(), svcs, client.InNamespace(ns))
	require.NoError(t, err)
	require.Len(t, svcs.Items, 1)

	// should wait for the service to be deleted
	wait, err := DeleteGroupSubresource(context.Background(), cli, runtime.FromPDGroup(pdg), &corev1.ServiceList{})
	require.NoError(t, err)
	require.True(t, wait)

	// list to ensure the service is deleted
	svcs = &corev1.ServiceList{}
	err = cli.List(context.Background(), svcs, client.InNamespace(ns))
	require.NoError(t, err)
	require.Empty(t, svcs.Items)

	// should not wait for the service to be deleted
	wait, err = DeleteGroupSubresource(context.Background(), cli, runtime.FromPDGroup(pdg), &corev1.ServiceList{})
	require.NoError(t, err)
	require.False(t, wait)
}
