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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func TestListPlacementPolicies(t *testing.T) {
	ctx := context.Background()
	cli := client.NewFakeClient(
		fakePlacementPolicy("z", "c", "g1"),
		fakePlacementPolicy("a", "c", "g2"),
		fakePlacementPolicy("other", "other", "g1"),
	)

	policies, err := ListPlacementPolicies(ctx, cli, "ns", "c")

	require.NoError(t, err)
	require.Equal(t, []string{"a", "z"}, placementPolicyNames(policies))
}

func TestListPlacementPoliciesReferencingTiKVGroup(t *testing.T) {
	ctx := context.Background()
	kvg := &v1alpha1.TiKVGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "g1",
		},
		Spec: v1alpha1.TiKVGroupSpec{
			Cluster: v1alpha1.ClusterReference{Name: "c"},
		},
	}
	cli := client.NewFakeClient(
		kvg,
		fakePlacementPolicy("ref-z", "c", "g1"),
		fakePlacementPolicy("ref-a", "c", "g1"),
		fakePlacementPolicy("other-ref", "other", "g1"),
		fakePlacementPolicy("not-ref", "c", "g2"),
	)

	policies, err := ListPlacementPoliciesReferencingTiKVGroup(ctx, cli, kvg)

	require.NoError(t, err)
	require.Equal(t, []string{"ref-a", "ref-z"}, placementPolicyNames(policies))
}

func fakePlacementPolicy(name, cluster string, groups ...string) *v1alpha1.PlacementPolicy {
	refs := make([]v1alpha1.PlacementPolicyGroupRef, 0, len(groups))
	for _, group := range groups {
		refs = append(refs, v1alpha1.PlacementPolicyGroupRef{
			Group: v1alpha1.GroupName,
			Kind:  "TiKVGroup",
			Name:  group,
		})
	}

	return &v1alpha1.PlacementPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      name,
		},
		Spec: v1alpha1.PlacementPolicySpec{
			Cluster:   v1alpha1.ClusterReference{Name: cluster},
			GroupRefs: refs,
		},
	}
}

func placementPolicyNames(policies []*v1alpha1.PlacementPolicy) []string {
	names := make([]string, 0, len(policies))
	for _, policy := range policies {
		names = append(names, policy.Name)
	}
	return names
}
