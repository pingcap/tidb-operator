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

	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestTSO(t *testing.T) {
	cases := append(
		transferTSOCases(t, Topology(), "spec", "topology"),
		transferTSOCases(t, ClusterReference(), "spec", "cluster")...,
	)
	cases = append(cases, transferTSOCases(t, PodOverlayLabels(), "spec", "overlay", "pod", "metadata")...)
	cases = append(cases, transferTSOCases(t, OverlayVolumeClaims(false), "spec")...)
	cases = append(cases, transferTSOCases(t, VolumeAttributesClassNameValidation(), "spec", "volumes")...)
	cases = append(cases, transferTSOCases(t, Version(), "spec", "version")...)
	cases = append(cases, transferTSOCases(t, NameLength(instanceNameLengthLimit), "metadata", "name")...)
	Validate(t, "crd/core.pingcap.com_tsos.yaml", cases)
}

func TestTSOGroup(t *testing.T) {
	var cases []Case
	cases = append(cases, transferTSOGroupCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases, transferTSOGroupCases(t, NameLength(groupNameLengthLimit), "metadata", "name")...)
	cases = append(cases, transferTSOGroupCases(t, MinReadySeconds(), "spec", "minReadySeconds")...)
	Validate(t, "crd/core.pingcap.com_tsogroups.yaml", cases)
}

func basicTSO() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: TSO
metadata:
  name: tso
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

func transferTSOCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicTSO()
		c.current = Patch(t, c.mode, current, c.current, fields...)

		if c.isCreate {
			c.old = nil
			continue
		}

		old := basicTSO()
		c.old = Patch(t, c.mode, old, c.old, fields...)
	}

	return cases
}

func basicTSOGroup() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: TSOGroup
metadata:
  name: tso-group
spec:
  cluster:
    name: test
  replicas: 2
  template:
    spec:
      version: v8.1.0
`)
	obj := map[string]any{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		panic(err)
	}

	return obj
}

func transferTSOGroupCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicTSOGroup()
		c.current = Patch(t, c.mode, current, c.current, fields...)

		if c.isCreate {
			c.old = nil
			continue
		}

		old := basicTSOGroup()
		c.old = Patch(t, c.mode, old, c.old, fields...)
	}

	return cases
}
