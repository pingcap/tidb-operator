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

package apicall

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
)

func TestListPods(t *testing.T) {
	pdg := fake.FakeObj[v1alpha1.PDGroup]("pdg",
		fake.SetNamespace[v1alpha1.PDGroup]("ns"),
		func(pdg *v1alpha1.PDGroup) *v1alpha1.PDGroup {
			pdg.Spec.Cluster.Name = "tc"
			return pdg
		},
	)
	labels := []fake.ChangeFunc[corev1.Pod, *corev1.Pod]{
		fake.SetNamespace[corev1.Pod]("ns"),
		fake.Label[corev1.Pod](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
		fake.Label[corev1.Pod](v1alpha1.LabelKeyCluster, "tc"),
		fake.Label[corev1.Pod](v1alpha1.LabelKeyGroup, "pdg"),
		fake.Label[corev1.Pod](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
	}

	cli := client.NewFakeClient(
		fake.FakeObj("pod-b", labels...),
		fake.FakeObj("pod-a", labels...),
		fake.FakeObj("pod-other", fake.SetNamespace[corev1.Pod]("ns")),
	)

	pods, err := ListPods[scope.PDGroup](context.Background(), cli, pdg)

	require.NoError(t, err)
	require.Len(t, pods.Items, 2)
	require.Equal(t, "pod-a", pods.Items[0].Name)
	require.Equal(t, "pod-b", pods.Items[1].Name)
}
