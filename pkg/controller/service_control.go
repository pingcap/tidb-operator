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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	tcinformers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	v1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	CreateService(*v1alpha1.TidbCluster, *corev1.Service) error
	UpdateService(*v1alpha1.TidbCluster, *corev1.Service) (*corev1.Service, error)
	DeleteService(*v1alpha1.TidbCluster, *corev1.Service) error
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

func (sc *realServiceControl) CreateService(tc *v1alpha1.TidbCluster, svc *corev1.Service) error {
	_, err := sc.kubeCli.CoreV1().Services(tc.Namespace).Create(svc)
	sc.recordServiceEvent("create", tc, svc, err)
	return err
}

func (sc *realServiceControl) UpdateService(tc *v1alpha1.TidbCluster, svc *corev1.Service) (*corev1.Service, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	svcName := svc.GetName()
	svcSpec := svc.Spec.DeepCopy()

	var updateSvc *corev1.Service
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var updateErr error
		updateSvc, updateErr = sc.kubeCli.CoreV1().Services(ns).Update(svc)
		if updateErr == nil {
			klog.Infof("update Service: [%s/%s] successfully, TidbCluster: %s", ns, svcName, tcName)
			return nil
		}

		if updated, err := sc.svcLister.Services(tc.Namespace).Get(svcName); err != nil {
			utilruntime.HandleError(fmt.Errorf("error getting updated Service %s/%s from lister: %v", ns, svcName, err))
		} else {
			svc = updated.DeepCopy()
			svc.Spec = *svcSpec
		}

		return updateErr
	})
	return updateSvc, err
}

func (sc *realServiceControl) DeleteService(tc *v1alpha1.TidbCluster, svc *corev1.Service) error {
	err := sc.kubeCli.CoreV1().Services(tc.Namespace).Delete(svc.Name, nil)
	sc.recordServiceEvent("delete", tc, svc, err)
	return err
}

func (sc *realServiceControl) recordServiceEvent(verb string, tc *v1alpha1.TidbCluster, svc *corev1.Service, err error) {
	tcName := tc.GetName()
	svcName := svc.GetName()
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in TidbCluster %s successful",
			strings.ToLower(verb), svcName, tcName)
		sc.recorder.Event(tc, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in TidbCluster %s failed error: %s",
			strings.ToLower(verb), svcName, tcName, err)
		sc.recorder.Event(tc, corev1.EventTypeWarning, reason, msg)
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
func (ssc *FakeServiceControl) SetCreateServiceError(err error, after int) {
	ssc.createServiceTracker.SetError(err).SetAfter(after)
}

// SetUpdateServiceError sets the error attributes of updateServiceTracker
func (ssc *FakeServiceControl) SetUpdateServiceError(err error, after int) {
	ssc.updateServiceTracker.SetError(err).SetAfter(after)
}

// SetDeleteServiceError sets the error attributes of deleteServiceTracker
func (ssc *FakeServiceControl) SetDeleteServiceError(err error, after int) {
	ssc.deleteStatefulSetTracker.SetError(err).SetAfter(after)
}

// CreateService adds the service to SvcIndexer
func (ssc *FakeServiceControl) CreateService(_ *v1alpha1.TidbCluster, svc *corev1.Service) error {
	defer ssc.createServiceTracker.Inc()
	if ssc.createServiceTracker.ErrorReady() {
		defer ssc.createServiceTracker.Reset()
		return ssc.createServiceTracker.GetError()
	}

	err := ssc.SvcIndexer.Add(svc)
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
		return ssc.EpsIndexer.Add(eps)
	}
	return nil
}

// UpdateService updates the service of SvcIndexer
func (ssc *FakeServiceControl) UpdateService(_ *v1alpha1.TidbCluster, svc *corev1.Service) (*corev1.Service, error) {
	defer ssc.updateServiceTracker.Inc()
	if ssc.updateServiceTracker.ErrorReady() {
		defer ssc.updateServiceTracker.Reset()
		return nil, ssc.updateServiceTracker.GetError()
	}

	if svc.Spec.Selector != nil {
		eps := &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svc.Name,
				Namespace: svc.Namespace,
			},
		}
		err := ssc.EpsIndexer.Update(eps)
		if err != nil {
			return nil, err
		}
	}
	return svc, ssc.SvcIndexer.Update(svc)
}

// DeleteService deletes the service of SvcIndexer
func (ssc *FakeServiceControl) DeleteService(_ *v1alpha1.TidbCluster, _ *corev1.Service) error {
	return nil
}

var _ ServiceControlInterface = &FakeServiceControl{}
