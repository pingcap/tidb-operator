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
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestGetResourceRequirements(t *testing.T) {
	cpu, err := resource.ParseQuantity("100m")
	require.NoError(t, err)
	memory, err := resource.ParseQuantity("100Mi")
	require.NoError(t, err)
	req := v1alpha1.ResourceRequirements{
		CPU:    &cpu,
		Memory: &memory,
	}

	res := GetResourceRequirements(req)
	assert.Equal(t, res.Requests[corev1.ResourceCPU], cpu)
	assert.Equal(t, res.Limits[corev1.ResourceCPU], cpu)
	assert.Equal(t, res.Requests[corev1.ResourceMemory], memory)
	assert.Equal(t, res.Limits[corev1.ResourceMemory], memory)
}

func TestPromAnno(t *testing.T) {
	anno := AnnoProm(8234, "/metrics")
	require.Equal(t, map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   "8234",
		"prometheus.io/path":   "/metrics",
	}, anno)

	annoAdditional := AnnoAdditionalProm("tiflash.proxy", 20292)
	require.Equal(t, map[string]string{
		"tiflash.proxy.prometheus.io/port": "20292",
	}, annoAdditional)
}

func TestLabelsK8sApp(t *testing.T) {
	labels := LabelsK8sApp("db", "tikv")
	require.Equal(t, map[string]string{
		"app.kubernetes.io/component":  "tikv",
		"app.kubernetes.io/instance":   "db",
		"app.kubernetes.io/managed-by": "tidb-operator",
		"app.kubernetes.io/name":       "tidb-cluster",
	}, labels)
}
