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
	gocmp "github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
)

func TestClientManager(t *testing.T) {
	cases := []struct {
		desc       string
		obj        client.Object
		updateFunc func(obj client.Object) client.Object
		changed    bool
	}{
		{
			desc: "not changed",
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			updateFunc: func(obj client.Object) client.Object { return obj },
			changed:    false,
		},
		{
			desc: "change ns",
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			updateFunc: func(obj client.Object) client.Object {
				obj.SetNamespace("test")
				return obj
			},
			changed: true,
		},
		{
			desc: "change uid",
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			updateFunc: func(obj client.Object) client.Object {
				obj.SetUID("xxxx")
				return obj
			},
			changed: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			count := 0
			cm := NewManagerBuilder[client.Object, int, int]().
				WithNewUnderlayClientFunc(func(client.Object) (int, error) {
					// add count for each time the underlay client is newed
					count++
					return count, nil
				}).
				WithCacheKeysFunc(func(obj client.Object) ([]string, error) {
					return []string{obj.GetName(), obj.GetNamespace(), string(obj.GetUID())}, nil
				}).
				WithNewClientFunc(func(obj client.Object, underlay int, _ SharedInformerFactory[int]) int {
					// check underlay client is newed by NewUnderlayClientFunc
					assert.Equal(tt, count, underlay)
					// key is equal with the primary key returned by cache keys
					assert.Equal(tt, c.obj, obj)
					return count
				}).
				Build()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cm.Start(ctx)

			// client is not registered
			_, ok := cm.Get(c.obj.GetName())
			assert.False(tt, ok)

			// client can be get after registered
			require.NoError(tt, cm.Register(c.obj))
			clientObj, ok := cm.Get(c.obj.GetName())
			assert.True(tt, ok)

			// update obj, client will be updated only when cache keys are changed
			updated := c.updateFunc(c.obj)
			require.NoError(tt, cm.Register(updated))
			updateClient, ok := cm.Get(c.obj.GetName())
			assert.True(tt, ok)

			if !c.changed {
				assert.Equal(tt, clientObj, updateClient)
			} else {
				assert.NotEqual(tt, clientObj, updateClient)
			}

			// Deregister obj
			cm.Deregister(updated.GetName())
			_, ok2 := cm.Get(updated.GetName())
			assert.False(tt, ok2)
		})
	}
}

func TestClientManagerSource(t *testing.T) {
	cases := []struct {
		desc     string
		previous []pdv1.Store
		updated  []pdv1.Store

		resyncPeriod time.Duration
		resyncTimes  int

		expectedCreateEvents []event.TypedCreateEvent[client.Object]
		expectedUpdateEvents []event.TypedUpdateEvent[client.Object]
		expectedDeleteEvents []event.TypedDeleteEvent[client.Object]
	}{
		{
			desc: "no update",
			previous: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				*fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
			},
			updated: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				*fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
			},
			expectedCreateEvents: []event.TypedCreateEvent[client.Object]{
				{
					Object: fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
				{
					Object: fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
			},
		},
		{
			desc: "add new obj",
			previous: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
			},
			updated: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				*fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
			},
			expectedCreateEvents: []event.TypedCreateEvent[client.Object]{
				{
					Object: fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
				{
					Object: fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
			},
		},
		{
			desc: "del existing obj",
			previous: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				*fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
			},
			updated: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
			},
			expectedCreateEvents: []event.TypedCreateEvent[client.Object]{
				{
					Object: fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
				{
					Object: fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
			},
			expectedDeleteEvents: []event.TypedDeleteEvent[client.Object]{
				{
					Object: fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
			},
		},
		{
			desc: "update existing obj",
			previous: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				*fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
			},
			updated: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				*fake.FakeObj("bb", func(obj *pdv1.Store) *pdv1.Store {
					obj.Labels = map[string]string{"test": "test"}
					obj.ResourceVersion = "bbb"
					return obj
				}),
			},
			expectedCreateEvents: []event.TypedCreateEvent[client.Object]{
				{
					Object: fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
				{
					Object: fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
			},
			expectedUpdateEvents: []event.TypedUpdateEvent[client.Object]{
				{
					ObjectOld: fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
					ObjectNew: fake.FakeObj("bb", func(obj *pdv1.Store) *pdv1.Store {
						obj.Labels = map[string]string{"test": "test"}
						obj.ResourceVersion = "bbb"
						return obj
					}),
				},
			},
			resyncPeriod: time.Second * 3,
			resyncTimes:  1,
		},
		{
			// Events after first resync will be lost if no rv
			// It's unexpected.
			desc: "update existing obj no rv [Unexpected]",
			previous: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				*fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
			},
			updated: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				*fake.FakeObj("bb", func(obj *pdv1.Store) *pdv1.Store {
					obj.Labels = map[string]string{"test": "test"}
					obj.ResourceVersion = "aaa"
					return obj
				}),
			},
			expectedCreateEvents: []event.TypedCreateEvent[client.Object]{
				{
					Object: fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
				{
					Object: fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
			},
			// NOTE: update event is lost
			expectedUpdateEvents: nil,
			resyncPeriod:         time.Second * 3,
			resyncTimes:          1,
		},
		{
			desc: "update existing obj no rv resync twice",
			previous: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				*fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
			},
			updated: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				*fake.FakeObj("bb", func(obj *pdv1.Store) *pdv1.Store {
					obj.Labels = map[string]string{"test": "test"}
					obj.ResourceVersion = "aaa"
					return obj
				}),
			},
			expectedCreateEvents: []event.TypedCreateEvent[client.Object]{
				{
					Object: fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
				{
					Object: fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
			},
			expectedUpdateEvents: []event.TypedUpdateEvent[client.Object]{
				{
					ObjectOld: fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
					ObjectNew: fake.FakeObj("bb", func(obj *pdv1.Store) *pdv1.Store {
						obj.Labels = map[string]string{"test": "test"}
						obj.ResourceVersion = "aaa"
						return obj
					}),
				},
			},
			resyncPeriod: time.Second * 3,
			resyncTimes:  2,
		},
		{
			desc: "update existing obj no rv no resync",
			previous: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				*fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
			},
			updated: []pdv1.Store{
				*fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				*fake.FakeObj("bb", func(obj *pdv1.Store) *pdv1.Store {
					obj.Labels = map[string]string{"test": "test"}
					obj.ResourceVersion = "aaa"
					return obj
				}),
			},
			expectedCreateEvents: []event.TypedCreateEvent[client.Object]{
				{
					Object: fake.FakeObj("aa", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
				{
					Object: fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
				},
			},
			expectedUpdateEvents: []event.TypedUpdateEvent[client.Object]{
				{
					ObjectOld: fake.FakeObj("bb", fake.ResourceVersion[pdv1.Store]("aaa")),
					ObjectNew: fake.FakeObj("bb", func(obj *pdv1.Store) *pdv1.Store {
						obj.Labels = map[string]string{"test": "test"}
						obj.ResourceVersion = "aaa"
						return obj
					}),
				},
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			lister := NewFakeLister(c.previous)
			cm := NewManagerBuilder[client.Object, int, int]().
				WithNewUnderlayClientFunc(func(client.Object) (int, error) {
					return 0, nil
				}).
				WithCacheKeysFunc(func(obj client.Object) ([]string, error) {
					return []string{obj.GetName()}, nil
				}).
				WithNewClientFunc(func(_ client.Object, _ int, f SharedInformerFactory[int]) int {
					f.InformerFor(&pdv1.Store{})
					return 0
				}).
				WithNewPollerFunc(&pdv1.Store{}, func(name string, logger logr.Logger, _ int) Poller {
<<<<<<< HEAD
					return NewPoller(name, logger, &lister, NewDeepEquality[pdv1.Store](), time.Millisecond*200)
=======
					return NewPoller(name, logger, lister, NewDeepEquality[pdv1.Store](logger), time.Millisecond*200)
>>>>>>> 7607c84ff (fix(timanager): fix lost event issue (#6572))
				}).
				WithResyncPeriod(c.resyncPeriod).
				Build()

			timeout := c.resyncPeriod * (time.Duration(c.resyncTimes + 1))
			if c.resyncPeriod == 0 {
				timeout = time.Second * 10
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			done := make(chan struct{})
			total := 0
			var createEvents []event.TypedCreateEvent[client.Object]
			var updateEvents []event.TypedUpdateEvent[client.Object]
			var deleteEvents []event.TypedDeleteEvent[client.Object]

			s := cm.Source(&pdv1.Store{}, handler.TypedFuncs[client.Object, reconcile.Request]{
				CreateFunc: func(_ context.Context, event event.TypedCreateEvent[client.Object], _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
					_, ok := event.Object.(*pdv1.Store)
					assert.True(tt, ok)

					createEvents = append(createEvents, event)

					total++
					if total == len(c.expectedCreateEvents)+len(c.expectedUpdateEvents)+len(c.expectedDeleteEvents) {
						close(done)
					}
				},

				UpdateFunc: func(_ context.Context, event event.TypedUpdateEvent[client.Object], _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
					// ignore no diff
					diff := gocmp.Diff(event.ObjectOld, event.ObjectNew)
					if diff == "" {
						return
					}

					_, ok1 := event.ObjectOld.(*pdv1.Store)
					assert.True(tt, ok1)

					_, ok2 := event.ObjectNew.(*pdv1.Store)
					assert.True(tt, ok2)

					updateEvents = append(updateEvents, event)

					total++
					if total == len(c.expectedCreateEvents)+len(c.expectedUpdateEvents)+len(c.expectedDeleteEvents) {
						close(done)
					}
				},
				DeleteFunc: func(_ context.Context, event event.TypedDeleteEvent[client.Object], _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
					_, ok := event.Object.(*pdv1.Store)
					assert.True(tt, ok)

					deleteEvents = append(deleteEvents, event)

					total++
					if total == len(c.expectedCreateEvents)+len(c.expectedUpdateEvents)+len(c.expectedDeleteEvents) {
						close(done)
					}
				},
			})

			cm.Start(ctx)
			assert.NoError(tt, s.Start(ctx, workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedItemBasedRateLimiter[reconcile.Request]())))

			assert.NoError(tt, cm.Register(fake.FakeObj[corev1.Pod]("aa")))

			es, ok := s.(EventSource)
			assert.True(tt, ok)

			synced := cache.WaitForCacheSync(ctx.Done(), func() bool {
				return es.HasSynced("aa")
			})
			assert.True(tt, synced)

			time.Sleep(c.resyncPeriod * time.Duration(c.resyncTimes))

			lister.UpdateItems(c.updated)

			select {
			case <-ctx.Done():
				assert.Fail(tt, "wait events timeout")
			case <-done:
			}

			slices.SortFunc(createEvents, func(a, b event.TypedCreateEvent[client.Object]) int {
				return cmp.Compare(a.Object.GetName(), b.Object.GetName())
			})
			slices.SortFunc(updateEvents, func(a, b event.TypedUpdateEvent[client.Object]) int {
				return cmp.Compare(a.ObjectNew.GetName(), b.ObjectNew.GetName())
			})
			slices.SortFunc(deleteEvents, func(a, b event.TypedDeleteEvent[client.Object]) int {
				return cmp.Compare(a.Object.GetName(), b.Object.GetName())
			})

			assert.Equal(tt, c.expectedCreateEvents, createEvents)
			assert.Equal(tt, c.expectedUpdateEvents, updateEvents)
			assert.Equal(tt, c.expectedDeleteEvents, deleteEvents)
		})
	}
}
