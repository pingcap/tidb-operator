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

package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func TestGet(t *testing.T) {
	cases := []struct {
		desc     string
		objs     []client.Object
		obj      client.Object
		expected client.Object
		hasErr   bool
	}{
		{
			desc: "get new obj",
			objs: []client.Object{
				fake.FakeObj(
					"aa",
					fake.Label[corev1.Pod]("test", "test"),
				),
			},
			// without label
			obj: fake.FakeObj[corev1.Pod]("aa"),
			expected: fake.FakeObj(
				"aa",
				fake.Label[corev1.Pod]("test", "test"),
			),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			p := NewFakeClient(c.objs...)
			err := p.Get(context.TODO(), ObjectKeyFromObject(c.obj), c.obj)
			if c.hasErr {
				assert.Error(tt, err)
			} else {
				require.NoError(tt, err)
				assert.Equal(tt, c.expected, c.obj)
			}
		})
	}
}

func TestList(t *testing.T) {
	cases := []struct {
		desc     string
		objs     []client.Object
		opts     []client.ListOption
		obj      client.ObjectList
		expected client.ObjectList
		hasErr   bool
	}{
		{
			desc: "list objs",
			objs: []client.Object{
				fake.FakeObj("aa", fake.Label[corev1.Pod]("test", "test")),
				fake.FakeObj("cc", fake.Label[corev1.Pod]("test", "test")),
				fake.FakeObj("bb", fake.Label[corev1.Pod]("test", "test")),
			},
			obj: &corev1.PodList{},
			// without label
			expected: &corev1.PodList{
				Items: []corev1.Pod{
					*fake.FakeObj("aa", fake.Label[corev1.Pod]("test", "test")),
					*fake.FakeObj("bb", fake.Label[corev1.Pod]("test", "test")),
					*fake.FakeObj("cc", fake.Label[corev1.Pod]("test", "test")),
				},
			},
		},
		{
			desc: "list objs with label selector",
			objs: []client.Object{
				fake.FakeObj("aa", fake.Label[corev1.Pod]("test", "test")),
				fake.FakeObj("cc", fake.Label[corev1.Pod]("test", "test")),
				fake.FakeObj("bb", fake.Label[corev1.Pod]("test", "not-selected")),
			},
			opts: []client.ListOption{
				client.MatchingLabels{"test": "test"},
			},
			obj: &corev1.PodList{},
			// without label
			expected: &corev1.PodList{
				Items: []corev1.Pod{
					*fake.FakeObj("aa", fake.Label[corev1.Pod]("test", "test")),
					*fake.FakeObj("cc", fake.Label[corev1.Pod]("test", "test")),
				},
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			p := NewFakeClient(c.objs...)

			err := p.List(context.TODO(), c.obj, c.opts...)
			if c.hasErr {
				assert.Error(tt, err)
			} else {
				require.NoError(tt, err)
				assert.Equal(tt, c.expected, c.obj)
			}
		})
	}
}
