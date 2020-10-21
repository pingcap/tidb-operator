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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
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

// PodControlInterface manages Pods used in TidbCluster
type PodControlInterface interface {
	// TODO change this to UpdatePod
	UpdateMetaInfo(*v1alpha1.TidbCluster, *corev1.Pod) (*corev1.Pod, error)
	DeletePod(runtime.Object, *corev1.Pod) error
	UpdatePod(runtime.Object, *corev1.Pod) (*corev1.Pod, error)
}

type realPodControl struct {
	kubeCli   kubernetes.Interface
	pdControl pdapi.PDControlInterface
	podLister corelisters.PodLister
	recorder  record.EventRecorder
}

// NewRealPodControl creates a new PodControlInterface
func NewRealPodControl(
	kubeCli kubernetes.Interface,
	pdControl pdapi.PDControlInterface,
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

func (c *realPodControl) UpdatePod(controller runtime.Object, pod *corev1.Pod) (*corev1.Pod, error) {
	controllerMo, ok := controller.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("%T is not a metav1.Object, cannot call setControllerReference", controller)
	}
	kind := controller.GetObjectKind().GroupVersionKind().Kind
	name := controllerMo.GetName()
	namespace := controllerMo.GetNamespace()
	podName := pod.GetName()

	labels := pod.GetLabels()
	ann := pod.GetAnnotations()

	var updatePod *corev1.Pod
	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		updatePod, updateErr = c.kubeCli.CoreV1().Pods(namespace).Update(pod)
		if updateErr == nil {
			klog.Infof("Pod: [%s/%s] updated successfully, %s: [%s/%s]", namespace, podName, kind, namespace, name)
			return nil
		}
		klog.Errorf("failed to update Pod: [%s/%s], error: %v", namespace, podName, updateErr)

		if updated, err := c.podLister.Pods(namespace).Get(podName); err == nil {
			// make a copy so we don't mutate the shared cache
			pod = updated.DeepCopy()
			pod.Labels = labels
			pod.Annotations = ann
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated Pod %s/%s from lister: %v", namespace, podName, err))
		}

		return updateErr
	})
	return updatePod, err
}

func (c *realPodControl) UpdateMetaInfo(tc *v1alpha1.TidbCluster, pod *corev1.Pod) (*corev1.Pod, error) {
	ns := pod.GetNamespace()
	podName := pod.GetName()
	labels := pod.GetLabels()
	tcName := tc.GetName()
	if labels == nil {
		return pod, fmt.Errorf("pod %s/%s has empty labels, TidbCluster: %s", ns, podName, tcName)
	}
	_, ok := labels[label.InstanceLabelKey]
	if !ok {
		return pod, fmt.Errorf("pod %s/%s doesn't have %s label, TidbCluster: %s", ns, podName, label.InstanceLabelKey, tcName)
	}
	clusterID := labels[label.ClusterIDLabelKey]
	memberID := labels[label.MemberIDLabelKey]
	storeID := labels[label.StoreIDLabelKey]

	var pdClient pdapi.PDClient
	if tc.IsHeterogeneous() {
		pdClient = c.pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.Spec.Cluster.Name, tc.IsTLSClusterEnabled())
	} else {
		pdClient = c.pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tcName, tc.IsTLSClusterEnabled())
	}

	if labels[label.ClusterIDLabelKey] == "" {
		cluster, err := pdClient.GetCluster()
		if err != nil {
			return pod, fmt.Errorf("failed to get tidb cluster info from pd, TidbCluster: %s/%s, err: %v", ns, tcName, err)
		}
		clusterID = strconv.FormatUint(cluster.Id, 10)
	}

	component := labels[label.ComponentLabelKey]
	switch component {
	case label.PDLabelVal:
		if labels[label.MemberIDLabelKey] == "" {
			// get member id
			members, err := pdClient.GetMembers()
			if err != nil {
				return pod, fmt.Errorf("failed to get pd members info from pd, TidbCluster: %s/%s, err: %v", ns, tcName, err)
			}
			for _, member := range members.Members {
				if member.GetName() == podName {
					memberID = strconv.FormatUint(member.GetMemberId(), 10)
					break
				}
			}
		}
	case label.TiKVLabelVal, label.TiFlashLabelVal:
		if labels[label.StoreIDLabelKey] == "" {
			// get store id
			stores, err := pdClient.GetStores()
			if err != nil {
				return pod, fmt.Errorf("failed to get tikv stores info from pd, TidbCluster: %s/%s, err: %v", ns, tcName, err)
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
		klog.V(4).Infof("pod %s/%s already has cluster labels set, skipping. TidbCluster: %s", ns, podName, tcName)
		return pod, nil
	}
	// labels is a pointer, modify labels will modify pod.Labels
	setIfNotEmpty(labels, label.ClusterIDLabelKey, clusterID)
	setIfNotEmpty(labels, label.MemberIDLabelKey, memberID)
	setIfNotEmpty(labels, label.StoreIDLabelKey, storeID)

	var updatePod *corev1.Pod
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var updateErr error
		updatePod, updateErr = c.kubeCli.CoreV1().Pods(ns).Update(pod)
		if updateErr == nil {
			klog.V(4).Infof("update pod %s/%s with cluster labels %v successfully, TidbCluster: %s", ns, podName, labels, tcName)
			return nil
		}
		klog.Errorf("failed to update pod %s/%s with cluster labels %v, TidbCluster: %s, err: %v", ns, podName, labels, tcName, updateErr)

		if updated, err := c.podLister.Pods(ns).Get(podName); err == nil {
			// make a copy so we don't mutate the shared cache
			pod = updated.DeepCopy()
			pod.Labels = labels
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated Pod %s/%s from lister: %v", ns, podName, err))
		}
		return updateErr
	})

	return updatePod, err
}

func (c *realPodControl) DeletePod(controller runtime.Object, pod *corev1.Pod) error {
	controllerMo, ok := controller.(metav1.Object)
	if !ok {
		return fmt.Errorf("%T is not a metav1.Object, cannot call setControllerReference", controller)
	}
	kind := controller.GetObjectKind().GroupVersionKind().Kind
	name := controllerMo.GetName()
	namespace := controllerMo.GetNamespace()

	podName := pod.GetName()
	preconditions := metav1.Preconditions{UID: &pod.UID, ResourceVersion: &pod.ResourceVersion}
	deleteOptions := metav1.DeleteOptions{Preconditions: &preconditions}
	err := c.kubeCli.CoreV1().Pods(namespace).Delete(podName, &deleteOptions)
	if err != nil {
		klog.Errorf("failed to delete Pod: [%s/%s], %s: %s, %v", namespace, podName, kind, namespace, err)
	} else {
		klog.V(4).Infof("delete Pod: [%s/%s] successfully, %s: %s", namespace, podName, kind, namespace)
	}
	c.recordPodEvent("delete", kind, name, controller, podName, err)
	return err
}

func (c *realPodControl) recordPodEvent(verb, kind, name string, object runtime.Object, podName string, err error) {
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Pod %s in %s %s successful",
			strings.ToLower(verb), podName, kind, name)
		c.recorder.Event(object, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Pod %s in %s %s failed error: %s",
			strings.ToLower(verb), podName, kind, name, err)
		c.recorder.Event(object, corev1.EventTypeWarning, reason, msg)
	}
}

var _ PodControlInterface = &realPodControl{}

var (
	TestStoreID       string = "000"
	TestMemberID      string = "111"
	TestClusterID     string = "222"
	TestName          string = "tidb-cluster"
	TestComponentName string = "tikv"
	TestPodName       string = "pod-1"
	TestManagedByName string = "tidb-operator"
	TestClusterName   string = "test"
)

// FakePodControl is a fake PodControlInterface
type FakePodControl struct {
	PodIndexer        cache.Indexer
	updatePodTracker  RequestTracker
	deletePodTracker  RequestTracker
	getClusterTracker RequestTracker
	getMemberTracker  RequestTracker
	getStoreTracker   RequestTracker
}

// NewFakePodControl returns a FakePodControl
func NewFakePodControl(podInformer coreinformers.PodInformer) *FakePodControl {
	return &FakePodControl{
		podInformer.Informer().GetIndexer(),
		RequestTracker{},
		RequestTracker{},
		RequestTracker{},
		RequestTracker{},
		RequestTracker{},
	}
}

// SetUpdatePodError sets the error attributes of updatePodTracker
func (c *FakePodControl) SetUpdatePodError(err error, after int) {
	c.updatePodTracker.SetError(err).SetAfter(after)
}

// SetDeletePodError sets the error attributes of deletePodTracker
func (c *FakePodControl) SetDeletePodError(err error, after int) {
	c.deletePodTracker.SetError(err).SetAfter(after)
}

// SetGetClusterError sets the error attributes of getClusterTracker
func (c *FakePodControl) SetGetClusterError(err error, after int) {
	c.getStoreTracker.SetError(err).SetAfter(after)
}

// SetGetMemberError sets the error attributes of getMemberTracker
func (c *FakePodControl) SetGetMemberError(err error, after int) {
	c.getStoreTracker.SetError(err).SetAfter(after)
}

// SetGetStoreError sets the error attributes of getStoreTracker
func (c *FakePodControl) SetGetStoreError(err error, after int) {
	c.getStoreTracker.SetError(err).SetAfter(after)
}

// UpdateMetaInfo update the meta info of Pod
func (c *FakePodControl) UpdateMetaInfo(_ *v1alpha1.TidbCluster, pod *corev1.Pod) (*corev1.Pod, error) {
	defer c.updatePodTracker.Inc()
	if c.updatePodTracker.ErrorReady() {
		defer c.updatePodTracker.Reset()
		return nil, c.updatePodTracker.GetError()
	}

	defer c.getClusterTracker.Inc()
	if c.getClusterTracker.ErrorReady() {
		defer c.getClusterTracker.Reset()
		return nil, c.getClusterTracker.GetError()
	}

	defer c.getMemberTracker.Inc()
	if c.getMemberTracker.ErrorReady() {
		defer c.getMemberTracker.Reset()
		return nil, c.getMemberTracker.GetError()
	}

	defer c.getStoreTracker.Inc()
	if c.getStoreTracker.ErrorReady() {
		defer c.getStoreTracker.Reset()
		return nil, c.getStoreTracker.GetError()
	}

	setIfNotEmpty(pod.Labels, label.NameLabelKey, TestName)
	setIfNotEmpty(pod.Labels, label.ManagedByLabelKey, TestManagedByName)
	setIfNotEmpty(pod.Labels, label.InstanceLabelKey, TestClusterName)
	setIfNotEmpty(pod.Labels, label.ClusterIDLabelKey, TestClusterID)
	setIfNotEmpty(pod.Labels, label.MemberIDLabelKey, TestMemberID)
	setIfNotEmpty(pod.Labels, label.StoreIDLabelKey, TestStoreID)
	return pod, c.PodIndexer.Update(pod)
}

func (c *FakePodControl) DeletePod(_ runtime.Object, pod *corev1.Pod) error {
	defer c.deletePodTracker.Inc()
	if c.deletePodTracker.ErrorReady() {
		defer c.deletePodTracker.Reset()
		return c.deletePodTracker.GetError()
	}

	return c.PodIndexer.Delete(pod)
}

func (c *FakePodControl) UpdatePod(_ runtime.Object, pod *corev1.Pod) (*corev1.Pod, error) {
	defer c.updatePodTracker.Inc()
	if c.updatePodTracker.ErrorReady() {
		defer c.updatePodTracker.Reset()
		return nil, c.updatePodTracker.GetError()
	}

	return pod, c.PodIndexer.Update(pod)
}

var _ PodControlInterface = &FakePodControl{}
