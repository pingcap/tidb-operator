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
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestPlacementTiKVGroupLabelValue(t *testing.T) {
	require.Equal(t, "ns.kvg", PlacementTiKVGroupLabelValue("ns", "kvg"))
}

func TestTiKVStorePlacementExclusive(t *testing.T) {
	require.False(t, TiKVStorePlacementExclusive(nil))
	require.False(t, TiKVStorePlacementExclusive(&v1alpha1.TiKVStorePlacement{}))
	require.False(t, TiKVStorePlacementExclusive(&v1alpha1.TiKVStorePlacement{Exclusive: ptr.To(false)}))
	require.True(t, TiKVStorePlacementExclusive(&v1alpha1.TiKVStorePlacement{Exclusive: ptr.To(true)}))
}

func TestPlacementPolicyGroupID(t *testing.T) {
	require.Equal(t, "tidb-operator", PlacementPolicyGroupID())
}

func TestPlacementPolicyRuleIDPrefix(t *testing.T) {
	require.Equal(t, "policy:", PlacementPolicyRuleIDPrefix("policy"))
}

func TestPlacementPolicyRuleID(t *testing.T) {
	require.Equal(t, "policy:voters-1-txn", PlacementPolicyRuleID("policy", "voters", "1", "txn"))
}
