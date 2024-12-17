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

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
	"sigs.k8s.io/yaml"

	"github.com/pingcap/tidb-operator/pkg/scheme"
)

type fakeParser struct{}

func (*fakeParser) Type(schema.GroupVersionKind) *typed.ParseableType {
	return &typed.DeducedParseableType
}

func NewFakeClient(objs ...client.Object) Client {
	c := newFakeUnderlayClient(objs...)

	return &applier{
		WithWatch: c,
		parser:    &fakeParser{},
	}
}

type fakeUnderlayClient struct {
	testing.Fake
	tracker    testing.ObjectTracker
	scheme     *runtime.Scheme
	restMapper meta.RESTMapper
}

var _ client.WithWatch = &fakeUnderlayClient{}

func newFakeUnderlayClient(objs ...client.Object) client.WithWatch {
	t := testing.NewObjectTracker(scheme.Scheme, scheme.Codecs.UniversalDecoder())
	for _, obj := range objs {
		if err := t.Add(obj); err != nil {
			panic(err)
		}
	}
	mapper := testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)

	c := fakeUnderlayClient{
		tracker:    t,
		scheme:     scheme.Scheme,
		restMapper: mapper,
	}
	c.AddReactor("*", "*", c.ObjectReaction())
	c.AddWatchReactor("*", func(action testing.Action) (handled bool, ret watch.Interface, err error) {
		gvr := action.GetResource()
		ns := action.GetNamespace()
		w, err := t.Watch(gvr, ns)
		if err != nil {
			return false, nil, err
		}
		return true, w, nil
	})

	return &c
}

func (c *fakeUnderlayClient) Scheme() *runtime.Scheme {
	return c.scheme
}

func (c *fakeUnderlayClient) RESTMapper() meta.RESTMapper {
	return c.restMapper
}

// GroupVersionKindFor returns the GroupVersionKind for the given object.
func (c *fakeUnderlayClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return apiutil.GVKForObject(obj, c.scheme)
}

// IsObjectNamespaced returns true if the GroupVersionKind of the object is namespaced.
func (c *fakeUnderlayClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return apiutil.IsObjectNamespaced(obj, c.scheme, c.restMapper)
}

func (c *fakeUnderlayClient) GroupVersionResourceFor(obj runtime.Object) (schema.GroupVersionResource, error) {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	return mapping.Resource, nil
}

func (c *fakeUnderlayClient) Create(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return err
	}

	namespaced, err := apiutil.IsGVKNamespaced(gvk, c.restMapper)
	if err != nil {
		return err
	}

	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	var action testing.CreateAction
	if namespaced {
		action = testing.NewCreateAction(mapping.Resource, obj.GetNamespace(), obj)
	} else {
		action = testing.NewRootCreateAction(mapping.Resource, obj)
	}
	newObj, err := c.Invokes(action, nil)
	if err != nil {
		return err
	}
	if newObj == nil {
		return fmt.Errorf("obj is not handled")
	}

	nv := reflect.ValueOf(newObj).Elem()
	v := reflect.ValueOf(obj).Elem()
	v.Set(nv)

	return nil
}

func (c *fakeUnderlayClient) Delete(_ context.Context, obj client.Object, opts ...client.DeleteOption) error {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return err
	}

	namespaced, err := apiutil.IsGVKNamespaced(gvk, c.restMapper)
	if err != nil {
		return err
	}

	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	options := client.DeleteOptions{}
	options.ApplyOptions(opts)

	var action testing.DeleteAction
	if namespaced {
		action = testing.NewDeleteActionWithOptions(mapping.Resource, obj.GetNamespace(), obj.GetName(), *options.AsDeleteOptions())
	} else {
		action = testing.NewRootDeleteActionWithOptions(mapping.Resource, obj.GetName(), *options.AsDeleteOptions())
	}

	if _, err := c.Invokes(action, nil); err != nil {
		return err
	}

	return nil
}

// TODO(liubo02): impl it
func (*fakeUnderlayClient) DeleteAllOf(_ context.Context, _ client.Object, _ ...client.DeleteAllOfOption) error {
	return nil
}

func (c *fakeUnderlayClient) Update(_ context.Context, obj client.Object, _ ...client.UpdateOption) error {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return err
	}

	namespaced, err := apiutil.IsGVKNamespaced(gvk, c.restMapper)
	if err != nil {
		return err
	}

	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	var action testing.UpdateAction
	if namespaced {
		action = testing.NewUpdateAction(mapping.Resource, obj.GetNamespace(), obj)
	} else {
		action = testing.NewRootUpdateAction(mapping.Resource, obj)
	}
	newObj, err := c.Invokes(action, nil)
	if err != nil {
		return err
	}
	if newObj == nil {
		return fmt.Errorf("obj is not handled")
	}

	nv := reflect.ValueOf(newObj).Elem()
	v := reflect.ValueOf(obj).Elem()
	v.Set(nv)

	return nil
}

func (c *fakeUnderlayClient) Patch(_ context.Context, obj client.Object, patch client.Patch, _ ...client.PatchOption) error {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return err
	}

	namespaced, err := apiutil.IsGVKNamespaced(gvk, c.restMapper)
	if err != nil {
		return err
	}

	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	data, err := patch.Data(obj)
	if err != nil {
		return err
	}

	var action testing.PatchAction
	if namespaced {
		action = testing.NewPatchAction(mapping.Resource, obj.GetNamespace(), obj.GetName(), patch.Type(), data)
	} else {
		action = testing.NewRootPatchAction(mapping.Resource, obj.GetName(), patch.Type(), data)
	}
	newObj, err := c.Invokes(action, nil)
	if err != nil {
		return err
	}
	if newObj == nil {
		return fmt.Errorf("obj is not handled")
	}

	nv := reflect.ValueOf(newObj).Elem()
	v := reflect.ValueOf(obj).Elem()
	v.Set(nv)

	return nil
}

func (c *fakeUnderlayClient) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return err
	}

	namespaced, err := apiutil.IsGVKNamespaced(gvk, c.restMapper)
	if err != nil {
		return err
	}

	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	var action testing.GetAction
	if namespaced {
		action = testing.NewGetAction(mapping.Resource, key.Namespace, key.Name)
	} else {
		action = testing.NewRootGetAction(mapping.Resource, key.Name)
	}

	newObj, err := c.Invokes(action, nil)
	if err != nil {
		return err
	}
	if newObj == nil {
		return fmt.Errorf("obj is not handled")
	}

	nv := reflect.ValueOf(newObj).Elem()
	v := reflect.ValueOf(obj).Elem()
	v.Set(nv)

	return nil
}

func (c *fakeUnderlayClient) Watch(_ context.Context, obj client.ObjectList, opts ...client.ListOption) (watch.Interface, error) {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return nil, err
	}

	namespaced, err := apiutil.IsGVKNamespaced(gvk, c.restMapper)
	if err != nil {
		return nil, err
	}

	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	options := client.ListOptions{}
	options.ApplyOptions(opts)

	var action testing.WatchAction
	if namespaced {
		action = testing.NewWatchAction(mapping.Resource, options.Namespace, *options.AsListOptions())
	} else {
		action = testing.NewRootWatchAction(mapping.Resource, *options.AsListOptions())
	}

	return c.InvokesWatch(action)
}

func (c *fakeUnderlayClient) List(_ context.Context, obj client.ObjectList, opts ...client.ListOption) error {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return err
	}

	itemGVK := gvk.GroupVersion().WithKind(strings.TrimSuffix(gvk.Kind, "List"))

	namespaced, err := apiutil.IsGVKNamespaced(itemGVK, c.restMapper)
	if err != nil {
		return err
	}

	mapping, err := c.restMapper.RESTMapping(itemGVK.GroupKind(), itemGVK.Version)
	if err != nil {
		return err
	}

	options := client.ListOptions{}
	options.ApplyOptions(opts)

	var action testing.ListAction
	if namespaced {
		action = testing.NewListAction(mapping.Resource, itemGVK, options.Namespace, *options.AsListOptions())
	} else {
		action = testing.NewRootListAction(mapping.Resource, itemGVK, *options.AsListOptions())
	}

	newObj, err := c.Invokes(action, nil)
	if err != nil {
		return err
	}
	if newObj == nil {
		return fmt.Errorf("obj is not handled")
	}

	nv := reflect.ValueOf(newObj).Elem()
	v := reflect.ValueOf(obj).Elem()
	v.Set(nv)

	return nil
}

type SubResourceClient struct {
	*fakeUnderlayClient
	subResource string
}

func (c *fakeUnderlayClient) Status() client.SubResourceWriter {
	return &SubResourceClient{
		fakeUnderlayClient: c,
		subResource:        "status",
	}
}

func (c *fakeUnderlayClient) SubResource(sub string) client.SubResourceClient {
	return &SubResourceClient{
		fakeUnderlayClient: c,
		subResource:        sub,
	}
}

func (*SubResourceClient) Create(_ context.Context, _, _ client.Object, _ ...client.SubResourceCreateOption) error {
	return nil
}

func (*SubResourceClient) Update(_ context.Context, _ client.Object, _ ...client.SubResourceUpdateOption) error {
	return nil
}

func (*SubResourceClient) Patch(_ context.Context, _ client.Object, _ client.Patch, _ ...client.SubResourcePatchOption) error {
	return nil
}

func (*SubResourceClient) Get(_ context.Context, _, _ client.Object, _ ...client.SubResourceGetOption) error {
	return nil
}

func (c *fakeUnderlayClient) ObjectReaction() testing.ReactionFunc {
	return func(action testing.Action) (bool, runtime.Object, error) {
		switch action := action.(type) {
		case testing.ListActionImpl:
			return c.ListReactionFunc(&action)
		case testing.PatchActionImpl:
			return c.PatchReactionFunc(&action)
		default:
			return testing.ObjectReaction(c.tracker)(action)
		}
	}
}

func (c *fakeUnderlayClient) ListReactionFunc(action *testing.ListActionImpl) (bool, runtime.Object, error) {
	obj, err := c.tracker.List(action.GetResource(), action.GetKind(), action.GetNamespace())
	if err != nil {
		return true, nil, err
	}

	items, err := meta.ExtractList(obj)
	if err != nil {
		return true, nil, err
	}

	filtered, err := filterList(items, action.ListRestrictions.Labels, action.ListRestrictions.Fields)
	if err != nil {
		return true, nil, err
	}

	if err := meta.SetList(obj, filtered); err != nil {
		return true, nil, err
	}

	return true, obj, nil
}

// TODO: support field selector
func filterList(objs []runtime.Object, ls labels.Selector, _ fields.Selector) ([]runtime.Object, error) {
	out := make([]runtime.Object, 0, len(objs))
	for _, obj := range objs {
		m, err := meta.Accessor(obj)
		if err != nil {
			return nil, err
		}
		if ls != nil {
			if !ls.Matches(labels.Set(m.GetLabels())) {
				continue
			}
		}
		out = append(out, obj)
	}
	return out, nil
}

//nolint:gocyclo // refactor if possible
func (c *fakeUnderlayClient) PatchReactionFunc(action *testing.PatchActionImpl) (bool, runtime.Object, error) {
	gvr := action.GetResource()
	ns := action.GetNamespace()
	name := action.GetName()

	gvk, err := c.restMapper.KindFor(gvr)
	if err != nil {
		return true, nil, err
	}

	manager, err := managedfields.NewDefaultFieldManager(
		managedfields.NewDeducedTypeConverter(),
		scheme.Scheme,
		scheme.Scheme,
		scheme.Scheme,
		gvk,
		gvk.GroupVersion(),
		action.Subresource,
		nil,
	)
	if err != nil {
		return true, nil, err
	}

	exist := true

	obj, err := c.tracker.Get(gvr, ns, name)
	if err != nil {
		if !errors.IsNotFound(err) {
			return true, nil, err
		}

		if action.GetPatchType() == types.ApplyPatchType {
			exist = false
			newObj, err2 := c.scheme.New(gvk)
			if err2 != nil {
				return true, nil, err2
			}
			obj = newObj
		} else {
			return true, nil, err
		}
	}

	old, err := json.Marshal(obj)
	if err != nil {
		return true, nil, err
	}

	// reset the object in preparation to unmarshal, since unmarshal does not guarantee that fields
	// in obj that are removed by patch are cleared
	value := reflect.ValueOf(obj)
	value.Elem().Set(reflect.New(value.Type().Elem()).Elem())

	switch action.GetPatchType() {
	case types.JSONPatchType:
		patch, err2 := jsonpatch.DecodePatch(action.GetPatch())
		if err2 != nil {
			return true, nil, err2
		}
		modified, err2 := patch.Apply(old)
		if err2 != nil {
			return true, nil, err2
		}

		//nolint:gocritic // use := shadow err
		if err2 = json.Unmarshal(modified, obj); err2 != nil {
			return true, nil, err2
		}
	case types.MergePatchType:
		modified, err2 := jsonpatch.MergePatch(old, action.GetPatch())
		if err2 != nil {
			return true, nil, err2
		}

		//nolint:gocritic // use := shadow err
		if err2 = json.Unmarshal(modified, obj); err2 != nil {
			return true, nil, err2
		}
	case types.StrategicMergePatchType:
		mergedByte, err2 := strategicpatch.StrategicMergePatch(old, action.GetPatch(), obj)
		if err2 != nil {
			return true, nil, err2
		}
		//nolint:gocritic // use := shadow err
		if err2 = json.Unmarshal(mergedByte, obj); err2 != nil {
			return true, nil, err2
		}
	case types.ApplyPatchType:
		patchObj := &unstructured.Unstructured{Object: map[string]any{}}
		if err = yaml.Unmarshal(action.GetPatch(), &patchObj.Object); err != nil {
			return true, nil, fmt.Errorf("error decoding YAML: %w", err)
		}
		obj, err = manager.Apply(obj, patchObj, "tidb-operator", true)
		if err != nil {
			return true, nil, err
		}

	default:
		return true, nil, fmt.Errorf("PatchType is not supported")
	}

	if !exist {
		if err := c.tracker.Create(gvr, obj, ns); err != nil {
			return true, nil, err
		}
	}

	if err := c.tracker.Update(gvr, obj, ns); err != nil {
		return true, nil, err
	}

	return true, obj, nil
}
