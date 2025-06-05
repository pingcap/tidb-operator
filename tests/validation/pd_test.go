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

func TestPD(t *testing.T) {
	var cases []Case
	cases = append(cases,
		transferPDCases(t, Topology(), "spec", "topology")...)
	cases = append(cases,
		transferPDCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases,
		transferPDCases(t, PDMode(), "spec", "mode")...)
	cases = append(cases,
		transferPDCases(t, PodOverlayLabels(), "spec", "overlay", "pod", "metadata")...)

	Validate(t, "crd/core.pingcap.com_pds.yaml", cases)
}

func PDMode() []Case {
	cases := []Case{
		{
			desc:     "ms mode is supported",
			isCreate: true,
			current:  "ms",
		},
		{
			desc:     "normal mode is supported",
			isCreate: true,
			current:  "",
		},
		{
			desc:     "unset mode is supported",
			isCreate: true,
		},
		{
			desc: "mode is always unset",
		},
		{
			desc:    "mode is always empty",
			old:     "",
			current: "",
		},
		{
			desc:    "mode is always ms",
			old:     "ms",
			current: "ms",
		},
		{
			desc:    "mode is changed to ms",
			old:     "",
			current: "ms",
			wantErrs: []string{
				`spec.mode: Invalid value: "string": pd mode is immutable now, it may be supported later`,
			},
		},
		{
			desc:    "mode is set to ms",
			current: "ms",
			wantErrs: []string{
				`spec.mode: Invalid value: "object": pd mode can only be set when creating now`,
			},
		},
		{
			desc:    "mode is changed to normal",
			old:     "ms",
			current: "",
			wantErrs: []string{
				`spec.mode: Invalid value: "string": pd mode is immutable now, it may be supported later`,
			},
		},
		{
			desc: "mode is unset",
			old:  "ms",
			wantErrs: []string{
				`spec.mode: Invalid value: "object": pd mode can only be set when creating now`,
			},
		},
	}

	return cases
}

func basicPD() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: PD
metadata:
  name: pd
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

func transferPDCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicPD()
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

		old := basicPD()
		if c.old == nil {
			unstructured.RemoveNestedField(old, fields...)
		} else {
			require.NoError(t, unstructured.SetNestedField(old, c.old, fields...))
		}

		c.old = old
	}

	return cases
}
