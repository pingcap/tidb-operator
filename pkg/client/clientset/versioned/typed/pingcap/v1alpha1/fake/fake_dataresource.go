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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeDataResources implements DataResourceInterface
type FakeDataResources struct {
	Fake *FakePingcapV1alpha1
	ns   string
}

var dataresourcesResource = schema.GroupVersionResource{Group: "pingcap.com", Version: "v1alpha1", Resource: "dataresources"}

var dataresourcesKind = schema.GroupVersionKind{Group: "pingcap.com", Version: "v1alpha1", Kind: "DataResource"}

// Get takes name of the dataResource, and returns the corresponding dataResource object, and an error if there is any.
func (c *FakeDataResources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.DataResource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(dataresourcesResource, c.ns, name), &v1alpha1.DataResource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DataResource), err
}

// List takes label and field selectors, and returns the list of DataResources that match those selectors.
func (c *FakeDataResources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.DataResourceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(dataresourcesResource, dataresourcesKind, c.ns, opts), &v1alpha1.DataResourceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.DataResourceList{ListMeta: obj.(*v1alpha1.DataResourceList).ListMeta}
	for _, item := range obj.(*v1alpha1.DataResourceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested dataResources.
func (c *FakeDataResources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(dataresourcesResource, c.ns, opts))

}

// Create takes the representation of a dataResource and creates it.  Returns the server's representation of the dataResource, and an error, if there is any.
func (c *FakeDataResources) Create(ctx context.Context, dataResource *v1alpha1.DataResource, opts v1.CreateOptions) (result *v1alpha1.DataResource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(dataresourcesResource, c.ns, dataResource), &v1alpha1.DataResource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DataResource), err
}

// Update takes the representation of a dataResource and updates it. Returns the server's representation of the dataResource, and an error, if there is any.
func (c *FakeDataResources) Update(ctx context.Context, dataResource *v1alpha1.DataResource, opts v1.UpdateOptions) (result *v1alpha1.DataResource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(dataresourcesResource, c.ns, dataResource), &v1alpha1.DataResource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DataResource), err
}

// Delete takes name of the dataResource and deletes it. Returns an error if one occurs.
func (c *FakeDataResources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(dataresourcesResource, c.ns, name, opts), &v1alpha1.DataResource{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeDataResources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(dataresourcesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.DataResourceList{})
	return err
}

// Patch applies the patch and returns the patched dataResource.
func (c *FakeDataResources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.DataResource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(dataresourcesResource, c.ns, name, pt, data, subresources...), &v1alpha1.DataResource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DataResource), err
}
