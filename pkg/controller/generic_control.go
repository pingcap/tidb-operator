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

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/scheme"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// GenericControlInterface is a wrapper to manage typed object that managed by an arbitrary controller
type TypedControlInterface interface {
	CreateOrUpdateConfigMap(controller runtime.Object, cm *corev1.ConfigMap) (*corev1.ConfigMap, error)
	Delete(controller, obj runtime.Object) error
}

type typedWrapper struct {
	GenericControlInterface
}

// NewTypedControl wraps a GenericControlInterface to a TypedControlInterface
func NewTypedControl(control GenericControlInterface) TypedControlInterface {
	return &typedWrapper{control}
}

func (w *typedWrapper) Delete(controller, obj runtime.Object) error {
	return w.GenericControlInterface.Delete(controller, obj)
}

func (w *typedWrapper) CreateOrUpdateConfigMap(controller runtime.Object, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	result, err := w.GenericControlInterface.CreateOrUpdate(controller, cm, func(existing, desired runtime.Object) error {
		existingCm := existing.(*corev1.ConfigMap)
		desiredCm := desired.(*corev1.ConfigMap)

		existingCm.Data = desiredCm.Data
		existingCm.Labels = desiredCm.Labels
		for k, v := range desiredCm.Annotations {
			existingCm.Annotations[k] = v
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*corev1.ConfigMap), nil
}

// GenericControlInterface manages generic object that managed by an arbitrary controller
type GenericControlInterface interface {
	CreateOrUpdate(controller, obj runtime.Object, mergeFn MergeFn) (runtime.Object, error)
	Delete(controller, obj runtime.Object) error
}

// MergeFn knows how to merge a desired object into the current object.
// Typically, merge should only set the specific fields the caller wants to control to the existing object
// instead of override a whole struct. e.g.
//
// Prefer:
//     existing.spec.type = desired.spec.type
//     existing.spec.externalTrafficPolicy = desired.spec.externalTrafficPolicy
// Instead of:
//     existing.spec = desired.spec
//
// However, this could be tedious for large object if the caller want to control lots of the fields,
// if there is no one else will mutate this object or cooperation is not needed, it is okay to do aggressive
// override. Note that aggressive override usually causes unnecessary updates because the object will be mutated
// after POST/PUT to api-server (e.g. Defaulting), an annotation based technique could be used to avoid such
// updating: set a last-applied-config annotation and diff the annotation instead of the real spec.
type MergeFn func(existing, desired runtime.Object) error

type realGenericControlInterface struct {
	client   client.Client
	recorder record.EventRecorder
}

func NewRealGenericControl(client client.Client, recorder record.EventRecorder) GenericControlInterface {
	return &realGenericControlInterface{client, recorder}
}

// CreateOrUpdate create an object to the Kubernetes cluster for controller, if the object to create is existed,
// call mergeFn to merge the change in new object to the existing object, then update the existing object.
// The object will also be adopted by the given controller.
func (c *realGenericControlInterface) CreateOrUpdate(controller, obj runtime.Object, mergeFn MergeFn) (runtime.Object, error) {

	// controller-runtime/client will mutate the object pointer in-place,
	// to be consistent with other methods in our controller, we copy the object
	// to avoid the in-place mutation here and hereafter.
	desired := obj.DeepCopyObject()
	if err := setControllerReference(controller, desired); err != nil {
		return desired, err
	}

	// 1. try to create and see if there is any conflicts
	err := c.client.Create(context.TODO(), desired)
	if errors.IsAlreadyExists(err) {

		// 2. object has already existed, merge our desired changes to it
		existing := desired.DeepCopyObject()
		key, err := client.ObjectKeyFromObject(existing)
		if err != nil {
			return nil, err
		}
		err = c.client.Get(context.TODO(), key, existing)
		if err != nil {
			return nil, err
		}

		// 3. try to adopt the existing object
		if err := setControllerReference(controller, existing); err != nil {
			return nil, err
		}

		mutated := existing.DeepCopyObject()
		// 4. invoke mergeFn to mutate a copy of the existing object
		if err := mergeFn(mutated, desired); err != nil {
			return nil, err
		}

		// 5. check if the copy is actually mutated
		if !apiequality.Semantic.DeepEqual(existing, mutated) {
			err := c.client.Update(context.TODO(), mutated)
			c.RecordControllerEvent("update", controller, obj, err)
			return mutated, err
		}

		return mutated, nil
	}

	// object do not exist, return the creation result
	c.RecordControllerEvent("create", controller, obj, err)
	return desired, nil
}

func (c *realGenericControlInterface) Delete(controller, obj runtime.Object) error {
	err := c.client.Delete(context.TODO(), obj)
	c.RecordControllerEvent("delete", controller, obj, err)
	return err
}

// RecordControllerEvent is a generic method to record event for controller
func (c *realGenericControlInterface) RecordControllerEvent(verb string, controller runtime.Object, obj runtime.Object, err error) {
	controllerKind := controller.GetObjectKind().GroupVersionKind().Kind
	var controllerName string
	if accessor, ok := controller.(metav1.ObjectMetaAccessor); ok {
		controllerName = accessor.GetObjectMeta().GetName()
	}
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	var name string
	if accessor, ok := obj.(metav1.ObjectMetaAccessor); ok {
		name = accessor.GetObjectMeta().GetName()
	}
	if err == nil {
		reason := fmt.Sprintf("Successfully %s", strings.Title(verb))
		msg := fmt.Sprintf("%s %s/%s for controller %s/%s successfully",
			strings.ToLower(verb),
			kind, name,
			controllerKind, controllerName)
		c.recorder.Event(controller, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed to %s", strings.Title(verb))
		msg := fmt.Sprintf("%s %s/%s for controller %s/%s failed, error: %s",
			strings.ToLower(verb),
			kind, name,
			controllerKind, controllerName,
			err)
		c.recorder.Event(controller, corev1.EventTypeWarning, reason, msg)
	}
}

func setControllerReference(controller, obj runtime.Object) error {
	controllerMo, ok := controller.(metav1.Object)
	if !ok {
		return fmt.Errorf("%T is not a metav1.Object, cannot call setControllerReference", controller)
	}
	objMo, ok := obj.(metav1.Object)
	if !ok {
		return fmt.Errorf("%T is not a metav1.Object, cannot call setControllerReference", obj)
	}
	return controllerutil.SetControllerReference(controllerMo, objMo, scheme.Scheme)
}

// FakeGenericControl is a fake GenericControlInterface
type FakeGenericControl struct {
	FakeCli               client.Client
	control               GenericControlInterface
	createOrUpdateTracker RequestTracker
	deleteTracker         RequestTracker
}

// NewFakeGenericControl returns a FakeGenericControl
func NewFakeGenericControl(initObjects ...runtime.Object) *FakeGenericControl {
	fakeCli := fake.NewFakeClientWithScheme(scheme.Scheme, initObjects...)
	control := NewRealGenericControl(fakeCli, record.NewFakeRecorder(10))
	return &FakeGenericControl{
		fakeCli,
		control,
		RequestTracker{},
		RequestTracker{},
	}
}

func (gc *FakeGenericControl) SetCreateOrUpdateError(err error, after int) {
	gc.createOrUpdateTracker.SetError(err).SetAfter(after)
}

func (gc *FakeGenericControl) SetDeleteError(err error, after int) {
	gc.deleteTracker.SetError(err).SetAfter(after)
}

// AddObject is used to prepare the indexer for fakeGenericControl
func (gc *FakeGenericControl) AddObject(object runtime.Object) error {
	return gc.FakeCli.Create(context.TODO(), object.DeepCopyObject())
}

func (gc *FakeGenericControl) CreateOrUpdate(controller, obj runtime.Object, fn MergeFn) (runtime.Object, error) {
	defer gc.createOrUpdateTracker.Inc()
	if gc.createOrUpdateTracker.ErrorReady() {
		defer gc.createOrUpdateTracker.Reset()
		return nil, gc.createOrUpdateTracker.GetError()
	}

	return gc.control.CreateOrUpdate(controller, obj, fn)
}

func (gc *FakeGenericControl) Delete(controller, obj runtime.Object) error {
	defer gc.deleteTracker.Inc()
	if gc.deleteTracker.ErrorReady() {
		defer gc.deleteTracker.Reset()
		return gc.deleteTracker.GetError()
	}

	return gc.control.Delete(controller, obj)
}

var _ GenericControlInterface = &FakeGenericControl{}
