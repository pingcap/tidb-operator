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

func TestTiProxy(t *testing.T) {
	cases := []Case{}
	cases = append(cases, transferTiProxyCases(t, Topology(), "spec", "topology")...)
	cases = append(cases, transferTiProxyCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases, transferTiProxyCases(t, ServerLabels(), "spec", "server", "labels")...)
	cases = append(cases, transferTiProxyCases(t, OverlayVolumeClaims(false), "spec")...)
	cases = append(cases, transferTiProxyCases(t, Version(), "spec", "version")...)
	cases = append(cases, transferTiProxyCases(t, NameLength(instanceNameLengthLimit), "metadata", "name")...)
	Validate(t, "crd/core.pingcap.com_tiproxies.yaml", cases)
}

func basicTiProxy() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: TiProxy
metadata:
  name: tiproxy
spec:
  cluster:
    name: test
  subdomain: test
  version: v8.1.0
`)
	obj := map[string]any{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		panic(err)
	}

	return obj
}

func transferTiProxyCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicTiProxy()
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

		old := basicTiProxy()
		if c.old == nil {
			unstructured.RemoveNestedField(old, fields...)
		} else {
			require.NoError(t, unstructured.SetNestedField(old, c.old, fields...))
		}

		c.old = old
	}

	return cases
}
