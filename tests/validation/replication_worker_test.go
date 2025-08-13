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

func TestReplicationWorker(t *testing.T) {
	var cases []Case
	cases = append(cases, transferReplicationWorkerCases(t, Topology(), "spec", "topology")...)
	cases = append(cases, transferReplicationWorkerCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases, transferReplicationWorkerCases(t, PodOverlayLabels(), "spec", "overlay", "pod", "metadata")...)
	cases = append(cases, transferReplicationWorkerCases(t, OverlayVolumeClaims(true), "spec")...)
	cases = append(cases, transferReplicationWorkerCases(t, DataVolumeRequired(), "spec")...)
	cases = append(cases, transferReplicationWorkerCases(t, Version(), "spec", "version")...)
	cases = append(cases, transferReplicationWorkerCases(t, NameLength(instanceNameLengthLimit), "metadata", "name")...)
	Validate(t, "crd/core.pingcap.com_replicationworkers.yaml", cases)
}

func TestReplicationWorkerGroup(t *testing.T) {
	var cases []Case
	cases = append(cases, transferReplicationWorkerGroupCases(t, NameLength(groupNameLengthLimit), "metadata", "name")...)
	Validate(t, "crd/core.pingcap.com_replicationworkergroups.yaml", cases)
}

func basicReplicationWorkerGroup() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: ReplicationWorkerGroup
metadata:
  name: rw-group
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

func basicReplicationWorker() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: ReplicationWorker
metadata:
  name: rw
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

func transferReplicationWorkerGroupCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicReplicationWorkerGroup()
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

		old := basicReplicationWorkerGroup()
		if c.old == nil {
			unstructured.RemoveNestedField(old, fields...)
		} else {
			require.NoError(t, unstructured.SetNestedField(old, c.old, fields...))
		}

		c.old = old
	}

	return cases
}

func transferReplicationWorkerCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicReplicationWorker()
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

		old := basicReplicationWorker()
		if c.old == nil {
			unstructured.RemoveNestedField(old, fields...)
		} else {
			require.NoError(t, unstructured.SetNestedField(old, c.old, fields...))
		}

		c.old = old
	}

	return cases
}
