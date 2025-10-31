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

func TestTiDB(t *testing.T) {
	cases := []Case{}
	cases = append(cases, transferTiDBCases(t, Topology(), "spec", "topology")...)
	cases = append(cases, transferTiDBCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases, transferTiDBCases(t, ServerLabels(), "spec", "server", "labels")...)
	cases = append(cases, transferTiDBCases(t, PodOverlayLabels(), "spec", "overlay", "pod", "metadata")...)
	cases = append(cases, transferTiDBCases(t, mysqlTLS(), "spec", "security", "tls", "mysql")...)
	cases = append(cases, transferTiDBCases(t, OverlayVolumeClaims(false), "spec")...)
	cases = append(cases, transferTiDBCases(t, Version(), "spec", "version")...)
	cases = append(cases, transferTiDBCases(t, NameLength(instanceNameLengthLimit), "metadata", "name")...)
	cases = append(cases, transferTiDBCases(t, keyspace(), "spec")...)
	Validate(t, "crd/core.pingcap.com_tidbs.yaml", cases)
}

func TestTiDBGroup(t *testing.T) {
	var cases []Case
	cases = append(cases, transferTiDBGroupCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases, transferTiDBGroupCases(t, NameLength(groupNameLengthLimit), "metadata", "name")...)
	cases = append(cases, transferTiDBGroupCases(t, MinReadySeconds(), "spec", "minReadySeconds")...)
	Validate(t, "crd/core.pingcap.com_tidbgroups.yaml", cases)
}

func basicTiDB() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: TiDB
metadata:
  name: tidb
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

func transferTiDBCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicTiDB()
		c.current = Patch(t, c.mode, current, c.current, fields...)

		if c.isCreate {
			c.old = nil
			continue
		}

		old := basicTiDB()
		c.old = Patch(t, c.mode, old, c.old, fields...)
	}

	return cases
}

func mysqlTLS() []Case {
	errMsg := `spec.security.tls.mysql: Invalid value: "object": field .mysql.enabled is immutable`
	return []Case{
		{
			desc:     "set enabled to true when creating",
			isCreate: true,
			current:  map[string]any{"enabled": true},
		},
		{
			desc:     "set enabled to false when creating",
			isCreate: true,
			current:  map[string]any{"enabled": false},
		},
		{
			desc:     "try to update enabled from false to true",
			old:      map[string]any{"enabled": false},
			current:  map[string]any{"enabled": true},
			wantErrs: []string{errMsg},
		},
		{
			desc:     "try to update enabled from true to false",
			old:      map[string]any{"enabled": true},
			current:  map[string]any{"enabled": false},
			wantErrs: []string{errMsg},
		},
		{
			desc:    "enabled is not changed",
			old:     map[string]any{"enabled": true},
			current: map[string]any{"enabled": true},
		},
	}
}

func keyspace() []Case {
	err1 := `spec: Invalid value: "object": keyspace cannot be set if mode is StandBy`
	return []Case{
		{
			desc:     "no keyspace and no mode",
			isCreate: true,
			mode:     PatchModeMerge,
		},
		{
			desc: "update: no keyspace and no mode",
			mode: PatchModeMerge,
		},
		{
			desc:     "mode is Normal and no keyspace",
			isCreate: true,
			current:  map[string]any{"mode": "Normal"},
			mode:     PatchModeMerge,
		},
		{
			desc:    "update: mode is Normal and no keyspace",
			old:     map[string]any{"mode": "Normal"},
			current: map[string]any{"mode": "Normal"},
			mode:    PatchModeMerge,
		},
		{
			desc:     "mode is Normal and keyspace is empty",
			isCreate: true,
			current:  map[string]any{"mode": "Normal", "keyspace": ""},
			mode:     PatchModeMerge,
		},
		{
			desc:    "update: mode is Normal and keyspace is empty",
			old:     map[string]any{"mode": "Normal", "keyspace": ""},
			current: map[string]any{"mode": "Normal", "keyspace": ""},
			mode:    PatchModeMerge,
		},
		{
			desc:     "mode is Normal and keyspace is set",
			isCreate: true,
			current:  map[string]any{"mode": "Normal", "keyspace": "xxx"},
			mode:     PatchModeMerge,
		},
		{
			desc:    "update: mode is Normal and keyspace is set",
			old:     map[string]any{"mode": "Normal", "keyspace": "xxx"},
			current: map[string]any{"mode": "Normal", "keyspace": "xxx"},
			mode:    PatchModeMerge,
		},
		{
			desc:     "mode is StandBy and no keyspace",
			isCreate: true,
			current:  map[string]any{"mode": "StandBy"},
			mode:     PatchModeMerge,
		},
		{
			desc:    "update: mode is StandBy and no keyspace",
			old:     map[string]any{"mode": "StandBy"},
			current: map[string]any{"mode": "StandBy"},
			mode:    PatchModeMerge,
		},
		{
			desc:     "mode is StandBy and keyspace is empty",
			isCreate: true,
			current:  map[string]any{"mode": "StandBy", "keyspace": ""},
			mode:     PatchModeMerge,
		},
		{
			desc:    "update: mode is StandBy and keyspace is empty",
			old:     map[string]any{"mode": "StandBy", "keyspace": ""},
			current: map[string]any{"mode": "StandBy", "keyspace": ""},
			mode:    PatchModeMerge,
		},
		{
			desc:     "mode is StandBy and keyspace is set",
			isCreate: true,
			current:  map[string]any{"mode": "StandBy", "keyspace": "xxx"},
			mode:     PatchModeMerge,
			wantErrs: []string{
				err1,
			},
		},
		{
			desc:    "update: mode is StandBy and try to set keyspace directly",
			old:     map[string]any{"mode": "StandBy"},
			current: map[string]any{"mode": "StandBy", "keyspace": "xxx"},
			mode:    PatchModeMerge,
			wantErrs: []string{
				err1,
			},
		},
		{
			desc:    "update: mode is Standby and try to activate keyspace",
			old:     map[string]any{"mode": "StandBy"},
			current: map[string]any{"mode": "Normal", "keyspace": "yyy"},
			mode:    PatchModeMerge,
		},
		{
			desc:    "update: mode is Standby and try to activate keyspace from empty",
			old:     map[string]any{"mode": "StandBy", "keyspace": ""},
			current: map[string]any{"mode": "Normal", "keyspace": "yyy"},
			mode:    PatchModeMerge,
		},
	}
}

func basicTiDBGroup() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: TiDBGroup
metadata:
  name: tidb-group
spec:
  cluster:
    name: test
  replicas: 1
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

func transferTiDBGroupCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicTiDBGroup()
		c.current = Patch(t, c.mode, current, c.current, fields...)

		if c.isCreate {
			c.old = nil
			continue
		}

		old := basicTiDBGroup()
		c.old = Patch(t, c.mode, old, c.old, fields...)
	}

	return cases
}
