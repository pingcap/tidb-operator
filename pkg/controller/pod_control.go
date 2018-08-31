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
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
)

// PodControlInterface manages Pods used in TidbCluster
type PodControlInterface interface {
	// TODO change this to UpdatePod
	UpdateMetaInfo(*v1alpha1.TidbCluster, *corev1.Pod) error
	DeletePod(*v1alpha1.TidbCluster, *corev1.Pod) error
	UpdatePod(*v1alpha1.TidbCluster, *corev1.Pod) error
}

type realPodControl struct {
	kubeCli   kubernetes.Interface
	pdControl PDControlInterface
	podLister corelisters.PodLister
	recorder  record.EventRecorder
}

// NewRealPodControl creates a new PodControlInterface
func NewRealPodControl(
	kubeCli kubernetes.Interface,
	pdControl PDControlInterface,
	podLister corelisters.PodLister,
	recorder record.EventRecorder,
) PodControlInterface {
	return &realPodControl{
		kubeCli:   kubeCli,
		pdControl: pdControl,
		podLister: podLister,
		recorder:  recorder,
	}
}

func (rpc *realPodControl) UpdatePod(tc *v1alpha1.TidbCluster, pod *corev1.Pod) error {
	ns := tc.GetNamespace()
	podName := pod.GetName()

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, updateErr := rpc.kubeCli.CoreV1().Pods(ns).Update(pod)
		if updateErr == nil {
			glog.Infof("Pod: [%s/%s] updated successfully", ns, podName)
			return nil
		}
		glog.Errorf("failed to update Pod: [%s/%s], error: %v", ns, podName, updateErr)

		if updated, err := rpc.podLister.Pods(ns).Get(podName); err == nil {
			// make a copy so we don't mutate the shared cache
			pod = updated.DeepCopy()
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated Pod %s/%s from lister: %v", ns, podName, err))
		}

		return updateErr
	})
	rpc.recordPodEvent("update", tc, podName, err)
	return err
}

func (rpc *realPodControl) UpdateMetaInfo(tc *v1alpha1.TidbCluster, pod *corev1.Pod) error {
	ns := pod.GetNamespace()
	podName := pod.GetName()
	labels := pod.GetLabels()
	tcName := tc.GetName()
	var clusterID, storeID, memberID string
	if labels == nil {
		return fmt.Errorf("pod %s/%s has empty labels, TidbCluster: %s", ns, podName, tcName)
	}
	_, ok := labels[label.ClusterLabelKey]
	if !ok {
		return fmt.Errorf("pod %s/%s doesn't have %s label, TidbCluster: %s", ns, podName, label.ClusterLabelKey, tcName)
	}
	pdClient := rpc.pdControl.GetPDClient(tc)
	if labels[label.ClusterIDLabelKey] == "" {
		cluster, err := pdClient.GetCluster()
		if err != nil {
			return fmt.Errorf("failed to get tidb cluster info from pd, TidbCluster: %s/%s, err: %v", ns, tcName, err)
		}
		clusterID = strconv.FormatUint(cluster.Id, 10)
	}

	app := labels[label.AppLabelKey]
	switch app {
	case label.PDLabelVal:
		if labels[label.MemberIDLabelKey] == "" {
			// get member id
			members, err := pdClient.GetMembers()
			if err != nil {
				return fmt.Errorf("failed to get pd members info from pd, TidbCluster: %s/%s, err: %v", ns, tcName, err)
			}
			for _, member := range members.Members {
				if member.GetName() == podName {
					memberID = strconv.FormatUint(member.GetMemberId(), 10)
					break
				}
			}
		}
	case label.TiKVLabelVal:
		if labels[label.StoreIDLabelKey] == "" {
			// get store id
			stores, err := pdClient.GetStores()
			if err != nil {
				return fmt.Errorf("failed to get tikv stores info from pd, TidbCluster: %s/%s, err: %v", ns, tcName, err)
			}
			for _, store := range stores.Stores {
				addr := store.Store.GetAddress()
				if strings.Split(addr, ".")[0] == podName {
					storeID = strconv.FormatUint(store.Store.GetId(), 10)
					break
				}
			}
		}
	}
	if labels[label.ClusterIDLabelKey] == clusterID &&
		labels[label.MemberIDLabelKey] == memberID &&
		labels[label.StoreIDLabelKey] == storeID {
		glog.V(4).Infof("pod %s/%s already has cluster labels set, skipping. TidbCluster: %s", ns, podName, tcName)
		return nil
	}
	// labels is a pointer, modify labels will modify pod.Labels
	setIfNotEmpty(labels, label.ClusterIDLabelKey, clusterID)
	setIfNotEmpty(labels, label.MemberIDLabelKey, memberID)
	setIfNotEmpty(labels, label.StoreIDLabelKey, storeID)

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, err := rpc.kubeCli.CoreV1().Pods(ns).Update(pod)
		if err != nil {
			glog.Errorf("failed to update pod %s/%s with cluster labels %v, TidbCluster: %s, err: %v", ns, podName, labels, tcName, err)
			return err
		}
		glog.V(4).Infof("update pod %s/%s with cluster labels %v successfully, TidbCluster: %s", ns, podName, labels, tcName)
		return nil
	})
	rpc.recordPodEvent("update", tc, podName, err)
	return err
}

func (rpc *realPodControl) DeletePod(tc *v1alpha1.TidbCluster, pod *corev1.Pod) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	podName := pod.GetName()
	err := rpc.kubeCli.CoreV1().Pods(ns).Delete(podName, nil)
	if err != nil {
		glog.Errorf("failed to delete Pod: [%s/%s], TidbCluster: %s, %v", ns, podName, tcName, err)
	} else {
		glog.V(4).Infof("delete Pod: [%s/%s] successfully, TidbCluster: %s", ns, podName, tcName)
	}
	rpc.recordPodEvent("delete", tc, podName, err)
	return err
}

func (rpc *realPodControl) recordPodEvent(verb string, tc *v1alpha1.TidbCluster, podName string, err error) {
	tcName := tc.GetName()
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Pod %s in TidbCluster %s successful",
			strings.ToLower(verb), podName, tcName)
		rpc.recorder.Event(tc, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Pod %s in TidbCluster %s failed error: %s",
			strings.ToLower(verb), podName, tcName, err)
		rpc.recorder.Event(tc, corev1.EventTypeWarning, reason, msg)
	}
}

var _ PodControlInterface = &realPodControl{}

var (
	TestStoreId     string = "000"
	TestMemberId    string = "111"
	TestClusterId   string = "222"
	TestAppName     string = "tikv"
	TestPodName     string = "pod-1"
	TestOwnerName   string = "tidbCluster"
	TestClusterName string = "test"
)

// FakePodControl is a fake PodControlInterface
type FakePodControl struct {
	PodIndexer        cache.Indexer
	updatePodTracker  requestTracker
	deletePodTracker  requestTracker
	getClusterTracker requestTracker
	getMemberTracker  requestTracker
	getStoreTracker   requestTracker
}

// NewFakePodControl returns a FakePodControl
func NewFakePodControl(podInformer coreinformers.PodInformer) *FakePodControl {
	return &FakePodControl{
		podInformer.Informer().GetIndexer(),
		requestTracker{0, nil, 0},
		requestTracker{0, nil, 0},
		requestTracker{0, nil, 0},
		requestTracker{0, nil, 0},
		requestTracker{0, nil, 0},
	}
}

// SetUpdatePodError sets the error attributes of updatePodTracker
func (fpc *FakePodControl) SetUpdatePodError(err error, after int) {
	fpc.updatePodTracker.err = err
	fpc.updatePodTracker.after = after
}

// SetDeletePodError sets the error attributes of deletePodTracker
func (fpc *FakePodControl) SetDeletePodError(err error, after int) {
	fpc.deletePodTracker.err = err
	fpc.deletePodTracker.after = after
}

// SetGetClusterError sets the error attributes of getClusterTracker
func (fpc *FakePodControl) SetGetClusterError(err error, after int) {
	fpc.getClusterTracker.err = err
	fpc.getClusterTracker.after = after
}

// SetGetMemberError sets the error attributes of getMemberTracker
func (fpc *FakePodControl) SetGetMemberError(err error, after int) {
	fpc.getMemberTracker.err = err
	fpc.getMemberTracker.after = after
}

// SetGetStoreError sets the error attributes of getStoreTracker
func (fpc *FakePodControl) SetGetStoreError(err error, after int) {
	fpc.getStoreTracker.err = err
	fpc.getStoreTracker.after = after
}

// UpdateMetaInfo update the meta info of Pod
func (fpc *FakePodControl) UpdateMetaInfo(tc *v1alpha1.TidbCluster, pod *corev1.Pod) error {
	defer fpc.updatePodTracker.inc()
	if fpc.updatePodTracker.errorReady() {
		defer fpc.updatePodTracker.reset()
		return fpc.updatePodTracker.err
	}

	defer fpc.getClusterTracker.inc()
	if fpc.getClusterTracker.errorReady() {
		defer fpc.getClusterTracker.reset()
		return fpc.getClusterTracker.err
	}

	defer fpc.getMemberTracker.inc()
	if fpc.getMemberTracker.errorReady() {
		defer fpc.getMemberTracker.reset()
		return fpc.getMemberTracker.err
	}

	defer fpc.getStoreTracker.inc()
	if fpc.getStoreTracker.errorReady() {
		defer fpc.getStoreTracker.reset()
		return fpc.getStoreTracker.err
	}

	setIfNotEmpty(pod.Labels, label.AppLabelKey, TestAppName)
	setIfNotEmpty(pod.Labels, label.OwnerLabelKey, TestOwnerName)
	setIfNotEmpty(pod.Labels, label.ClusterLabelKey, TestClusterName)
	setIfNotEmpty(pod.Labels, label.ClusterIDLabelKey, TestClusterId)
	setIfNotEmpty(pod.Labels, label.MemberIDLabelKey, TestMemberId)
	setIfNotEmpty(pod.Labels, label.StoreIDLabelKey, TestStoreId)
	return fpc.PodIndexer.Update(pod)
}

func (fpc *FakePodControl) DeletePod(tc *v1alpha1.TidbCluster, pod *corev1.Pod) error {
	defer fpc.deletePodTracker.inc()
	if fpc.deletePodTracker.errorReady() {
		defer fpc.deletePodTracker.reset()
		return fpc.deletePodTracker.err
	}

	return fpc.PodIndexer.Delete(pod)
}

func (fpc *FakePodControl) UpdatePod(tc *v1alpha1.TidbCluster, pod *corev1.Pod) error {
	defer fpc.updatePodTracker.inc()
	if fpc.updatePodTracker.errorReady() {
		defer fpc.updatePodTracker.reset()
		return fpc.updatePodTracker.err
	}

	return fpc.PodIndexer.Update(pod)
}

var _ PodControlInterface = &FakePodControl{}
