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

func TestTiFlash(t *testing.T) {
	cases := []Case{}
	cases = append(cases, transferTiFlashCases(t, Topology(), "spec", "topology")...)
	cases = append(cases, transferTiFlashCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases, transferTiFlashCases(t, PodOverlayLabels(), "spec", "overlay", "pod", "metadata")...)
	cases = append(cases, transferTiFlashCases(t, OverlayVolumeClaims(true), "spec")...)
	cases = append(cases, transferTiFlashCases(t, DataVolumeRequired(), "spec")...)
	cases = append(cases, transferTiFlashCases(t, VolumeAttributesClassNameValidation(), "spec", "volumes")...)
	cases = append(cases, transferTiFlashCases(t, Version(), "spec", "version")...)
	cases = append(cases, transferTiFlashCases(t, NameLength(instanceNameLengthLimit), "metadata", "name")...)
	Validate(t, "crd/core.pingcap.com_tiflashes.yaml", cases)
}

func TestTiFlashGroup(t *testing.T) {
	var cases []Case
	cases = append(cases, transferTiFlashGroupCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases, transferTiFlashGroupCases(t, NameLength(groupNameLengthLimit), "metadata", "name")...)
	cases = append(cases, transferTiFlashGroupCases(t, MinReadySeconds(), "spec", "minReadySeconds")...)
	Validate(t, "crd/core.pingcap.com_tiflashgroups.yaml", cases)
}

func basicTiFlash() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: TiFlash
metadata:
  name: tiflash
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

func transferTiFlashCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicTiFlash()
		c.current = Patch(t, c.mode, current, c.current, fields...)

		if c.isCreate {
			c.old = nil
			continue
		}

		old := basicTiFlash()
		c.old = Patch(t, c.mode, old, c.old, fields...)
	}

	return cases
}

func basicTiFlashGroup() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: TiFlashGroup
metadata:
  name: tiflash-group
spec:
  cluster:
    name: test
  replicas: 2
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

func transferTiFlashGroupCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicTiFlashGroup()
		c.current = Patch(t, c.mode, current, c.current, fields...)

		if c.isCreate {
			c.old = nil
			continue
		}

		old := basicTiFlashGroup()
		c.old = Patch(t, c.mode, old, c.old, fields...)
	}

	return cases
}
