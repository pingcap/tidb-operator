// Copyright 2019. PingCAP, Inc.
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

// Code generated by informer-gen. DO NOT EDIT.

package internalversion

import (
	time "time"

	example "github.com/pingcap/tidb-operator/tests/pkg/apiserver/apis/example"
	clientsetinternalversion "github.com/pingcap/tidb-operator/tests/pkg/apiserver/client/clientset/internalversion"
	internalinterfaces "github.com/pingcap/tidb-operator/tests/pkg/apiserver/client/informers/internalversion/internalinterfaces"
	internalversion "github.com/pingcap/tidb-operator/tests/pkg/apiserver/client/listers/example/internalversion"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// PodInformer provides access to a shared informer and lister for
// Pods.
type PodInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() internalversion.PodLister
}

type podInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewPodInformer constructs a new informer for Pod type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewPodInformer(client clientsetinternalversion.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredPodInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredPodInformer constructs a new informer for Pod type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredPodInformer(client clientsetinternalversion.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Example().Pods(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.Example().Pods(namespace).Watch(options)
			},
		},
		&example.Pod{},
		resyncPeriod,
		indexers,
	)
}

func (f *podInformer) defaultInformer(client clientsetinternalversion.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredPodInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *podInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&example.Pod{}, f.defaultInformer)
}

func (f *podInformer) Lister() internalversion.PodLister {
	return internalversion.NewPodLister(f.Informer().GetIndexer())
}
