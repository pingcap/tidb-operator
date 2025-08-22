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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestPD(t *testing.T) {
	var cases []Case
	cases = append(cases, transferPDCases(t, Topology(), "spec", "topology")...)
	cases = append(cases, transferPDCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases, transferPDCases(t, PDMode(), "spec", "mode")...)
	cases = append(cases, transferPDCases(t, PodOverlayLabels(), "spec", "overlay", "pod", "metadata")...)
	cases = append(cases, transferPDCases(t, OverlayVolumeClaims(true), "spec")...)
	cases = append(cases, transferPDCases(t, DataVolumeRequired(), "spec")...)
	cases = append(cases, transferPDCases(t, VolumeAttributesClassNameValidation(), "spec")...)
	cases = append(cases, transferPDCases(t, Version(), "spec", "version")...)
	cases = append(cases, transferPDCases(t, NameLength(instanceNameLengthLimit), "metadata", "name")...)
	Validate(t, "crd/core.pingcap.com_pds.yaml", cases)
}

func TestPDGroup(t *testing.T) {
	var cases []Case
	cases = append(cases, transferPDGroupCases(t, bootstrapped(), "spec", "bootstrapped")...)
	cases = append(cases, transferPDGroupCases(t, replicasWithBootstrapped(t), "spec")...)
	cases = append(cases, transferPDGroupCases(t, NameLength(groupNameLengthLimit), "metadata", "name")...)
	Validate(t, "crd/core.pingcap.com_pdgroups.yaml", cases)
}

func bootstrapped() []Case {
	errMsg := `spec.bootstrapped: Invalid value: "boolean": bootstrapped cannot be changed from true to false`
	cases := []Case{
		{
			desc:     "set to true when create",
			isCreate: true,
			current:  true,
		},
		{
			desc:     "set to false when create",
			isCreate: true,
			current:  false,
		},
		{
			desc:    "not change from false",
			old:     false,
			current: false,
		},
		{
			desc:    "not change from true",
			old:     true,
			current: true,
		},
		{
			desc:    "change from false to true",
			old:     false,
			current: true,
		},
		{
			desc:     "change from true to false",
			old:      true,
			current:  false,
			wantErrs: []string{errMsg},
		},
	}

	return cases
}

func replicasWithBootstrapped(t *testing.T) []Case {
	errMsg := `spec: Invalid value: "object": replicas cannot be changed when bootstrapped is false`

	basicSpec, ok, err := unstructured.NestedMap(basicPDGroup(), "spec")
	require.True(t, ok)
	require.NoError(t, err)

	mustCloneSpec := func() map[string]any {
		return runtime.DeepCopyJSON(basicSpec)
	}

	cases := []Case{
		{
			desc:     "set replicas to 1 when create",
			isCreate: true,
			current: func() map[string]any {
				spec := mustCloneSpec()
				spec["bootstrapped"] = false
				spec["replicas"] = int64(1)
				return spec
			}(),
		},
		{
			desc:     "set replicas to 3 when create",
			isCreate: true,
			current: func() map[string]any {
				spec := mustCloneSpec()
				spec["bootstrapped"] = true
				spec["replicas"] = int64(3)
				return spec
			}(),
		},
		{
			desc: "not change replicas when bootstrapped is false",
			old: func() map[string]any {
				spec := mustCloneSpec()
				spec["bootstrapped"] = false
				spec["replicas"] = int64(1)
				return spec
			}(),
			current: func() map[string]any {
				spec := mustCloneSpec()
				spec["bootstrapped"] = false
				spec["replicas"] = int64(1)
				return spec
			}(),
		},
		{
			desc: "change replicas when bootstrapped is true",
			old: func() map[string]any {
				spec := mustCloneSpec()
				spec["bootstrapped"] = true
				spec["replicas"] = int64(1)
				return spec
			}(),
			current: func() map[string]any {
				spec := mustCloneSpec()
				spec["bootstrapped"] = true
				spec["replicas"] = int64(3)
				return spec
			}(),
		},
		{
			desc: "change replicas from false to true bootstrapped",
			old: func() map[string]any {
				spec := mustCloneSpec()
				spec["bootstrapped"] = false
				spec["replicas"] = int64(1)
				return spec
			}(),
			current: func() map[string]any {
				spec := mustCloneSpec()
				spec["bootstrapped"] = true
				spec["replicas"] = int64(3)
				return spec
			}(),
		},
		{
			desc: "cannot change replicas when bootstrapped is false",
			old: func() map[string]any {
				spec := mustCloneSpec()
				spec["bootstrapped"] = false
				spec["replicas"] = int64(1)
				return spec
			}(),
			current: func() map[string]any {
				spec := mustCloneSpec()
				spec["bootstrapped"] = false
				spec["replicas"] = int64(3)
				return spec
			}(),
			wantErrs: []string{errMsg},
		},
	}

	return cases
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

func basicPDGroup() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: PDGroup
metadata:
  name: pd-group
spec:
  cluster:
    name: test
  replicas: 1
  template:
    spec:
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

func transferPDGroupCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicPDGroup()
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

		old := basicPDGroup()
		if c.old == nil {
			unstructured.RemoveNestedField(old, fields...)
		} else {
			require.NoError(t, unstructured.SetNestedField(old, c.old, fields...))
		}

		c.old = old
	}

	return cases
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
