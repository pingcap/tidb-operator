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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateConfigMap(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name       string
		old        *corev1.ConfigMap
		new        *corev1.ConfigMap
		updateKeys []string
		err        error
		equal      bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		origOld := test.old.DeepCopy()
		origNew := test.new.DeepCopy()
		equal, err := updateConfigMap(test.old, test.new)

		if test.err != nil {
			g.Expect(err).To(Equal(test.err))
			return
		}
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(equal).To(Equal(test.equal))

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
			name:  "two empty config map",
			old:   &corev1.ConfigMap{},
			new:   &corev1.ConfigMap{},
			equal: true,
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
			equal:      false,
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
			equal:      false,
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
			equal:      false,
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
			equal:      false,
		},
		{
			name: "the data of old and new configmaps are the same",
			old: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "old",
				},
				Data: map[string]string{
					"config-file": "a = \"b\"",
				},
			},
			new: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new",
				},
				Data: map[string]string{
					"config-file": "a = \"b\"",
				},
			},
			equal: true,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestConfirmNameByData(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name         string
		existing     *corev1.ConfigMap
		desired      *corev1.ConfigMap
		afterConfirm *corev1.ConfigMap
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		dataEqual, err := updateConfigMap(test.existing, test.desired)
		g.Expect(err).NotTo(HaveOccurred())

		confirmNameByData(test.existing, test.desired, dataEqual)
		g.Expect(test.afterConfirm).To(Equal(test.desired))
	}

	tests := []testcase{
		{
			name: "data is equal but existing cm name is not equal to desired cm name",
			existing: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing",
				},
				Data: map[string]string{
					"config-file": "a = \"b\"",
				},
			},
			desired: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "desired",
				},
				Data: map[string]string{
					"config-file": "a = \"b\"",
				},
			},
			afterConfirm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing",
				},
				Data: map[string]string{
					"config-file": "a = \"b\"",
				},
			},
		},
		{
			name: "data is not equal but existing cm name is equal to desired cm name",
			existing: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cm-12345",
				},
				Data: map[string]string{
					"config-file": "a = \"b\"",
				},
			},
			desired: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cm-12345",
				},
				Data: map[string]string{
					"config-file": "a = \"c\"",
				},
			},
			afterConfirm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cm-12345-new",
				},
				Data: map[string]string{
					"config-file": "a = \"c\"",
				},
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}
