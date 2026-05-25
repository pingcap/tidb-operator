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

func TestDMWorkerGroup(t *testing.T) {
	var cases []Case
	cases = append(cases, transferDMWorkerGroupCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases, transferDMWorkerGroupCases(t, NameLength(groupNameLengthLimit), "metadata", "name")...)
	cases = append(cases, transferDMWorkerGroupCases(t, MinReadySeconds(), "spec", "minReadySeconds")...)
	cases = append(cases, transferDMWorkerGroupCases(t, DMGroupRefRequired(), "spec", "dmGroupRef")...)
	Validate(t, "crd/core.pingcap.com_dmworkergroups.yaml", cases)
}

func DMGroupRefRequired() []Case {
	return []Case{
		{
			desc:     "dmGroupRef name is required",
			isCreate: true,
			current:  map[string]any{},
			wantErrs: []string{`spec.dmGroupRef.name: Required value`},
		},
		{
			desc:     "dmGroupRef name is set",
			isCreate: true,
			current:  map[string]any{"name": "dm-group"},
		},
	}
}

func basicDMWorkerGroup() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: DMWorkerGroup
metadata:
  name: dm-worker-group
spec:
  cluster:
    name: test
  dmGroupRef:
    name: dm-group
  replicas: 1
  template:
    spec:
      version: v8.5.2
      relayVolume:
        name: relay
        storage: 1Gi
        mounts:
        - type: relay-dir
`)
	obj := map[string]any{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		panic(err)
	}
	return obj
}

func transferDMWorkerGroupCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]
		current := basicDMWorkerGroup()
		c.current = Patch(t, c.mode, current, c.current, fields...)
		if c.isCreate {
			c.old = nil
			continue
		}
		old := basicDMWorkerGroup()
		c.old = Patch(t, c.mode, old, c.old, fields...)
	}
	return cases
}
