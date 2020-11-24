// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package member

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func TestUpdateConfigMap(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name       string
		old        *corev1.ConfigMap
		new        *corev1.ConfigMap
		updateKeys []string
		err        error
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		origOld := test.old.DeepCopy()
		origNew := test.new.DeepCopy()
		err := updateConfigMap(test.old, test.new)

		if test.err != nil {
			g.Expect(err).To(Equal(test.err))
			return
		}
		g.Expect(err).NotTo(HaveOccurred())

		// different keys are updated in new
		if len(test.new.Data) > 0 {
			for _, k := range test.updateKeys {
				// only keys in both old and new are updated
				_, ok := origOld.Data[k]
				if !ok {
					continue
				}
				g.Expect(test.new.Data[k]).To(Equal(origOld.Data[k]))
			}
		}

		// other keys should not be modified
		for k, v := range origNew.Data {
			_, ok := origOld.Data[k]
			if ok {
				continue
			}
			g.Expect(test.new.Data[k]).To(Equal(v))
		}
	}

	tests := []testcase{
		{
			name: "two empty config map",
			old:  &corev1.ConfigMap{},
			new:  &corev1.ConfigMap{},
		},
		{
			name: "empty new config map",
			old: &corev1.ConfigMap{
				Data: map[string]string{
					"config-file": "foo",
				},
			},
			new:        &corev1.ConfigMap{},
			updateKeys: []string{"config-file"},
		},
		{
			name: "empty old config map",
			old:  &corev1.ConfigMap{},
			new: &corev1.ConfigMap{
				Data: map[string]string{
					"config-file": "bar",
				},
			},
			updateKeys: []string{"config-file"},
		},
		{
			name: "add new key",
			old: &corev1.ConfigMap{
				Data: map[string]string{
					"config-file":       "foo",
					"pump-config":       "a = \"b\"",
					"config_templ.toml": "c = \"d\"",
				},
			},
			new: &corev1.ConfigMap{
				Data: map[string]string{
					"pump-config":       "a = \"b\"",
					"config_templ.toml": "# comment \nc = \"d\"",
				},
			},
			updateKeys: []string{"pump-config", "config_templ.toml"},
		},
		{
			name: "some keys are not updated",
			old: &corev1.ConfigMap{
				Data: map[string]string{
					"config-file":       "foo",
					"pump-config":       "a = \"b\"",
					"config_templ.toml": "c = \"d\"",
				},
			},
			new: &corev1.ConfigMap{
				Data: map[string]string{
					"pump-config":       "a = \"b\"",
					"config_templ.toml": "# comment \nc = \"d\"",
					"proxy_templ.toml":  "e = \"f\"",
				},
			},
			updateKeys: []string{"pump-config", "config_templ.toml"},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}
