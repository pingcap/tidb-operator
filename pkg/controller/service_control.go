// Copyright 2018 PingCAP, Inc.
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
	"fmt"
	"strings"

	tcinformers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	v1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// ExternalTrafficPolicy denotes if this Service desires to route external traffic to node-local or cluster-wide endpoints.
var ExternalTrafficPolicy string

// ServiceControlInterface manages Services used in TidbCluster
type ServiceControlInterface interface {
	CreateService(runtime.Object, *corev1.Service) error
	UpdateService(runtime.Object, *corev1.Service) (*corev1.Service, error)
	DeleteService(runtime.Object, *corev1.Service) error
}

type realServiceControl struct {
	kubeCli   kubernetes.Interface
	svcLister corelisters.ServiceLister
	recorder  record.EventRecorder
}

// NewRealServiceControl creates a new ServiceControlInterface
func NewRealServiceControl(kubeCli kubernetes.Interface, svcLister corelisters.ServiceLister, recorder record.EventRecorder) ServiceControlInterface {
	return &realServiceControl{
		kubeCli,
		svcLister,
		recorder,
	}
}

func (c *realServiceControl) CreateService(controller runtime.Object, svc *corev1.Service) error {
	controllerMo, ok := controller.(metav1.Object)
	if !ok {
		return fmt.Errorf("%T is not a metav1.Object, cannot call setControllerReference", controller)
	}
	kind := controller.GetObjectKind().GroupVersionKind().Kind
	name := controllerMo.GetName()
	namespace := controllerMo.GetNamespace()
	_, err := c.kubeCli.CoreV1().Services(namespace).Create(svc)
	c.recordServiceEvent("create", name, kind, controller, svc, err)
	return err
}

func (c *realServiceControl) UpdateService(controller runtime.Object, svc *corev1.Service) (*corev1.Service, error) {
	controllerMo, ok := controller.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("%T is not a metav1.Object, cannot call setControllerReference", controller)
	}
	kind := controller.GetObjectKind().GroupVersionKind().Kind
	name := controllerMo.GetName()
	namespace := controllerMo.GetNamespace()
	svcName := svc.GetName()
	svcSpec := svc.Spec.DeepCopy()

	var updateSvc *corev1.Service
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var updateErr error
		updateSvc, updateErr = c.kubeCli.CoreV1().Services(namespace).Update(svc)
		if updateErr == nil {
			klog.Infof("update Service: [%s/%s] successfully, kind: %s, name: %s", namespace, svcName, kind, name)
			return nil
		}

		if updated, err := c.svcLister.Services(namespace).Get(svcName); err != nil {
			utilruntime.HandleError(fmt.Errorf("error getting updated Service %s/%s from lister: %v", namespace, svcName, err))
		} else {
			svc = updated.DeepCopy()
			svc.Spec = *svcSpec
		}

		return updateErr
	})
	return updateSvc, err
}

func (c *realServiceControl) DeleteService(controller runtime.Object, svc *corev1.Service) error {
	controllerMo, ok := controller.(metav1.Object)
	if !ok {
		return fmt.Errorf("%T is not a metav1.Object, cannot call setControllerReference", controller)
	}
	kind := controller.GetObjectKind().GroupVersionKind().Kind
	name := controllerMo.GetName()
	namespace := controllerMo.GetNamespace()

	err := c.kubeCli.CoreV1().Services(namespace).Delete(svc.Name, nil)
	c.recordServiceEvent("delete", name, kind, controller, svc, err)
	return err
}

func (c *realServiceControl) recordServiceEvent(verb, name, kind string, object runtime.Object, svc *corev1.Service, err error) {
	svcName := svc.GetName()
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in %s %s successful",
			strings.ToLower(verb), svcName, kind, name)
		c.recorder.Event(object, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in %s %s failed error: %s",
			strings.ToLower(verb), svcName, kind, name, err)
		c.recorder.Event(object, corev1.EventTypeWarning, reason, msg)
	}
}

var _ ServiceControlInterface = &realServiceControl{}

// FakeServiceControl is a fake ServiceControlInterface
type FakeServiceControl struct {
	SvcLister                corelisters.ServiceLister
	SvcIndexer               cache.Indexer
	EpsIndexer               cache.Indexer
	TcLister                 v1listers.TidbClusterLister
	TcIndexer                cache.Indexer
	createServiceTracker     RequestTracker
	updateServiceTracker     RequestTracker
	deleteStatefulSetTracker RequestTracker
}

// NewFakeServiceControl returns a FakeServiceControl
func NewFakeServiceControl(svcInformer coreinformers.ServiceInformer, epsInformer coreinformers.EndpointsInformer, tcInformer tcinformers.TidbClusterInformer) *FakeServiceControl {
	return &FakeServiceControl{
		svcInformer.Lister(),
		svcInformer.Informer().GetIndexer(),
		epsInformer.Informer().GetIndexer(),
		tcInformer.Lister(),
		tcInformer.Informer().GetIndexer(),
		RequestTracker{},
		RequestTracker{},
		RequestTracker{},
	}
}

// SetCreateServiceError sets the error attributes of createServiceTracker
func (c *FakeServiceControl) SetCreateServiceError(err error, after int) {
	c.createServiceTracker.SetError(err).SetAfter(after)
}

// SetUpdateServiceError sets the error attributes of updateServiceTracker
func (c *FakeServiceControl) SetUpdateServiceError(err error, after int) {
	c.updateServiceTracker.SetError(err).SetAfter(after)
}

// SetDeleteServiceError sets the error attributes of deleteServiceTracker
func (c *FakeServiceControl) SetDeleteServiceError(err error, after int) {
	c.deleteStatefulSetTracker.SetError(err).SetAfter(after)
}

// CreateService adds the service to SvcIndexer
func (c *FakeServiceControl) CreateService(_ runtime.Object, svc *corev1.Service) error {
	defer c.createServiceTracker.Inc()
	if c.createServiceTracker.ErrorReady() {
		defer c.createServiceTracker.Reset()
		return c.createServiceTracker.GetError()
	}

	err := c.SvcIndexer.Add(svc)
	if err != nil {
		return err
	}
	// add a new endpoint to indexer if svc has selector
	if svc.Spec.Selector != nil {
		eps := &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svc.Name,
				Namespace: svc.Namespace,
			},
		}
		return c.EpsIndexer.Add(eps)
	}
	return nil
}

// UpdateService updates the service of SvcIndexer
func (c *FakeServiceControl) UpdateService(_ runtime.Object, svc *corev1.Service) (*corev1.Service, error) {
	defer c.updateServiceTracker.Inc()
	if c.updateServiceTracker.ErrorReady() {
		defer c.updateServiceTracker.Reset()
		return nil, c.updateServiceTracker.GetError()
	}

	if svc.Spec.Selector != nil {
		eps := &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svc.Name,
				Namespace: svc.Namespace,
			},
		}
		err := c.EpsIndexer.Update(eps)
		if err != nil {
			return nil, err
		}
	}
	return svc, c.SvcIndexer.Update(svc)
}

// DeleteService deletes the service of SvcIndexer
func (c *FakeServiceControl) DeleteService(_ runtime.Object, _ *corev1.Service) error {
	return nil
}

var _ ServiceControlInterface = &FakeServiceControl{}
