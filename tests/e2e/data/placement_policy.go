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

package data

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

const defaultPlacementPolicyName = "placement-policy"

type PlacementPolicyPatch func(*v1alpha1.PlacementPolicy)

func NewPlacementPolicy(ns string, patches ...PlacementPolicyPatch) *v1alpha1.PlacementPolicy {
	policy := &v1alpha1.PlacementPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      defaultPlacementPolicyName,
		},
		Spec: v1alpha1.PlacementPolicySpec{
			Cluster: v1alpha1.ClusterReference{Name: defaultClusterName},
		},
	}
	for _, patch := range patches {
		patch(policy)
	}

	return policy
}

func WithPlacementPolicyName(name string) PlacementPolicyPatch {
	return func(policy *v1alpha1.PlacementPolicy) {
		policy.Name = name
	}
}

func WithPlacementPolicyCluster(cluster string) PlacementPolicyPatch {
	return func(policy *v1alpha1.PlacementPolicy) {
		policy.Spec.Cluster.Name = cluster
	}
}

func WithPlacementPolicyTiKVGroups(groups ...string) PlacementPolicyPatch {
	return func(policy *v1alpha1.PlacementPolicy) {
		refs := make([]v1alpha1.PlacementPolicyGroupRef, 0, len(groups))
		for _, group := range groups {
			refs = append(refs, v1alpha1.PlacementPolicyGroupRef{
				Group: v1alpha1.GroupName,
				Kind:  "TiKVGroup",
				Name:  group,
			})
		}
		policy.Spec.GroupRefs = refs
	}
}

func WithPlacementPolicyKeyspaceRule(name string, count int32, keyspaces ...string) PlacementPolicyPatch {
	return func(policy *v1alpha1.PlacementPolicy) {
		policy.Spec.Rules = []v1alpha1.PlacementPolicyRule{
			{
				Name:  name,
				Role:  v1alpha1.PlacementPolicyRoleVoter,
				Count: count,
				Selector: v1alpha1.PlacementPolicySelector{
					Keyspace: &v1alpha1.PlacementPolicyKeyspaceSelector{
						IDs: keyspaces,
					},
				},
			},
		}
	}
}
