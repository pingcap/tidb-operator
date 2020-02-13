// Copyright 2019 PingCAP, Inc.
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

package storage

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
)

// ApiServerRestOptionsFactory produce RestOptions for resources using kube-apiserver as storage
type ApiServerRestOptionsFactory struct {
	RestConfig       *rest.Config
	StorageNamespace string
	Codec            runtime.Codec
}

// GetRestOptions fetch a new RestOptions from the factory
func (f *ApiServerRestOptionsFactory) GetRESTOptions(resource schema.GroupResource) (generic.RESTOptions, error) {
	opts := generic.RESTOptions{
		ResourcePrefix:          getResourcePrefix(resource),
		EnableGarbageCollection: true,
		StorageConfig:           storagebackend.NewDefaultConfig("", f.Codec),
		Decorator:               f.newApiServerStorageDecorator(),
	}
	return opts, nil
}

// newApiServerStorage returns a new storage decorator which creates a kube-apiserver backed storage interface
func (f *ApiServerRestOptionsFactory) newApiServerStorageDecorator() generic.StorageDecorator {
	return func(
		config *storagebackend.Config,
		resourcePrefix string,
		keyFunc func(obj runtime.Object) (string, error),
		newFunc func() runtime.Object,
		newListFunc func() runtime.Object,
		getAttrsFunc storage.AttrFunc,
		trigger storage.IndexerFuncs,
	) (storage.Interface, factory.DestroyFunc, error) {
		cli, err := versioned.NewForConfig(f.RestConfig)
		if err != nil {
			klog.Fatalf("failed to create Clientset: %v", err)
		}
		objectType := newFunc()
		return NewApiServerStore(cli, f.Codec, f.StorageNamespace, objectType, newListFunc)
	}
}

// getResourcePrefix return a '/{group}/{kind}' prefix for resource key with '/{group}/{kind}/{ns}/{name}' format
func getResourcePrefix(resource schema.GroupResource) string {
	return fmt.Sprintf("/%s/%s", resource.Group, resource.Resource)
}
