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

package storage

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	clientv1alpha1 "github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/typed/pingcap.com/v1alpha1"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	informerv1alpha1 "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap.com/v1alpha1"
	listerv1alpha1 "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap.com/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
	"k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// store implements a ConfigMap backed storage.Interface
type store struct {
	lister    listerv1alpha1.DataResourceLister
	informer  informerv1alpha1.DataResourceInformer
	client    clientv1alpha1.DataResourceInterface
	codec     runtime.Codec
	versioner storage.Versioner

	objType     runtime.Object
	newListFunc func() runtime.Object

	//queue     workqueue.RateLimitingInterface
}

// New returns an kubernetes configmap implementation of storage.Interface.
func NewApiServerStore(restConfig *rest.Config, codec runtime.Codec, namespace string, objType runtime.Object, newListFunc func() runtime.Object) (storage.Interface, factory.DestroyFunc) {
	cli, err := versioned.NewForConfig(restConfig)
	if err != nil {
		glog.Fatalf("failed to create Clientset: %v", err)
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(cli, 1*time.Minute, informers.WithNamespace(namespace))

	inf := informerFactory.Pingcap().V1alpha1().DataResources()
	s := &store{
		lister:    inf.Lister(),
		informer:  inf,
		client:    cli.PingcapV1alpha1().DataResources(namespace),
		versioner: etcd.APIObjectVersioner{},
		codec:     codec,

		objType:     objType,
		newListFunc: newListFunc,

		//queue: workqueue.NewNamedRateLimitingQueue(
		//	workqueue.DefaultControllerRateLimiter(),
		//	"dataresources",
		//),
	}
	// TODO: informer based watch implementation
	//s.informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
	//	AddFunc: s.addEvent,
	//	UpdateFunc: s.updateEvent,
	//	DeleteFunc: s.deleteEvent,
	//})
	ctx, cancel := context.WithCancel(context.Background())
	informerFactory.Start(ctx.Done())
	for v, synced := range informerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			glog.Fatalf("error syncing informer for %v", v)
		}
	}
	destroy := func() {
		cancel()
	}
	return s, destroy
}

func (s *store) Versioner() storage.Versioner {
	return s.versioner
}

func (s *store) Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error {
	if version, err := s.versioner.ObjectResourceVersion(obj); err == nil && version != 0 {
		return errors.New("resourceVersion should not be set on objects to be created")
	}
	if err := s.versioner.PrepareObjectForStorage(obj); err != nil {
		return fmt.Errorf("PrepareObjectForStorage failed: %v", err)
	}
	data, err := runtime.Encode(s.codec, obj)
	if err != nil {
		return err
	}
	objKey := newObjectKey(key)
	dr := &v1alpha1.DataResource{
		ObjectMeta: objKey.objectMeta(),
		Data:       data,
	}
	// Set the UID of dataresource same as the resource stored in favor of preconditions on deleting
	if metaobj, ok := obj.(metav1.Object); ok {
		dr.UID = metaobj.GetUID()
	}
	ret, err := s.client.Create(dr)
	if err != nil {
		return err
	}
	if out != nil {
		rv, err := s.versioner.ParseResourceVersion(ret.ResourceVersion)
		if err != nil {
			return err
		}
		err = decode(s.codec, s.versioner, ret.Data, out, int64(rv))
		return err
	}
	return nil
}

func (s *store) Delete(ctx context.Context, key string, out runtime.Object, preconditions *storage.Preconditions) error {
	objKey := newObjectKey(key)
	return s.client.Delete(objKey.fullName(), &metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{
			UID: preconditions.UID,
		},
	})
}

func (s *store) Watch(ctx context.Context, key string, resourceVersion string, p storage.SelectionPredicate) (watch.Interface, error) {
	// Client based watching hold a connection to apiserver for each watch request
	// TODO: replace with a sharedInformer based watching strategy
	return s.WatchList(ctx, key, resourceVersion, p)
}

func (s *store) WatchList(ctx context.Context, key string, resourceVersion string, p storage.SelectionPredicate) (watch.Interface, error) {
	// Client based watching hold a connection to apiserver for each watch request
	// TODO: replace with a sharedInformer based watching strategy
	objKey := newObjectKey(key)
	w, err := s.client.Watch(metav1.ListOptions{
		LabelSelector:   objKey.labelSelectorStr(),
		ResourceVersion: resourceVersion,
	})
	if err != nil {
		return nil, err
	}
	outWatcher := newWatcherWrapperWithPrediction(s, w, p)
	go outWatcher.run()
	return outWatcher, nil
}

func (s *store) Get(ctx context.Context, key string, resourceVersion string, out runtime.Object, ignoreNotFound bool) error {
	objKey := newObjectKey(key)
	selector, err := objKey.selector()
	if err != nil {
		return err
	}
	ret, err := s.lister.List(selector)
	if err != nil {
		return err
	}
	if len(ret) == 0 {
		if ignoreNotFound {
			return runtime.SetZeroValue(out)
		}
		return storage.NewKeyNotFoundError(key, 0)
	}
	if len(ret) > 1 {
		return storage.NewInternalError("more than 1 resources found")
	}
	rv, err := s.versioner.ParseResourceVersion(ret[0].ResourceVersion)
	if err != nil {
		return err
	}
	return decode(s.codec, s.versioner, ret[0].Data, out, int64(rv))
}

func (s *store) GetToList(ctx context.Context, key string, resourceVersion string, p storage.SelectionPredicate, listObj runtime.Object) error {
	listPtr, err := meta.GetItemsPtr(listObj)
	if err != nil {
		return err
	}
	v, err := conversion.EnforcePtr(listPtr)
	if err != nil || v.Kind() != reflect.Slice {
		panic("need ptr to slice")
	}
	objKey := newObjectKey(key)
	selector, err := objKey.selector()
	if err != nil {
		return err
	}
	ret, err := s.lister.List(selector)
	if err != nil {
		return err
	}
	if len(ret) > 0 {
		rv, err := s.versioner.ParseResourceVersion(ret[0].ResourceVersion)
		if err != nil {
			return err
		}
		if err := appendListItem(v, ret[0].Data, rv, p, s.codec, s.versioner); err != nil {
			return err
		}
	}
	return s.versioner.UpdateList(listObj, 0, "")
}

func (s *store) List(ctx context.Context, key string, resourceVersion string, pred storage.SelectionPredicate, listObj runtime.Object) error {
	listPtr, err := meta.GetItemsPtr(listObj)
	if err != nil {
		return err
	}
	li, err := conversion.EnforcePtr(listPtr)
	if err != nil || li.Kind() != reflect.Slice {
		panic("need ptr to slice")
	}
	objKey := newObjectKey(key)
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: objKey.labelMap(),
	})
	if err != nil {
		return err
	}
	ret, err := s.lister.List(selector)
	if err != nil {
		return err
	}
	for _, v := range ret {
		rv, err := s.versioner.ParseResourceVersion(v.ResourceVersion)
		if err != nil {
			return err
		}
		if err := appendListItem(li, v.Data, rv, pred, s.codec, s.versioner); err != nil {
			return err
		}
	}

	return s.versioner.UpdateList(listObj, 0, "")
}

func (s *store) GuaranteedUpdate(ctx context.Context,
	key string,
	out runtime.Object,
	ignoreNotFound bool,
	precondtions *storage.Preconditions,
	tryUpdate storage.UpdateFunc,
	suggestion ...runtime.Object,
) error {
	v, err := conversion.EnforcePtr(out)
	if err != nil {
		panic("unable to convert output object to pointer")
	}
	objKey := newObjectKey(key)
	getCurrentState := func() (*objState, error) {
		ret, err := s.client.Get(objKey.fullName(), metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		rv, err := s.versioner.ParseResourceVersion(ret.ResourceVersion)
		if err != nil {
			return nil, err
		}
		state := &objState{
			obj:      reflect.New(v.Type()).Interface().(runtime.Object),
			meta:     &storage.ResponseMeta{},
			resource: ret,
		}
		state.meta.ResourceVersion = uint64(rv)
		if err := decode(s.codec, s.versioner, ret.Data, state.obj, int64(rv)); err != nil {
			return nil, err
		}
		return state, nil
	}
	var origState *objState
	origState, err = getCurrentState()
	if err != nil {
		return err
	}
	shouldRefresh := false
	for {
		if shouldRefresh {
			// Refresh object
			origState, err = getCurrentState()
			if err != nil {
				return err
			}
			shouldRefresh = false
		}
		if err := precondtions.Check(key, origState.obj); err != nil {
			return err
		}

		// ttl is not supported and ignored
		ret, _, err := tryUpdate(origState.obj, *origState.meta)
		if err != nil {
			if apierrors.IsConflict(err) {
				shouldRefresh = true
				// Retry
				continue
			}
			return err
		}

		data, err := runtime.Encode(s.codec, ret)
		if err != nil {
			return err
		}
		toUpdate := origState.resource.DeepCopy()
		toUpdate.Data = data
		updateResp, err := s.client.Update(toUpdate)
		if err != nil {
			if apierrors.IsConflict(err) {
				shouldRefresh = true
				// Retry
				continue
			}
			return err
		}
		rv, err := s.versioner.ParseResourceVersion(updateResp.ResourceVersion)
		if err != nil {
			return err
		}
		return decode(s.codec, s.versioner, updateResp.Data, out, int64(rv))
	}
}

func (s *store) Count(key string) (int64, error) {
	// Count is used to update metric, we have not enable the metric so leave the empty implementation now
	// TODO: implement is necessary
	return 0, nil
}

// decode decodes value of bytes into object. It will also set the object resource version to rev.
// On success, objPtr would be set to the object.
func decode(codec runtime.Codec, versioner storage.Versioner, value []byte, objPtr runtime.Object, rev int64) error {
	if _, err := conversion.EnforcePtr(objPtr); err != nil {
		panic("unable to convert output object to pointer")
	}
	_, _, err := codec.Decode(value, nil, objPtr)
	if err != nil {
		return err
	}
	// being unable to set the version does not prevent the object from being extracted
	return versioner.UpdateObject(objPtr, uint64(rev))
}

// appendListItem decodes and appends the object (if it passes filter) to v, which must be a slice.
func appendListItem(v reflect.Value, data []byte, rev uint64, pred storage.SelectionPredicate, codec runtime.Codec, versioner storage.Versioner) error {
	obj, _, err := codec.Decode(data, nil, reflect.New(v.Type().Elem()).Interface().(runtime.Object))
	if err != nil {
		return err
	}
	// being unable to set the version does not prevent the object from being extracted
	if err := versioner.UpdateObject(obj, rev); err != nil {
		return err
	}
	if matched, err := pred.Matches(obj); err == nil && matched {
		v.Set(reflect.Append(v, reflect.ValueOf(obj).Elem()))
	}
	return nil
}

type objState struct {
	obj  runtime.Object
	meta *storage.ResponseMeta

	resource *v1alpha1.DataResource
}

// An naive watcher implementation
// TODO: replace this with sharedInformer based watcher
type watcherWrapperWithPrediction struct {
	*store

	watcher watch.Interface
	pred    storage.SelectionPredicate
	stopCh  chan struct{}

	resultChan chan watch.Event
}

func newWatcherWrapperWithPrediction(s *store, w watch.Interface, pred storage.SelectionPredicate) *watcherWrapperWithPrediction {
	return &watcherWrapperWithPrediction{
		store:      s,
		watcher:    w,
		pred:       pred,
		stopCh:     make(chan struct{}),
		resultChan: make(chan watch.Event),
	}
}

func (w *watcherWrapperWithPrediction) run() {
	ch := w.watcher.ResultChan()
	for {
		select {
		case event := <-ch:
			if event.Type == watch.Error {
				// TODO: need more investigation to determine whether or not to send this error out
				klog.Errorf("Error when watching resources, %v", event.Object)
				continue
			}
			dr, ok := event.Object.(*v1alpha1.DataResource)
			if !ok {
				klog.Errorf("Encounters unknown object when watching, %v", event.Object)
				continue
			}
			rv, err := w.versioner.ParseResourceVersion(dr.ResourceVersion)
			if err != nil {
				klog.Errorf("Error when parse resource version when watching, %v", err)
				continue
			}
			out := w.store.objType.DeepCopyObject()
			err = decode(w.store.codec, w.store.versioner, dr.Data, out, int64(rv))
			if err != nil {
				klog.Errorf("Error decoding object when watching, %v", err)
				continue
			}
			// If the resource is interested by client, send it out
			b, err := w.pred.Matches(out)
			if err != nil {
				klog.Errorf("Error when watching resources, %v", err)
				continue
			}
			if b {
				w.resultChan <- watch.Event{
					Type:   event.Type,
					Object: out,
				}
			}
		case <-w.stopCh:
			return
		}
	}
}

func (w *watcherWrapperWithPrediction) Stop() {
	close(w.stopCh)
	w.watcher.Stop()
}

func (w *watcherWrapperWithPrediction) ResultChan() <-chan watch.Event {
	return w.resultChan
}
