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

package informertest

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const defaultEventBuffer = 100

type delayedListerWatcher struct {
	lw    cache.ListerWatcher
	delay time.Duration
}

// List implements cache.ListerWatcher
// nolint: gocritic // it implements cache.ListerWatcher
func (lw *delayedListerWatcher) List(options metav1.ListOptions) (runtime.Object, error) {
	return lw.lw.List(options)
}

// Watch implements cache.ListerWatcher
// nolint: gocritic // it implements cache.ListerWatcher
func (lw *delayedListerWatcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	w, err := lw.lw.Watch(options)
	if err != nil {
		return nil, err
	}

	queue := workqueue.NewTypedDelayingQueue[*watch.Event]()
	ch := make(chan watch.Event, defaultEventBuffer)
	pw := watch.NewProxyWatcher(ch)
	go func() {
		for e := range w.ResultChan() {
			queue.AddAfter(&e, lw.delay)
		}
		queue.ShutDownWithDrain()
	}()
	go func() {
		for {
			e, done := queue.Get()
			if done {
				break
			}
			ch <- *e
			queue.Done(e)
		}
		close(ch)
	}()
	go func() {
		<-pw.StopChan()
		w.Stop()
	}()

	return pw, nil
}

func NewDelayedInformerFunc(delay time.Duration) func(
	lw cache.ListerWatcher,
	exampleObject runtime.Object,
	defaultEventHandlerResyncPeriod time.Duration,
	indexers cache.Indexers,
) cache.SharedIndexInformer {
	return func(
		lw cache.ListerWatcher,
		exampleObject runtime.Object,
		defaultEventHandlerResyncPeriod time.Duration,
		indexers cache.Indexers,
	) cache.SharedIndexInformer {
		delayed := &delayedListerWatcher{
			lw:    lw,
			delay: delay,
		}
		return cache.NewSharedIndexInformer(delayed, exampleObject, defaultEventHandlerResyncPeriod, indexers)
	}
}
