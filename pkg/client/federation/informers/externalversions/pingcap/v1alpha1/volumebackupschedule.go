// Copyright PingCAP, Inc.
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

package v1alpha1

import (
	"context"
	time "time"

	pingcapv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	versioned "github.com/pingcap/tidb-operator/pkg/client/federation/clientset/versioned"
	internalinterfaces "github.com/pingcap/tidb-operator/pkg/client/federation/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/pingcap/tidb-operator/pkg/client/federation/listers/pingcap/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// VolumeBackupScheduleInformer provides access to a shared informer and lister for
// VolumeBackupSchedules.
type VolumeBackupScheduleInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.VolumeBackupScheduleLister
}

type volumeBackupScheduleInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewVolumeBackupScheduleInformer constructs a new informer for VolumeBackupSchedule type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewVolumeBackupScheduleInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredVolumeBackupScheduleInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredVolumeBackupScheduleInformer constructs a new informer for VolumeBackupSchedule type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredVolumeBackupScheduleInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.FederationV1alpha1().VolumeBackupSchedules(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.FederationV1alpha1().VolumeBackupSchedules(namespace).Watch(context.TODO(), options)
			},
		},
		&pingcapv1alpha1.VolumeBackupSchedule{},
		resyncPeriod,
		indexers,
	)
}

func (f *volumeBackupScheduleInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredVolumeBackupScheduleInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *volumeBackupScheduleInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&pingcapv1alpha1.VolumeBackupSchedule{}, f.defaultInformer)
}

func (f *volumeBackupScheduleInformer) Lister() v1alpha1.VolumeBackupScheduleLister {
	return v1alpha1.NewVolumeBackupScheduleLister(f.Informer().GetIndexer())
}
