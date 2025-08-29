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

func TestScheduler(t *testing.T) {
	cases := append(
		transferSchedulerCases(t, Topology(), "spec", "topology"),
		transferSchedulerCases(t, ClusterReference(), "spec", "cluster")...,
	)
	cases = append(cases, transferSchedulerCases(t, PodOverlayLabels(), "spec", "overlay", "pod", "metadata")...)
	cases = append(cases, transferSchedulerCases(t, OverlayVolumeClaims(false), "spec")...)
	cases = append(cases, transferSchedulerCases(t, VolumeAttributesClassNameValidation(), "spec", "volumes")...)
	cases = append(cases, transferSchedulerCases(t, Version(), "spec", "version")...)
	cases = append(cases, transferSchedulerCases(t, NameLength(instanceNameLengthLimit), "metadata", "name")...)
	Validate(t, "crd/core.pingcap.com_schedulers.yaml", cases)
}

func basicScheduler() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: Scheduler
metadata:
  name: scheduler
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

func transferSchedulerCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicScheduler()
		c.current = Patch(t, c.mode, current, c.current, fields...)

		if c.isCreate {
			c.old = nil
			continue
		}

		old := basicScheduler()
		c.old = Patch(t, c.mode, old, c.old, fields...)
	}

	return cases
}
