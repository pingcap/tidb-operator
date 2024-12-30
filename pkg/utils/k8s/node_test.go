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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func TestGetNodeLabelsForKeys(t *testing.T) {
	node := fake.FakeObj[corev1.Node]("node",
		fake.Label[corev1.Node]("foo", "bar"),
		fake.Label[corev1.Node](corev1.LabelTopologyRegion, "us-west1"),
		fake.Label[corev1.Node](corev1.LabelTopologyZone, "us-west1-a"),
		fake.Label[corev1.Node](corev1.LabelHostname, "k8s-node-1"),
		fake.Label[corev1.Node]("rack", "rack-1"),
	)
	labels := GetNodeLabelsForKeys(node, []string{"region", "zone", "host", "rack"})
	require.Len(t, labels, 4)
	assert.Equal(t, "us-west1", labels["region"])
	assert.Equal(t, "us-west1-a", labels["zone"])
	assert.Equal(t, "k8s-node-1", labels["host"])
	assert.Equal(t, "rack-1", labels["rack"])
}
