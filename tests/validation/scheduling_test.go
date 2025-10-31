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

func TestScheduling(t *testing.T) {
	cases := append(
		transferSchedulingCases(t, Topology(), "spec", "topology"),
		transferSchedulingCases(t, ClusterReference(), "spec", "cluster")...,
	)
	cases = append(cases, transferSchedulingCases(t, PodOverlayLabels(), "spec", "overlay", "pod", "metadata")...)
	cases = append(cases, transferSchedulingCases(t, OverlayVolumeClaims(false), "spec")...)
	cases = append(cases, transferSchedulingCases(t, VolumeAttributesClassNameValidation(), "spec", "volumes")...)
	cases = append(cases, transferSchedulingCases(t, Version(), "spec", "version")...)
	cases = append(cases, transferSchedulingCases(t, NameLength(instanceNameLengthLimit), "metadata", "name")...)
	Validate(t, "crd/core.pingcap.com_schedulings.yaml", cases)
}

func TestSchedulingGroup(t *testing.T) {
	var cases []Case
	cases = append(cases, transferSchedulingGroupCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases, transferSchedulingGroupCases(t, NameLength(groupNameLengthLimit), "metadata", "name")...)
	cases = append(cases, transferSchedulingGroupCases(t, MinReadySeconds(), "spec", "minReadySeconds")...)
	Validate(t, "crd/core.pingcap.com_schedulinggroups.yaml", cases)
}

func basicScheduling() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: Scheduling
metadata:
  name: scheduling
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

func transferSchedulingCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicScheduling()
		c.current = Patch(t, c.mode, current, c.current, fields...)

		if c.isCreate {
			c.old = nil
			continue
		}

		old := basicScheduling()
		c.old = Patch(t, c.mode, old, c.old, fields...)
	}

	return cases
}

func basicSchedulingGroup() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: SchedulingGroup
metadata:
  name: scheduling-group
spec:
  cluster:
    name: test
  replicas: 3
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

func transferSchedulingGroupCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicSchedulingGroup()
		c.current = Patch(t, c.mode, current, c.current, fields...)

		if c.isCreate {
			c.old = nil
			continue
		}

		old := basicSchedulingGroup()
		c.old = Patch(t, c.mode, old, c.old, fields...)
	}

	return cases
}
