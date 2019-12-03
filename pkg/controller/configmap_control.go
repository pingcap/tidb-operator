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
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// ConfigMapControlInterface manages configmaps used by TiDB clusters
type ConfigMapControlInterface interface {
	CreateConfigMap(*v1alpha1.TidbCluster, *corev1.ConfigMap) error
	UpdateConfigMap(*v1alpha1.TidbCluster, *corev1.ConfigMap) (*corev1.ConfigMap, error)
	DeleteConfigMap(*v1alpha1.TidbCluster, *corev1.ConfigMap) error
}

type realConfigMapControl struct {
	kubeCli  kubernetes.Interface
	cmLister corelisters.ConfigMapLister
	recorder record.EventRecorder
}

// NewRealSecretControl creates a new SecretControlInterface
func NewRealConfigMapControl(
	kubeCli kubernetes.Interface,
	cmLister corelisters.ConfigMapLister,
	recorder record.EventRecorder,
) ConfigMapControlInterface {
	return &realConfigMapControl{
		kubeCli:  kubeCli,
		cmLister: cmLister,
		recorder: recorder,
	}
}

func (cc *realConfigMapControl) CreateConfigMap(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) error {
	_, err := cc.kubeCli.CoreV1().ConfigMaps(tc.Namespace).Create(cm)
	cc.recordConfigMapEvent("create", tc, cm, err)
	return err
}

func (cc *realConfigMapControl) UpdateConfigMap(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	cmName := cm.GetName()
	cmData := cm.Data

	var updatedCm *corev1.ConfigMap
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var updateErr error
		updatedCm, updateErr = cc.kubeCli.CoreV1().ConfigMaps(ns).Update(cm)
		if updateErr == nil {
			klog.Infof("update ConfigMap: [%s/%s] successfully, TidbCluster: %s", ns, cmName, tcName)
			return nil
		}

		if updated, err := cc.cmLister.ConfigMaps(tc.Namespace).Get(cmName); err != nil {
			utilruntime.HandleError(fmt.Errorf("error getting updated ConfigMap %s/%s from lister: %v", ns, cmName, err))
		} else {
			cm = updated.DeepCopy()
			cm.Data = cmData
		}

		return updateErr
	})
	cc.recordConfigMapEvent("update", tc, cm, err)
	return updatedCm, err
}

func (cc *realConfigMapControl) DeleteConfigMap(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) error {
	err := cc.kubeCli.CoreV1().ConfigMaps(tc.Namespace).Delete(cm.Name, nil)
	cc.recordConfigMapEvent("delete", tc, cm, err)
	return err
}

func (cc *realConfigMapControl) recordConfigMapEvent(verb string, tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap, err error) {
	tcName := tc.GetName()
	cmName := cm.GetName()
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s ConfigMap %s in TidbCluster %s successful",
			strings.ToLower(verb), cmName, tcName)
		cc.recorder.Event(tc, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s ConfigMap %s in TidbCluster %s failed error: %s",
			strings.ToLower(verb), cmName, tcName, err)
		cc.recorder.Event(tc, corev1.EventTypeWarning, reason, msg)
	}
}

var _ ConfigMapControlInterface = &realConfigMapControl{}

// NewFakeConfigMapControl returns a FakeConfigMapControl
func NewFakeConfigMapControl(cmInformer coreinformers.ConfigMapInformer) *FakeConfigMapControl {
	return &FakeConfigMapControl{
		cmInformer.Lister(),
		cmInformer.Informer().GetIndexer(),
		RequestTracker{},
		RequestTracker{},
		RequestTracker{},
	}
}

// FakeConfigMapControl is a fake ConfigMapControlInterface
type FakeConfigMapControl struct {
	CmLister               corelisters.ConfigMapLister
	CmIndexer              cache.Indexer
	createConfigMapTracker RequestTracker
	updateConfigMapTracker RequestTracker
	deleteConfigMapTracker RequestTracker
}

// SetCreateConfigMapError sets the error attributes of createConfigMapTracker
func (cc *FakeConfigMapControl) SetCreateConfigMapError(err error, after int) {
	cc.createConfigMapTracker.SetError(err).SetAfter(after)
}

// SetUpdateConfigMapError sets the error attributes of updateConfigMapTracker
func (cc *FakeConfigMapControl) SetUpdateConfigMapError(err error, after int) {
	cc.updateConfigMapTracker.SetError(err).SetAfter(after)
}

// SetDeleteConfigMapError sets the error attributes of deleteConfigMapTracker
func (cc *FakeConfigMapControl) SetDeleteConfigMapError(err error, after int) {
	cc.deleteConfigMapTracker.SetError(err).SetAfter(after)
}

// CreateConfigMap adds the ConfigMap to ConfigMapIndexer
func (cc *FakeConfigMapControl) CreateConfigMap(_ *v1alpha1.TidbCluster, cm *corev1.ConfigMap) error {
	defer cc.createConfigMapTracker.Inc()
	if cc.createConfigMapTracker.ErrorReady() {
		defer cc.createConfigMapTracker.Reset()
		return cc.createConfigMapTracker.GetError()
	}

	err := cc.CmIndexer.Add(cm)
	if err != nil {
		return err
	}
	return nil
}

// UpdateConfigMap updates the ConfigMap of CmIndexer
func (cc *FakeConfigMapControl) UpdateConfigMap(_ *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	defer cc.updateConfigMapTracker.Inc()
	if cc.updateConfigMapTracker.ErrorReady() {
		defer cc.updateConfigMapTracker.Reset()
		return nil, cc.updateConfigMapTracker.GetError()
	}

	return cm, cc.CmIndexer.Update(cm)
}

// DeleteConfigMap deletes the ConfigMap of CmIndexer
func (cc *FakeConfigMapControl) DeleteConfigMap(_ *v1alpha1.TidbCluster, _ *corev1.ConfigMap) error {
	return nil
}

var _ ConfigMapControlInterface = &FakeConfigMapControl{}
