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

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource(t *testing.T) {
	cases := []struct {
		desc         string
		ns           NamespaceOption
		name         NameOption
		obj          int
		expectedNs   string
		expectedName string
		expectedObj  int
	}{
		{
			desc:         "normal",
			ns:           Namespace("aaa"),
			name:         Name("bbb"),
			obj:          42,
			expectedNs:   "aaa",
			expectedName: "bbb",
			expectedObj:  42,
		},
		{
			desc:         "use name func",
			ns:           NameFunc(func() string { return "aaa" }),
			name:         NameFunc(func() string { return "bbb" }),
			obj:          42,
			expectedNs:   "aaa",
			expectedName: "bbb",
			expectedObj:  42,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			var obj int
			r := NewResource(func(t int) {
				obj = t
			}).
				WithNamespace(c.ns).
				WithName(c.name).
				Initializer()

			r.Set(c.obj)

			assert.Equal(tt, c.expectedNs, r.Namespace(), c.desc)
			assert.Equal(tt, c.expectedName, r.Name(), c.desc)
			assert.Equal(tt, c.expectedObj, obj, c.desc)
		})
	}
}

func TestResourceSlice(t *testing.T) {
	cases := []struct {
		desc           string
		ns             NamespaceOption
		labels         LabelsOption
		objs           []int
		expectedNs     string
		expectedLabels map[string]string
		expectedObjs   []int
	}{
		{
			desc:       "normal",
			ns:         Namespace("aaa"),
			labels:     Labels(map[string]string{"xxx": "yyy"}),
			objs:       []int{42},
			expectedNs: "aaa",
			expectedLabels: map[string]string{
				"xxx": "yyy",
			},
			expectedObjs: []int{42},
		},
		{
			desc:       "use func",
			ns:         NameFunc(func() string { return "aaa" }),
			labels:     LabelsFunc(func() map[string]string { return map[string]string{"xxx": "yyy"} }),
			objs:       []int{42},
			expectedNs: "aaa",
			expectedLabels: map[string]string{
				"xxx": "yyy",
			},
			expectedObjs: []int{42},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			var objs []int
			r := NewResourceSlice(func(t []int) {
				objs = t
			}).
				WithNamespace(c.ns).
				WithLabels(c.labels).
				Initializer()

			r.Set(c.objs)

			assert.Equal(tt, c.expectedNs, r.Namespace(), c.desc)
			assert.Equal(tt, c.expectedLabels, r.Labels(), c.desc)
			assert.Equal(tt, c.expectedObjs, objs, c.desc)
		})
	}
}
