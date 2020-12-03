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
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// StatefulSetControlInterface defines the interface that uses to create, update, and delete StatefulSets,
type StatefulSetControlInterface interface {
	// CreateStatefulSet creates a StatefulSet in a TidbCluster.
	CreateStatefulSet(*v1alpha1.TidbCluster, *apps.StatefulSet) error
	// UpdateStatefulSet updates a StatefulSet in a TidbCluster.
	UpdateStatefulSet(*v1alpha1.TidbCluster, *apps.StatefulSet) (*apps.StatefulSet, error)
	// DeleteStatefulSet deletes a StatefulSet in a TidbCluster.
	DeleteStatefulSet(*v1alpha1.TidbCluster, *apps.StatefulSet) error
}

type realStatefulSetControl struct {
	kubeCli   kubernetes.Interface
	setLister appslisters.StatefulSetLister
	recorder  record.EventRecorder
}

// NewRealStatefuSetControl returns a StatefulSetControlInterface
func NewRealStatefuSetControl(kubeCli kubernetes.Interface, setLister appslisters.StatefulSetLister, recorder record.EventRecorder) StatefulSetControlInterface {
	return &realStatefulSetControl{kubeCli, setLister, recorder}
}

// CreateStatefulSet create a StatefulSet in a TidbCluster.
func (sc *realStatefulSetControl) CreateStatefulSet(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	_, err := sc.kubeCli.AppsV1().StatefulSets(tc.Namespace).Create(set)
	// sink already exists errors
	if apierrors.IsAlreadyExists(err) {
		return err
	}
	sc.recordStatefulSetEvent("create", tc, set, err)
	return err
}

// UpdateStatefulSet update a StatefulSet in a TidbCluster.
func (sc *realStatefulSetControl) UpdateStatefulSet(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) (*apps.StatefulSet, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	setName := set.GetName()
	setSpec := set.Spec.DeepCopy()
	var updatedSS *apps.StatefulSet

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// TODO: verify if StatefulSet identity(name, namespace, labels) matches TidbCluster
		var updateErr error
		updatedSS, updateErr = sc.kubeCli.AppsV1().StatefulSets(ns).Update(set)
		if updateErr == nil {
			klog.Infof("TidbCluster: [%s/%s]'s StatefulSet: [%s/%s] updated successfully", ns, tcName, ns, setName)
			return nil
		}
		klog.Errorf("failed to update TidbCluster: [%s/%s]'s StatefulSet: [%s/%s], error: %v", ns, tcName, ns, setName, updateErr)

		if updated, err := sc.setLister.StatefulSets(ns).Get(setName); err == nil {
			// make a copy so we don't mutate the shared cache
			set = updated.DeepCopy()
			set.Spec = *setSpec
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated StatefulSet %s/%s from lister: %v", ns, setName, err))
		}
		return updateErr
	})

	return updatedSS, err
}

// DeleteStatefulSet delete a StatefulSet in a TidbCluster.
func (sc *realStatefulSetControl) DeleteStatefulSet(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	err := sc.kubeCli.AppsV1().StatefulSets(tc.Namespace).Delete(set.Name, nil)
	sc.recordStatefulSetEvent("delete", tc, set, err)
	return err
}

func (sc *realStatefulSetControl) recordStatefulSetEvent(verb string, tc *v1alpha1.TidbCluster, set *apps.StatefulSet, err error) {
	tcName := tc.Name
	setName := set.Name
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		message := fmt.Sprintf("%s StatefulSet %s in TidbCluster %s successful",
			strings.ToLower(verb), setName, tcName)
		sc.recorder.Event(tc, corev1.EventTypeNormal, reason, message)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		message := fmt.Sprintf("%s StatefulSet %s in TidbCluster %s failed error: %s",
			strings.ToLower(verb), setName, tcName, err)
		sc.recorder.Event(tc, corev1.EventTypeWarning, reason, message)
	}
}

var _ StatefulSetControlInterface = &realStatefulSetControl{}

// FakeStatefulSetControl is a fake StatefulSetControlInterface
type FakeStatefulSetControl struct {
	SetLister                appslisters.StatefulSetLister
	SetIndexer               cache.Indexer
	TcLister                 v1listers.TidbClusterLister
	TcIndexer                cache.Indexer
	createStatefulSetTracker RequestTracker
	updateStatefulSetTracker RequestTracker
	deleteStatefulSetTracker RequestTracker
	statusChange             func(set *apps.StatefulSet)
}

// NewFakeStatefulSetControl returns a FakeStatefulSetControl
func NewFakeStatefulSetControl(setInformer appsinformers.StatefulSetInformer, tcInformer tcinformers.TidbClusterInformer) *FakeStatefulSetControl {
	return &FakeStatefulSetControl{
		setInformer.Lister(),
		setInformer.Informer().GetIndexer(),
		tcInformer.Lister(),
		tcInformer.Informer().GetIndexer(),
		RequestTracker{},
		RequestTracker{},
		RequestTracker{},
		nil,
	}
}

// SetCreateStatefulSetError sets the error attributes of createStatefulSetTracker
func (ssc *FakeStatefulSetControl) SetCreateStatefulSetError(err error, after int) {
	ssc.createStatefulSetTracker.SetError(err).SetAfter(after)
}

// SetUpdateStatefulSetError sets the error attributes of updateStatefulSetTracker
func (ssc *FakeStatefulSetControl) SetUpdateStatefulSetError(err error, after int) {
	ssc.updateStatefulSetTracker.SetError(err).SetAfter(after)
}

// SetDeleteStatefulSetError sets the error attributes of deleteStatefulSetTracker
func (ssc *FakeStatefulSetControl) SetDeleteStatefulSetError(err error, after int) {
	ssc.deleteStatefulSetTracker.SetError(err).SetAfter(after)
}

func (ssc *FakeStatefulSetControl) SetStatusChange(fn func(*apps.StatefulSet)) {
	ssc.statusChange = fn
}

// CreateStatefulSet adds the statefulset to SetIndexer
func (ssc *FakeStatefulSetControl) CreateStatefulSet(_ *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	defer func() {
		ssc.createStatefulSetTracker.Inc()
		ssc.statusChange = nil
	}()

	if ssc.createStatefulSetTracker.ErrorReady() {
		defer ssc.createStatefulSetTracker.Reset()
		return ssc.createStatefulSetTracker.GetError()
	}

	if ssc.statusChange != nil {
		ssc.statusChange(set)
	}

	return ssc.SetIndexer.Add(set)
}

// UpdateStatefulSet updates the statefulset of SetIndexer
func (ssc *FakeStatefulSetControl) UpdateStatefulSet(_ *v1alpha1.TidbCluster, set *apps.StatefulSet) (*apps.StatefulSet, error) {
	defer func() {
		ssc.updateStatefulSetTracker.Inc()
		ssc.statusChange = nil
	}()

	if ssc.updateStatefulSetTracker.ErrorReady() {
		defer ssc.updateStatefulSetTracker.Reset()
		return nil, ssc.updateStatefulSetTracker.GetError()
	}

	if ssc.statusChange != nil {
		ssc.statusChange(set)
	}
	return set, ssc.SetIndexer.Update(set)
}

// DeleteStatefulSet deletes the statefulset of SetIndexer
func (ssc *FakeStatefulSetControl) DeleteStatefulSet(_ *v1alpha1.TidbCluster, _ *apps.StatefulSet) error {
	return nil
}

var _ StatefulSetControlInterface = &FakeStatefulSetControl{}
