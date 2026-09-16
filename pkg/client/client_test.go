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
		{
			desc: "apply for an existing obj with ignore diff annotation",
			objs: []client.Object{
				fake.FakeObj("aa", fake.Label[corev1.Pod]("test", "test"), func(obj *corev1.Pod) *corev1.Pod {
					obj.Annotations = map[string]string{AnnoKeyIgnoreDiff: "spec.nodeName,spec.dnsPolicy"}

					obj.Spec.NodeName = "xxx"
					obj.Spec.DNSPolicy = corev1.DNSDefault
					return obj
				}),
			},
			obj: fake.FakeObj("aa", fake.Label[corev1.Pod]("test", "test"), func(obj *corev1.Pod) *corev1.Pod {
				obj.Annotations = map[string]string{AnnoKeyIgnoreDiff: "spec.nodeName,spec.dnsPolicy"}
				// they are immutable
				obj.Spec.NodeName = "yyy"
				obj.Spec.DNSPolicy = corev1.DNSClusterFirst
				return obj
			}),
			expected: fake.FakeObj("aa", fake.GVK[corev1.Pod](corev1.SchemeGroupVersion), fake.Label[corev1.Pod]("test", "test"), func(obj *corev1.Pod) *corev1.Pod {
				obj.Annotations = map[string]string{AnnoKeyIgnoreDiff: "spec.nodeName,spec.dnsPolicy"}

				obj.Spec.NodeName = "xxx"
				obj.Spec.DNSPolicy = corev1.DNSDefault
				return obj
			}),
			res: ApplyResultUnchanged,
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

func TestApplyTransformers(t *testing.T) {
	for _, exists := range []bool{false, true} {
		name := "create"
		if exists {
			name = "update"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			desired := fake.FakeObj[corev1.Pod]("transformed")
			desired.Spec.NodeName = "new-node"
			cli := NewFakeClient()
			if exists {
				current := desired.DeepCopy()
				current.Spec.NodeName = "old-node"
				current.Status.Phase = corev1.PodRunning
				cli = NewFakeClient(current)
			}
			var seenCurrent client.Object
			var calls []string
			first := TransformerFunc(func(current, expected client.Object) client.Object {
				calls = append(calls, "first")
				require.NotSame(t, expected, client.Object(desired))
				if current != nil {
					require.Same(t, client.Object(desired), current)
				}
				seenCurrent = current
				if exists {
					require.Equal(t, corev1.PodRunning, current.(*corev1.Pod).Status.Phase)
					require.Equal(t, "old-node", current.(*corev1.Pod).Spec.NodeName)
				} else {
					require.Nil(t, current)
				}
				require.Equal(t, "new-node", expected.(*corev1.Pod).Spec.NodeName)
				// Return a replacement to verify the next transformer receives the result.
				replacement := expected.(*corev1.Pod).DeepCopy()
				replacement.Labels = map[string]string{"first": "true"}
				return replacement
			})
			second := TransformerFunc(func(current, expected client.Object) client.Object {
				calls = append(calls, "second")
				require.Equal(t, seenCurrent, current)
				require.Equal(t, "true", expected.GetLabels()["first"])
				expected.GetLabels()["second"] = "true"
				return expected
			})
			third := TransformerFunc(func(current, expected client.Object) client.Object {
				calls = append(calls, "third")
				require.Equal(t, "true", expected.GetLabels()["second"])
				if current != nil {
					require.Empty(t, current.GetLabels())
				}
				return expected
			})
			res, err := cli.ApplyWithResult(ctx, desired,
				Transformers(first), Transformers(second, third), Immutable("spec", "nodeName"))
			require.NoError(t, err)
			require.Equal(t, []string{"first", "second", "third"}, calls)
			require.Equal(t, map[string]string{"first": "true", "second": "true"}, desired.Labels)
			if exists {
				require.Equal(t, ApplyResultUpdated, res)
				require.Equal(t, "old-node", desired.Spec.NodeName)
			} else {
				require.Equal(t, ApplyResultCreated, res)
				require.Equal(t, "new-node", desired.Spec.NodeName)
			}
		})
	}
}

func TestApplyUnchangedReturnsCurrentStatus(t *testing.T) {
	ctx := context.Background()
	cli := NewFakeClient()
	obj := fake.FakeObj("unchanged", fake.Label[corev1.Pod]("test", "test"))
	require.NoError(t, cli.Apply(ctx, obj))
	// Set status after the initial Apply, as a controller would through the status subresource.
	obj.Status.Phase = corev1.PodRunning
	require.NoError(t, cli.Status().Update(ctx, obj))
	expected := fake.FakeObj("unchanged", fake.Label[corev1.Pod]("test", "test"))
	result, err := cli.ApplyWithResult(ctx, expected)
	require.NoError(t, err)
	require.Equal(t, ApplyResultUnchanged, result)
	require.Equal(t, corev1.PodRunning, expected.Status.Phase)
}
