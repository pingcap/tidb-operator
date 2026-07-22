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

package coreutil

import (
	"fmt"
	"slices"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func PlacementTiKVGroupLabelValue(namespace, group string) string {
	return fmt.Sprintf("%s.%s", namespace, group)
}

func TiKVStorePlacementExclusive(placement *v1alpha1.TiKVStorePlacement) bool {
	return placement != nil && placement.Exclusive != nil && *placement.Exclusive
}

func PlacementPolicyGroupID() string {
	return "tidb-operator"
}

func PlacementPolicyRuleIDPrefix(policyName string) string {
	return policyName + ":"
}

func PlacementPolicyRuleID(policyName, ruleName, keyspaceID, keyRangeType string) string {
	id := fmt.Sprintf("%s%s-%s-%s", PlacementPolicyRuleIDPrefix(policyName), ruleName, keyspaceID, keyRangeType)

	return id
}

func PlacementPolicyTiKVGroupRefNames(policy *v1alpha1.PlacementPolicy) []string {
	names := []string{}
	seen := map[string]struct{}{}
	for _, ref := range policy.Spec.GroupRefs {
		if ref.Group != v1alpha1.GroupName || ref.Kind != "TiKVGroup" || ref.Name == "" {
			continue
		}
		if _, ok := seen[ref.Name]; ok {
			continue
		}
		seen[ref.Name] = struct{}{}
		names = append(names, ref.Name)
	}
	slices.Sort(names)

	return names
}

func PlacementPolicyReferencesTiKVGroup(policy *v1alpha1.PlacementPolicy, name string) bool {
	_, ok := slices.BinarySearch(PlacementPolicyTiKVGroupRefNames(policy), name)
	return ok
}
