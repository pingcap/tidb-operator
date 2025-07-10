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
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/pingcap/tidb-operator/pkg/client"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
)

const (
	bufSize = 100
)

// NewPollerFunc is the function to new a poller by underlay client
type NewPollerFunc[UnderlayClient any] func(name string, logger logr.Logger, c UnderlayClient) Poller

// NewUnderlayClientFunc is the func to new a underlay client, for example, pdapi.PDClient
type NewUnderlayClientFunc[Object client.Object, UnderlayClient any] func(obj Object) (UnderlayClient, error)

// NewClientFunc is the func to new an external client with the cache layer, for example, timanager.PDClient
type NewClientFunc[Object client.Object, UnderlayClient, Client any] func(
	Object, UnderlayClient, SharedInformerFactory[UnderlayClient]) Client

type CacheKeysFunc[Object client.Object] func(obj Object) ([]string, error)

type Manager[Object client.Object, Client any] interface {
	Register(obj Object) error
	Deregister(key string)
	Get(key string) (Client, bool)

	Source(obj runtime.Object, h handler.EventHandler) source.Source
	Start(ctx context.Context)
}

type ManagerBuilder[Object client.Object, UnderlayClient, Client any] interface {
	WithLogger(logger logr.Logger) ManagerBuilder[Object, UnderlayClient, Client]
	WithCacheKeysFunc(f CacheKeysFunc[Object]) ManagerBuilder[Object, UnderlayClient, Client]
	WithNewUnderlayClientFunc(f NewUnderlayClientFunc[Object, UnderlayClient]) ManagerBuilder[Object, UnderlayClient, Client]
	WithNewClientFunc(f NewClientFunc[Object, UnderlayClient, Client]) ManagerBuilder[Object, UnderlayClient, Client]
	WithNewPollerFunc(obj runtime.Object, f NewPollerFunc[UnderlayClient]) ManagerBuilder[Object, UnderlayClient, Client]
	Build() Manager[Object, Client]
}

type builder[Object client.Object, UnderlayClient, Client any] struct {
	logger                logr.Logger
	newUnderlayClientFunc NewUnderlayClientFunc[Object, UnderlayClient]
	newClientFunc         NewClientFunc[Object, UnderlayClient, Client]
	cacheKeysFunc         CacheKeysFunc[Object]

	newPollerFuncMap map[reflect.Type]NewPollerFunc[UnderlayClient]
}

func NewManagerBuilder[Object client.Object, UnderlayClient, Client any]() ManagerBuilder[Object, UnderlayClient, Client] {
	return &builder[Object, UnderlayClient, Client]{
		newPollerFuncMap: map[reflect.Type]NewPollerFunc[UnderlayClient]{},
	}
}

func (b *builder[Object, UnderlayClient, Client]) WithLogger(logger logr.Logger) ManagerBuilder[Object, UnderlayClient, Client] {
	b.logger = logger
	return b
}

func (b *builder[Object, UnderlayClient, Client]) WithCacheKeysFunc(
	f CacheKeysFunc[Object],
) ManagerBuilder[Object, UnderlayClient, Client] {
	b.cacheKeysFunc = f
	return b
}

func (b *builder[Object, UnderlayClient, Client]) WithNewUnderlayClientFunc(
	f NewUnderlayClientFunc[Object, UnderlayClient],
) ManagerBuilder[Object, UnderlayClient, Client] {
	b.newUnderlayClientFunc = f
	return b
}

func (b *builder[Object, UnderlayClient, Client]) WithNewClientFunc(
	f NewClientFunc[Object, UnderlayClient, Client],
) ManagerBuilder[Object, UnderlayClient, Client] {
	b.newClientFunc = f
	return b
}

func (b *builder[Object, UnderlayClient, Client]) WithNewPollerFunc(
	obj runtime.Object, f NewPollerFunc[UnderlayClient],
) ManagerBuilder[Object, UnderlayClient, Client] {
	t := reflect.TypeOf(obj)
	b.newPollerFuncMap[t] = f

	return b
}

func (b *builder[Object, UnderlayClient, Client]) Build() Manager[Object, Client] {
	s := runtime.NewScheme()
	if err := pdv1.Install(s); err != nil {
		panic(err)
	}

	return &clientManager[Object, UnderlayClient, Client]{
		logger:                b.logger,
		scheme:                s,
		newUnderlayClientFunc: b.newUnderlayClientFunc,
		newClientFunc:         b.newClientFunc,
		cacheKeysFunc:         b.cacheKeysFunc,
		newPollerFuncMap:      b.newPollerFuncMap,
		sources:               map[reflect.Type][]EventSource{},
	}
}

type clientManager[Object client.Object, UnderlayClient, Client any] struct {
	logger logr.Logger
	scheme *runtime.Scheme

	newUnderlayClientFunc NewUnderlayClientFunc[Object, UnderlayClient]
	newClientFunc         NewClientFunc[Object, UnderlayClient, Client]
	cacheKeysFunc         CacheKeysFunc[Object]

	cs maputil.Map[string, Cache[Client, UnderlayClient]]

	newPollerFuncMap map[reflect.Type]NewPollerFunc[UnderlayClient]
	sources          map[reflect.Type][]EventSource

	ctx     context.Context
	started bool
}

func (m *clientManager[Object, UnderlayClient, Client]) Register(obj Object) error {
	keys, err := m.cacheKeysFunc(obj)
	if err != nil {
		return err
	}
	cacheObj, ok := m.cs.Load(keys[0])
	if ok {
		if reflect.DeepEqual(keys, cacheObj.Keys()) {
			m.logger.Info("hit client cache", "obj", client.ObjectKeyFromObject(obj))
			return nil
		}

		m.logger.Info("renew client", "obj", client.ObjectKeyFromObject(obj))

		if m.cs.CompareAndDelete(keys[0], cacheObj) {
			cacheObj.Stop()
		}
	} else {
		m.logger.Info("register client", "obj", client.ObjectKeyFromObject(obj))
	}

	underlay, err := m.newUnderlayClientFunc(obj)
	if err != nil {
		return err
	}

	f := NewSharedInformerFactory(keys[0], m.logger, m.scheme, underlay, m.newPollerFuncMap, time.Hour)
	c := m.newClientFunc(obj, underlay, f)

	cacheObj = NewCache[Client, UnderlayClient](keys, c, f)
	m.cs.Store(keys[0], cacheObj)
	go func() {
		cacheObj.Start(m.ctx)
		f.WaitForCacheSync(m.ctx.Done())
		if err := f.ForEach(func(t reflect.Type, informer cache.SharedIndexInformer) error {
			ss := m.sources[t]
			for _, s := range ss {
				if err := s.For(keys[0], informer); err != nil {
					return fmt.Errorf("cannot add event handler for %v:%v: %w", keys[0], t, err)
				}
			}

			return nil
		}); err != nil {
			m.logger.Error(err, "failed to add event handler")
		}
	}()
	return nil
}

func (m *clientManager[Object, UnderlayClient, Client]) Get(primaryKey string) (c Client, ok bool) {
	cacheObj, found := m.cs.Load(primaryKey)
	if !found {
		return
	}

	return cacheObj.Client(), true
}

func (m *clientManager[Object, UnderlayClient, Client]) Deregister(primaryKey string) {
	m.logger.Info("deregister client", "key", primaryKey)
	c, ok := m.cs.LoadAndDelete(primaryKey)
	if ok {
		c.Stop()
		m.logger.Info("client is successfully stopped", "key", primaryKey)
	}
}

func (m *clientManager[Object, UnderlayClient, Client]) InformerFor(key string, obj runtime.Object) cache.SharedIndexInformer {
	c, ok := m.cs.Load(key)
	if !ok {
		return nil
	}

	return c.InformerFactory().InformerFor(obj)
}

func (m *clientManager[Object, UnderlayClient, Client]) Source(obj runtime.Object, h handler.EventHandler) source.Source {
	if m.started {
		// TODO: optimize the panic
		panic("cannot add source after manager is started")
	}
	t := reflect.TypeOf(obj)
	_, ok := m.newPollerFuncMap[t]
	if !ok {
		// TODO: optimize the panic
		panic("cannot get source of type " + t.Name())
	}

	s := NewEventSource(h)
	ss := m.sources[t]
	ss = append(ss, s)

	m.sources[t] = ss

	return s
}

func (m *clientManager[Object, UnderlayClient, Client]) Start(ctx context.Context) {
	m.started = true
	m.ctx = ctx
}

type EventSource interface {
	source.Source
	For(key string, f cache.SharedIndexInformer) error
	HasSynced(key string) bool
}

type eventSource struct {
	h handler.EventHandler

	ctx context.Context
	rh  cache.ResourceEventHandler

	synced maputil.Map[string, cache.InformerSynced]
}

func (s *eventSource) Start(ctx context.Context, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) error {
	s.ctx = ctx
	s.rh = NewResourceEventHandler(s.ctx, s.h, queue)

	return nil
}

func NewEventSource(h handler.EventHandler) EventSource {
	return &eventSource{
		h: h,
	}
}

func (s *eventSource) For(key string, f cache.SharedIndexInformer) error {
	res, err := f.AddEventHandler(s.rh)
	s.synced.Store(key, res.HasSynced)
	return err
}

func (s *eventSource) HasSynced(key string) bool {
	hasSynced, ok := s.synced.Load(key)
	if !ok {
		return false
	}
	return hasSynced()
}

// NewResourceEventHandler creates a ResourceEventHandler.
// See "sigs.k8s.io/controller-runtime/pkg/internal/source.NewEventHandler"
func NewResourceEventHandler[O client.Object, R comparable](
	ctx context.Context, h handler.TypedEventHandler[O, R],
	q workqueue.TypedRateLimitingInterface[R],
) cache.ResourceEventHandler {
	logger := logr.FromContextOrDiscard(ctx)

	return cache.ResourceEventHandlerDetailedFuncs{
		AddFunc: func(obj any, _ bool) {
			e := event.TypedCreateEvent[O]{}

			if o, ok := obj.(O); ok {
				e.Object = o
			} else {
				logger.Error(nil, "OnAdd missing Object",
					"object", obj, "type", fmt.Sprintf("%T", obj))
				return
			}
			h.Create(ctx, e, q)
		},
		UpdateFunc: func(oldObj, newObj any) {
			e := event.TypedUpdateEvent[O]{}

			if o, ok := oldObj.(O); ok {
				e.ObjectOld = o
			} else {
				logger.Error(nil, "OnUpdate missing ObjectOld",
					"object", oldObj, "type", fmt.Sprintf("%T", oldObj))
				return
			}

			if o, ok := newObj.(O); ok {
				e.ObjectNew = o
			} else {
				logger.Error(nil, "OnUpdate missing ObjectNew",
					"object", newObj, "type", fmt.Sprintf("%T", newObj))
				return
			}
			h.Update(ctx, e, q)
		},
		DeleteFunc: func(obj any) {
			e := event.TypedDeleteEvent[O]{}

			var ok bool
			if _, ok = obj.(client.Object); !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					logger.Error(nil, "Error decoding objects.  Expected cache.DeletedFinalStateUnknown",
						"type", fmt.Sprintf("%T", obj),
						"object", obj)
					return
				}

				e.DeleteStateUnknown = true

				obj = tombstone.Obj
			}

			if o, ok := obj.(O); ok {
				e.Object = o
			} else {
				logger.Error(nil, "OnDelete missing Object",
					"object", obj, "type", fmt.Sprintf("%T", obj))
				return
			}

			h.Delete(ctx, e, q)
		},
	}
}
