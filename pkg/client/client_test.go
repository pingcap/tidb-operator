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

	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
)

func TestApply(t *testing.T) {
	cases := []struct {
		desc      string
		objs      []client.Object
		obj       client.Object
		expected  client.Object
		res       ApplyResult
		immutable []string
		hasErr    bool
	}{
		{
			desc: "apply a new obj",
			objs: []client.Object{
				fake.FakeObj[corev1.Pod]("aa"),
			},
			obj:      fake.FakeObj[corev1.Pod]("bb"),
			expected: fake.FakeObj("bb", fake.GVK[corev1.Pod](corev1.SchemeGroupVersion)),
			res:      ApplyResultCreated,
		},
		{
			desc: "add label for an existing obj",
			objs: []client.Object{
				fake.FakeObj[corev1.Pod]("aa"),
			},
			obj:      fake.FakeObj("aa", fake.Label[corev1.Pod]("test", "test")),
			expected: fake.FakeObj("aa", fake.GVK[corev1.Pod](corev1.SchemeGroupVersion), fake.Label[corev1.Pod]("test", "test")),
			res:      ApplyResultUpdated,
		},
		{
			desc: "apply again for an existing obj",
			objs: []client.Object{
				fake.FakeObj("aa", fake.Label[corev1.Pod]("test", "test")),
			},
			obj:      fake.FakeObj("aa", fake.Label[corev1.Pod]("test", "test")),
			expected: fake.FakeObj("aa", fake.GVK[corev1.Pod](corev1.SchemeGroupVersion), fake.Label[corev1.Pod]("test", "test")),
			res:      ApplyResultUnchanged,
		},
		{
			desc: "apply for an existing obj with immutable fields",
			objs: []client.Object{
				fake.FakeObj("aa", fake.Label[corev1.Pod]("test", "test"), func(obj *corev1.Pod) *corev1.Pod {
					obj.Spec.NodeName = "xxx"
					return obj
				}),
			},
			obj: fake.FakeObj("aa", fake.Label[corev1.Pod]("test", "test"), func(obj *corev1.Pod) *corev1.Pod {
				// nodeName is immutable
				obj.Spec.NodeName = "yyy"
				return obj
			}),
			expected: fake.FakeObj("aa", fake.GVK[corev1.Pod](corev1.SchemeGroupVersion), fake.Label[corev1.Pod]("test", "test"), func(obj *corev1.Pod) *corev1.Pod {
				obj.Spec.NodeName = "xxx"
				return obj
			}),
			immutable: []string{"spec", "nodeName"},
			res:       ApplyResultUnchanged,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			p := NewFakeClient()
			for _, obj := range c.objs {
				err := p.Apply(context.TODO(), obj, Immutable(c.immutable...))
				require.NoError(tt, err)
			}
			res, err := p.ApplyWithResult(context.TODO(), c.obj, Immutable(c.immutable...))
			if c.hasErr {
				assert.Error(tt, err)
			} else {
				require.NoError(tt, err)
				assert.Equal(tt, c.res, res)

				c.obj.SetManagedFields(nil)
				assert.Equal(tt, c.expected, c.obj)
			}
		})
	}
}
