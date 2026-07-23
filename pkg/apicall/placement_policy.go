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
	"cmp"
	"context"
	"slices"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func ListPlacementPolicies(ctx context.Context, c client.Client, ns, cluster string) ([]*v1alpha1.PlacementPolicy, error) {
	var list v1alpha1.PlacementPolicyList
	if err := c.List(ctx, &list,
		client.InNamespace(ns),
		client.MatchingFields{"spec.cluster.name": cluster},
	); err != nil {
		return nil, err
	}

	policies := make([]*v1alpha1.PlacementPolicy, 0, len(list.Items))
	for i := range list.Items {
		policies = append(policies, &list.Items[i])
	}
	slices.SortFunc(policies, func(a, b *v1alpha1.PlacementPolicy) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return policies, nil
}

func ListPlacementPoliciesReferencingTiKVGroup(
	ctx context.Context,
	c client.Client,
	kvg *v1alpha1.TiKVGroup,
) ([]*v1alpha1.PlacementPolicy, error) {
	policies, err := ListPlacementPolicies(ctx, c, kvg.Namespace, kvg.Spec.Cluster.Name)
	if err != nil {
		return nil, err
	}

	out := make([]*v1alpha1.PlacementPolicy, 0, len(policies))
	for _, policy := range policies {
		if coreutil.PlacementPolicyReferencesTiKVGroup(policy, kvg.Name) {
			out = append(out, policy)
		}
	}

	return out, nil
}
