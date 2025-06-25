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
	"reflect"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// SharedInformerFactory is modified from k8s.io/client-go/informers/factory.go
// to support poll from tidb clusters.
type SharedInformerFactory[UnderlayClient any] interface {
	Start(stopCh <-chan struct{})
	Shutdown()
	WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool
	InformerFor(obj runtime.Object) cache.SharedIndexInformer

	// call function for each registered informer
	ForEach(func(t reflect.Type, informer cache.SharedIndexInformer) error) error

	// Refresh will poll once immediately
	Refresh(obj runtime.Object)
}

func NewSharedInformerFactory[UnderlayClient any](
	name string,
	logger logr.Logger,
	scheme *runtime.Scheme,
	c UnderlayClient,
	newPollerFuncMap map[reflect.Type]NewPollerFunc[UnderlayClient],
	resyncPeriod time.Duration,
) SharedInformerFactory[UnderlayClient] {
	return &factory[UnderlayClient]{
		logger:           logger,
		name:             name,
		resyncPeriod:     resyncPeriod,
		pollers:          map[reflect.Type]Poller{},
		informers:        map[reflect.Type]cache.SharedIndexInformer{},
		startedInformers: map[reflect.Type]bool{},
		scheme:           scheme,
		c:                c,
		newPollerFuncMap: newPollerFuncMap,
	}
}

type factory[UnderlayClient any] struct {
	logger logr.Logger

	name string

	c UnderlayClient

	lock sync.Mutex

	wg sync.WaitGroup

	shuttingDown bool

	resyncPeriod time.Duration

	pollers map[reflect.Type]Poller

	informers        map[reflect.Type]cache.SharedIndexInformer
	startedInformers map[reflect.Type]bool

	scheme *runtime.Scheme

	newPollerFuncMap map[reflect.Type]NewPollerFunc[UnderlayClient]
}

func (f *factory[UnderlayClient]) Start(stopCh <-chan struct{}) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.shuttingDown {
		return
	}

	for informerType, informer := range f.informers {
		if !f.startedInformers[informerType] {
			f.wg.Add(1)
			// We need a new variable in each loop iteration,
			// otherwise the goroutine would use the loop variable
			// and that keeps changing.
			informer := informer
			go func() {
				defer f.wg.Done()
				informer.Run(stopCh)
			}()
			f.startedInformers[informerType] = true
		}
	}
}

func (f *factory[UnderlayClient]) Shutdown() {
	f.lock.Lock()
	f.shuttingDown = true
	f.lock.Unlock()

	// Will return immediately if there is nothing to wait for.
	f.wg.Wait()
}

func (f *factory[UnderlayClient]) WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool {
	informers := func() map[reflect.Type]cache.SharedIndexInformer {
		f.lock.Lock()
		defer f.lock.Unlock()

		informers := map[reflect.Type]cache.SharedIndexInformer{}
		for informerType, informer := range f.informers {
			if f.startedInformers[informerType] {
				informers[informerType] = informer
			}
		}
		return informers
	}()

	res := map[reflect.Type]bool{}
	for informType, informer := range informers {
		res[informType] = cache.WaitForCacheSync(stopCh, informer.HasSynced)
	}
	return res
}

func (f *factory[UnderlayClient]) InformerFor(obj runtime.Object) cache.SharedIndexInformer {
	f.lock.Lock()
	defer f.lock.Unlock()

	informerType := reflect.TypeOf(obj)
	informer, exists := f.informers[informerType]
	if exists {
		return informer
	}

	p, ok := f.pollers[informerType]
	if !ok {
		newPollerFunc, ok2 := f.newPollerFuncMap[informerType]
		if !ok2 {
			// TODO: fix it
			panic("unrecognized type")
		}
		p = newPollerFunc(f.name, f.logger, f.c)
		f.pollers[informerType] = p
	}

	lw := NewListerWatcher[UnderlayClient](f.logger, p)

	informer = cache.NewSharedIndexInformer(
		lw,
		obj,
		f.resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	f.informers[informerType] = informer

	return informer
}

func (f *factory[UnderlayClient]) ForEach(forEach func(t reflect.Type, informer cache.SharedIndexInformer) error) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	for t, informer := range f.informers {
		if err := forEach(t, informer); err != nil {
			return err
		}
	}

	return nil
}

func (f *factory[UnderlayClient]) Refresh(obj runtime.Object) {
	f.lock.Lock()
	defer f.lock.Unlock()

	informerType := reflect.TypeOf(obj)
	p, ok := f.pollers[informerType]
	if !ok {
		f.logger.Info("no poller for type", "type", informerType)
		return
	}
	p.Refresh()
}

type ListerWatcher[UnderlayClient any] interface {
	cache.ListerWatcher
}

type listerWatcher[UnderlayClient any] struct {
	logger logr.Logger
	p      Poller
}

func NewListerWatcher[UnderlayClient any](
	logger logr.Logger,
	p Poller,
) ListerWatcher[UnderlayClient] {
	return &listerWatcher[UnderlayClient]{
		logger: logger,
		p:      p,
	}
}

// List implements the ListerWatcher interface.
//
//nolint:gocritic // implements an interface
func (lw *listerWatcher[UnderlayClient]) List(_ metav1.ListOptions) (runtime.Object, error) {
	//nolint:mnd // refactor to use a constant if necessary
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	list, err := lw.p.Sync(ctx)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// Watch implements the ListerWatcher interface.
//
//nolint:gocritic // implements an interface
func (lw *listerWatcher[UnderlayClient]) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	ctx, cancel := context.WithCancel(context.Background())

	resultCh := make(chan watch.Event, bufSize)
	w := watch.NewProxyWatcher(resultCh)

	go func() {
		<-w.StopChan()
		cancel()
		close(resultCh)
	}()

	go lw.p.Run(ctx, resultCh)

	return w, nil
}

type GlobalCacheLister[T any, PT Object[T]] interface {
	ByCluster(cluster string) CacheLister[T, PT]
}

type CacheLister[T any, PT Object[T]] interface {
	List(selector labels.Selector) ([]PT, error)
	Get(name string) (PT, error)
}

type globalCacheLister[T any, PT Object[T]] struct {
	indexer cache.Indexer
	gr      schema.GroupResource
}

type cacheLister[T any, PT Object[T]] struct {
	indexer cache.Indexer
	gr      schema.GroupResource
	ns      string
}

func (s *globalCacheLister[T, PT]) ByCluster(cluster string) CacheLister[T, PT] {
	return &cacheLister[T, PT]{
		indexer: s.indexer,
		ns:      cluster,
		gr:      s.gr,
	}
}

func (s *cacheLister[T, PT]) List(selector labels.Selector) (ret []PT, _ error) {
	if err := cache.ListAllByNamespace(s.indexer, s.ns, selector, func(m any) {
		ret = append(ret, m.(PT))
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *cacheLister[T, PT]) Get(name string) (PT, error) {
	obj, exists, err := s.indexer.GetByKey(s.ns + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(s.gr, name)
	}
	return obj.(PT), nil
}

func NewGlobalCacheLister[T any, PT Object[T]](indexer cache.Indexer, gr schema.GroupResource) GlobalCacheLister[T, PT] {
	return &globalCacheLister[T, PT]{
		indexer: indexer,
		gr:      gr,
	}
}

type RefreshableCacheLister[T any, PT Object[T]] interface {
	CacheLister[T, PT]
	Refresher
}

type Refresher interface {
	Refresh()
}

type RefreshFunc func()

func (f RefreshFunc) Refresh() {
	f()
}

type refreshableCacheLister[T any, PT Object[T]] struct {
	CacheLister[T, PT]
	Refresher
}

func CacheWithRefresher[T any, PT Object[T]](c CacheLister[T, PT], r Refresher) RefreshableCacheLister[T, PT] {
	return &refreshableCacheLister[T, PT]{
		CacheLister: c,
		Refresher:   r,
	}
}
