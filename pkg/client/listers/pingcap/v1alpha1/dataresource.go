/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// DataResourceLister helps list DataResources.
type DataResourceLister interface {
	// List lists all DataResources in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.DataResource, err error)
	// DataResources returns an object that can list and get DataResources.
	DataResources(namespace string) DataResourceNamespaceLister
	DataResourceListerExpansion
}

// dataResourceLister implements the DataResourceLister interface.
type dataResourceLister struct {
	indexer cache.Indexer
}

// NewDataResourceLister returns a new DataResourceLister.
func NewDataResourceLister(indexer cache.Indexer) DataResourceLister {
	return &dataResourceLister{indexer: indexer}
}

// List lists all DataResources in the indexer.
func (s *dataResourceLister) List(selector labels.Selector) (ret []*v1alpha1.DataResource, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.DataResource))
	})
	return ret, err
}

// DataResources returns an object that can list and get DataResources.
func (s *dataResourceLister) DataResources(namespace string) DataResourceNamespaceLister {
	return dataResourceNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// DataResourceNamespaceLister helps list and get DataResources.
type DataResourceNamespaceLister interface {
	// List lists all DataResources in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.DataResource, err error)
	// Get retrieves the DataResource from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.DataResource, error)
	DataResourceNamespaceListerExpansion
}

// dataResourceNamespaceLister implements the DataResourceNamespaceLister
// interface.
type dataResourceNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all DataResources in the indexer for a given namespace.
func (s dataResourceNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.DataResource, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.DataResource))
	})
	return ret, err
}

// Get retrieves the DataResource from the indexer for a given namespace and name.
func (s dataResourceNamespaceLister) Get(name string) (*v1alpha1.DataResource, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("dataresource"), name)
	}
	return obj.(*v1alpha1.DataResource), nil
}
