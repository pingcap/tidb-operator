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

package timanager

import (
	"cmp"
	"context"
	"slices"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

type FakeLister[T any, PT Object[T]] struct {
	L List[T, PT]
}

func (l *FakeLister[T, PT]) List(_ context.Context) (*List[T, PT], error) {
	return &l.L, nil
}

func (l *FakeLister[T, PT]) GetItems(_ *List[T, PT]) []PT {
	objs := make([]PT, 0, len(l.L.Items))
	for i := range l.L.Items {
		objs = append(objs, &l.L.Items[i])
	}
	return objs
}

func (*FakeLister[T, PT]) MarkAsInvalid(PT) bool {
	return false
}

func TestPoller(t *testing.T) {
	cases := []struct {
		desc     string
		previous []corev1.Pod
		updated  []corev1.Pod

		expectedList   runtime.Object
		expectedEvents []watch.Event
	}{
		{
			desc: "no update",
			previous: []corev1.Pod{
				*fake.FakeObj[corev1.Pod]("aa"),
				*fake.FakeObj[corev1.Pod]("bb"),
			},
			updated: []corev1.Pod{
				*fake.FakeObj[corev1.Pod]("aa"),
				*fake.FakeObj[corev1.Pod]("bb"),
			},

			expectedList: &List[corev1.Pod, *corev1.Pod]{
				Items: []corev1.Pod{
					*fake.FakeObj[corev1.Pod]("aa"),
					*fake.FakeObj[corev1.Pod]("bb"),
				},
			},
			expectedEvents: []watch.Event{},
		},
		{
			desc: "add new obj",
			previous: []corev1.Pod{
				*fake.FakeObj[corev1.Pod]("aa"),
			},
			updated: []corev1.Pod{
				*fake.FakeObj[corev1.Pod]("aa"),
				*fake.FakeObj[corev1.Pod]("bb"),
			},

			expectedList: &List[corev1.Pod, *corev1.Pod]{
				Items: []corev1.Pod{
					*fake.FakeObj[corev1.Pod]("aa"),
				},
			},
			expectedEvents: []watch.Event{
				{
					Type:   watch.Added,
					Object: fake.FakeObj[corev1.Pod]("bb"),
				},
			},
		},
		{
			desc: "del existing obj",
			previous: []corev1.Pod{
				*fake.FakeObj[corev1.Pod]("aa"),
				*fake.FakeObj[corev1.Pod]("bb"),
			},
			updated: []corev1.Pod{
				*fake.FakeObj[corev1.Pod]("aa"),
			},

			expectedList: &List[corev1.Pod, *corev1.Pod]{
				Items: []corev1.Pod{
					*fake.FakeObj[corev1.Pod]("aa"),
					*fake.FakeObj[corev1.Pod]("bb"),
				},
			},
			expectedEvents: []watch.Event{
				{
					Type:   watch.Deleted,
					Object: fake.FakeObj[corev1.Pod]("bb"),
				},
			},
		},
		{
			desc: "update existing obj",
			previous: []corev1.Pod{
				*fake.FakeObj[corev1.Pod]("aa"),
				*fake.FakeObj[corev1.Pod]("bb"),
			},
			updated: []corev1.Pod{
				*fake.FakeObj[corev1.Pod]("aa"),
				*fake.FakeObj("bb", func(obj *corev1.Pod) *corev1.Pod {
					obj.Labels = map[string]string{"test": "test"}
					return obj
				}),
			},

			expectedList: &List[corev1.Pod, *corev1.Pod]{
				Items: []corev1.Pod{
					*fake.FakeObj[corev1.Pod]("aa"),
					*fake.FakeObj[corev1.Pod]("bb"),
				},
			},
			expectedEvents: []watch.Event{
				{
					Type: watch.Modified,
					Object: fake.FakeObj("bb", func(obj *corev1.Pod) *corev1.Pod {
						obj.Labels = map[string]string{"test": "test"}
						return obj
					}),
				},
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			lister := FakeLister[corev1.Pod, *corev1.Pod]{
				L: List[corev1.Pod, *corev1.Pod]{
					Items: c.previous,
				},
			}

			p := NewPoller(c.desc, logr.Discard(), &lister, NewDeepEquality[corev1.Pod](), time.Millisecond*500)

			ctx, cancel := context.WithCancel(context.Background())
			list, err := p.Sync(ctx)
			require.NoError(tt, err)

			assert.Equal(tt, c.expectedList, list)

			lister.L.Items = c.updated

			events := []watch.Event{}
			ch := make(chan watch.Event, bufSize)

			go p.Run(ctx, ch)

			waited := false
			func() {
				for {
					select {
					case event := <-ch:
						events = append(events, event)
					default:
						if waited {
							return
						}
						// sleep at least 2 * interval
						time.Sleep(time.Second)
						waited = true
					}
				}
			}()

			cancel()
			close(ch)

			slices.SortFunc(c.expectedEvents, CompareEvent)
			slices.SortFunc(events, CompareEvent)
			assert.Equal(tt, c.expectedEvents, events)
		})
	}
}

func CompareObject(a, b runtime.Object) int {
	aname := a.(client.Object).GetName()
	bname := b.(client.Object).GetName()

	return cmp.Compare(aname, bname)
}

func CompareEvent(a, b watch.Event) int {
	aname := a.Object.(client.Object).GetName()
	bname := b.Object.(client.Object).GetName()

	if aname == bname {
		return cmp.Compare(a.Type, b.Type)
	}

	return cmp.Compare(aname, bname)
}
