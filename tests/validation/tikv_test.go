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

package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestTiKV(t *testing.T) {
	cases := []Case{}
	cases = append(cases, transferTiKVCases(t, Topology(), "spec", "topology")...)
	cases = append(cases, transferTiKVCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases, transferTiKVCases(t, PodOverlayLabels(), "spec", "overlay", "pod", "metadata")...)
	cases = append(cases, transferTiKVCases(t, OverlayVolumeClaims(), "spec")...)
	cases = append(cases, transferTiKVCases(t, DataVolumeRequired(), "spec")...)
	cases = append(cases, transferTiKVCases(t, Version(), "spec", "version")...)
	Validate(t, "crd/core.pingcap.com_tikvs.yaml", cases)
}

func basicTiKV() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: TiKV
metadata:
  name: tikv
spec:
  cluster:
    name: test
  subdomain: test
  version: v8.1.0
  volumes:
  - name: data
    mounts:
    - type: data
    storage: 20Gi
`)
	obj := map[string]any{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		panic(err)
	}

	return obj
}

func transferTiKVCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicTiKV()
		if c.current == nil {
			unstructured.RemoveNestedField(current, fields...)
		} else {
			require.NoError(t, unstructured.SetNestedField(current, c.current, fields...))
		}

		c.current = current

		if c.isCreate {
			c.old = nil
			continue
		}

		old := basicTiKV()
		if c.old == nil {
			unstructured.RemoveNestedField(old, fields...)
		} else {
			require.NoError(t, unstructured.SetNestedField(old, c.old, fields...))
		}

		c.old = old
	}

	return cases
}
