// Copyright 2020 PingCAP, Inc.
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

package tests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"time"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	utilstatefulset "github.com/pingcap/tidb-operator/tests/e2e/util/statefulset"
	"github.com/pingcap/tidb-operator/tests/slack"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/klog"
)

type CrdTestUtil struct {
	cli         versioned.Interface
	kubeCli     kubernetes.Interface
	tcStsGetter typedappsv1.StatefulSetsGetter
	asCli       asclientset.Interface
}

func NewCrdTestUtil(cli versioned.Interface, kubeCli kubernetes.Interface, asCli asclientset.Interface, astsEnabled bool) *CrdTestUtil {
	var tcStsGetter typedappsv1.StatefulSetsGetter
	if astsEnabled {
		tcStsGetter = helper.NewHijackClient(kubeCli, asCli).AppsV1()
	} else {
		tcStsGetter = kubeCli.AppsV1()
	}

	return &CrdTestUtil{
		cli:         cli,
		kubeCli:     kubeCli,
		tcStsGetter: tcStsGetter,
		asCli:       asCli,
	}
}

func (ctu *CrdTestUtil) GetTidbClusterOrDie(name, namespace string) *v1alpha1.TidbCluster {
	tc, err := ctu.cli.PingcapV1alpha1().TidbClusters(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		slack.NotifyAndPanic(err)
	}
	return tc
}

func (ctu *CrdTestUtil) CreateTidbClusterOrDie(tc *v1alpha1.TidbCluster) {
	_, err := ctu.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (ctu *CrdTestUtil) UpdateTidbClusterOrDie(tc *v1alpha1.TidbCluster) {
	err := wait.Poll(5*time.Second, 3*time.Minute, func() (done bool, err error) {
		latestTC, err := ctu.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		latestTC.Spec = tc.Spec
		_, err = ctu.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Update(latestTC)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (ctu *CrdTestUtil) CheckDisasterToleranceOrDie(tc *v1alpha1.TidbCluster) {
	err := ctu.CheckDisasterTolerance(tc)
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (ctu *CrdTestUtil) CheckDisasterTolerance(cluster *v1alpha1.TidbCluster) error {
	pds, err := ctu.kubeCli.CoreV1().Pods(cluster.Namespace).List(
		metav1.ListOptions{LabelSelector: labels.SelectorFromSet(
			label.New().Instance(cluster.Name).PD().Labels(),
		).String()})
	if err != nil {
		return err
	}
	err = checkPodsDisasterTolerance(pds.Items)
	if err != nil {
		return err
	}

	tikvs, err := ctu.kubeCli.CoreV1().Pods(cluster.Namespace).List(
		metav1.ListOptions{LabelSelector: labels.SelectorFromSet(
			label.New().Instance(cluster.Name).TiKV().Labels(),
		).String()})
	if err != nil {
		return err
	}
	err = checkPodsDisasterTolerance(tikvs.Items)
	if err != nil {
		return err
	}

	tidbs, err := ctu.kubeCli.CoreV1().Pods(cluster.Namespace).List(
		metav1.ListOptions{LabelSelector: labels.SelectorFromSet(
			label.New().Instance(cluster.Name).TiDB().Labels(),
		).String()})
	if err != nil {
		return err
	}
	return checkPodsDisasterTolerance(tidbs.Items)
}

func checkPodsDisasterTolerance(allPods []corev1.Pod) error {
	for _, pod := range allPods {
		if pod.Spec.Affinity == nil {
			return fmt.Errorf("the pod:[%s/%s] has not Affinity", pod.Namespace, pod.Name)
		}
		if pod.Spec.Affinity.PodAntiAffinity == nil {
			return fmt.Errorf("the pod:[%s/%s] has not Affinity.PodAntiAffinity", pod.Namespace, pod.Name)
		}
		if len(pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) == 0 {
			return fmt.Errorf("the pod:[%s/%s] has not PreferredDuringSchedulingIgnoredDuringExecution", pod.Namespace, pod.Name)
		}
		for _, prefer := range pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			if prefer.PodAffinityTerm.TopologyKey != RackLabel {
				return fmt.Errorf("the pod:[%s/%s] topology key is not %s", pod.Namespace, pod.Name, RackLabel)
			}
		}
	}
	return nil
}

func (ctu *CrdTestUtil) DeleteTidbClusterOrDie(tc *v1alpha1.TidbCluster) {
	err := ctu.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Delete(tc.Name, &metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return
		}
		slack.NotifyAndPanic(err)
	}
}

func (ctu *CrdTestUtil) WaitTidbClusterReadyOrDie(tc *v1alpha1.TidbCluster, timeout time.Duration) {
	err := ctu.WaitForTidbClusterReady(tc, timeout, 5*time.Second)
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (ctu *CrdTestUtil) WaitForTidbClusterReady(tc *v1alpha1.TidbCluster, timeout, pollInterval time.Duration) error {
	if tc == nil {
		return fmt.Errorf("tidbcluster is nil, cannot call WaitForTidbClusterReady")
	}
	return wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		var local *v1alpha1.TidbCluster
		var err error
		if local, err = ctu.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{}); err != nil {
			klog.Errorf("failed to get tidbcluster: %s/%s, %v", tc.Namespace, tc.Name, err)
			return false, nil
		}

		if b, err := ctu.pdMembersReadyFn(local); !b && err == nil {
			return false, nil
		}
		if b, err := ctu.tikvMembersReadyFn(local); !b && err == nil {
			return false, nil
		}
		if b, err := ctu.tidbMembersReadyFn(local); !b && err == nil {
			return false, nil
		}
		if tc.Spec.TiFlash != nil && tc.Spec.TiFlash.Replicas > int32(0) {
			if b, err := ctu.tiflashMembersReadyFn(local); !b && err == nil {
				return false, nil
			}
		}
		if tc.Spec.Pump != nil {
			if b, err := ctu.pumpMembersReadyFn(local); !b && err == nil {
				return false, nil
			}
		}
		return true, nil
	})
}

func (ctu *CrdTestUtil) pdMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	pdSetName := controller.PDMemberName(tcName)

	pdSet, err := ctu.tcStsGetter.StatefulSets(ns).Get(pdSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get statefulset: %s/%s, %v", ns, pdSetName, err)
		return false, nil
	}

	if pdSet.Status.CurrentRevision != pdSet.Status.UpdateRevision {
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(ctu.kubeCli, ctu.asCli), pdSet) {
		return false, nil
	}

	if tc.Status.PD.StatefulSet == nil {
		klog.Infof("tidbcluster: %s/%s .status.PD.StatefulSet is nil", ns, tcName)
		return false, nil
	}
	failureCount := len(tc.Status.PD.FailureMembers)
	replicas := tc.Spec.PD.Replicas + int32(failureCount)
	if *pdSet.Spec.Replicas != replicas {
		klog.Infof("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, pdSetName, *pdSet.Spec.Replicas, replicas)
		return false, nil
	}
	if pdSet.Status.ReadyReplicas != tc.Spec.PD.Replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, pdSetName, pdSet.Status.ReadyReplicas, tc.Spec.PD.Replicas)
		return false, nil
	}
	if len(tc.Status.PD.Members) != int(tc.Spec.PD.Replicas) {
		klog.Infof("tidbcluster: %s/%s .status.PD.Members count(%d) != %d",
			ns, tcName, len(tc.Status.PD.Members), tc.Spec.PD.Replicas)
		return false, nil
	}
	if pdSet.Status.ReadyReplicas != pdSet.Status.Replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, pdSetName, pdSet.Status.ReadyReplicas, pdSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(ctu.kubeCli, ctu.tcStsGetter, ns, tc.Name, label.PDLabelVal)
	if !found {
		klog.Infof("statefulset: %s/%s not found containers[name=pd] or pod %s-0",
			ns, pdSetName, pdSetName)
		return false, nil
	}

	if tc.PDImage() != c.Image {
		klog.Infof("statefulset: %s/%s .spec.template.spec.containers[name=pd].image(%s) != %s",
			ns, pdSetName, c.Image, tc.PDImage())
		return false, nil
	}

	for _, member := range tc.Status.PD.Members {
		if !member.Health {
			klog.Infof("tidbcluster: %s/%s pd member(%s/%s) is not health",
				ns, tcName, member.ID, member.Name)
			return false, nil
		}
	}

	pdServiceName := controller.PDMemberName(tcName)
	pdPeerServiceName := controller.PDPeerMemberName(tcName)
	if _, err := ctu.kubeCli.CoreV1().Services(ns).Get(pdServiceName, metav1.GetOptions{}); err != nil {
		klog.Errorf("failed to get service: %s/%s", ns, pdServiceName)
		return false, nil
	}
	if _, err := ctu.kubeCli.CoreV1().Services(ns).Get(pdPeerServiceName, metav1.GetOptions{}); err != nil {
		klog.Errorf("failed to get peer service: %s/%s", ns, pdPeerServiceName)
		return false, nil
	}

	return true, nil
}

func (ctu *CrdTestUtil) tikvMembersReadyFn(obj runtime.Object) (bool, error) {
	meta, ok := obj.(metav1.Object)
	if !ok {
		return false, fmt.Errorf("failed to convert to meta.Object")
	}
	name := meta.GetName()
	ns := meta.GetNamespace()
	var tikvSetName string
	if tc, ok := obj.(*v1alpha1.TidbCluster); ok {
		tikvSetName = controller.TiKVMemberName(tc.Name)
	} else if tg, ok := obj.(*v1alpha1.TiKVGroup); ok {
		tikvSetName = controller.TiKVGroupMemberName(tg.Name)
	}
	if len(tikvSetName) < 1 {
		return false, fmt.Errorf("failed to parse obj to TikvGroup or TidbCluster")
	}

	tikvSet, err := ctu.tcStsGetter.StatefulSets(ns).Get(tikvSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get statefulset: %s/%s, %v", ns, tikvSetName, err)
		return false, nil
	}

	if tikvSet.Status.CurrentRevision != tikvSet.Status.UpdateRevision {
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(ctu.kubeCli, ctu.asCli), tikvSet) {
		return false, nil
	}
	var tikvStatus v1alpha1.TiKVStatus
	var replicas int32
	var storeCounts int32
	var image string
	var stores map[string]v1alpha1.TiKVStore
	var tikvPeerServiceName string
	if tc, ok := obj.(*v1alpha1.TidbCluster); ok {
		tikvStatus = tc.Status.TiKV
		replicas = tc.Spec.TiKV.Replicas + int32(len(tc.Status.TiKV.FailureStores))
		storeCounts = int32(len(tc.Status.TiKV.Stores))
		image = tc.TiKVImage()
		stores = tc.Status.TiKV.Stores
		tikvPeerServiceName = controller.TiKVPeerMemberName(tc.GetName())
	} else if tg, ok := obj.(*v1alpha1.TiKVGroup); ok {
		tikvStatus = tg.Status.TiKVStatus
		replicas = tg.Spec.TiKVSpec.Replicas
		storeCounts = int32(len(tg.Status.Stores))
		image = tg.Spec.Image
		stores = tg.Status.TiKVStatus.Stores
		tikvPeerServiceName = controller.TiKVGroupPeerMemberName(tg.Name)
	}

	if tikvStatus.StatefulSet == nil {
		klog.Infof("%s/%s .status.StatefulSet is nil", ns, name)
		return false, nil
	}
	if *tikvSet.Spec.Replicas != replicas {
		klog.Infof("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, tikvSetName, *tikvSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tikvSet.Status.ReadyReplicas != replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, tikvSetName, tikvSet.Status.ReadyReplicas, replicas)
		return false, nil
	}
	if storeCounts != replicas {
		klog.Infof("%s/%s .status.TiKV.Stores.count(%d) != %d",
			ns, name, storeCounts, replicas)
		return false, nil
	}
	if tikvSet.Status.ReadyReplicas != tikvSet.Status.Replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, tikvSetName, tikvSet.Status.ReadyReplicas, tikvSet.Status.Replicas)
		return false, nil
	}

	c, found := getStsContainer(ctu.kubeCli, tikvSet, label.TiKVLabelVal)
	if !found {
		klog.Infof("statefulset: %s/%s not found containers[name=tikv] or pod %s-0",
			ns, tikvSetName, tikvSetName)
		return false, nil
	}

	if image != c.Image {
		klog.Infof("statefulset: %s/%s .spec.template.spec.containers[name=tikv].image(%s) != %s",
			ns, tikvSetName, c.Image, image)
		return false, nil
	}

	for _, store := range stores {
		if store.State != v1alpha1.TiKVStateUp {
			klog.Infof("%s/%s's store(%s) state != %s", ns, name, store.ID, v1alpha1.TiKVStateUp)
			return false, nil
		}
	}
	if _, err := ctu.kubeCli.CoreV1().Services(ns).Get(tikvPeerServiceName, metav1.GetOptions{}); err != nil {
		klog.Errorf("failed to get peer service: %s/%s", ns, tikvPeerServiceName)
		return false, nil
	}
	return true, nil
}

func (ctu *CrdTestUtil) tidbMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	tidbSetName := controller.TiDBMemberName(tcName)

	tidbSet, err := ctu.tcStsGetter.StatefulSets(ns).Get(tidbSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get statefulset: %s/%s, %v", ns, tidbSetName, err)
		return false, nil
	}

	if tidbSet.Status.CurrentRevision != tidbSet.Status.UpdateRevision {
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(ctu.kubeCli, ctu.asCli), tidbSet) {
		return false, nil
	}

	if tc.Status.TiDB.StatefulSet == nil {
		klog.Infof("tidbcluster: %s/%s .status.TiDB.StatefulSet is nil", ns, tcName)
		return false, nil
	}
	failureCount := len(tc.Status.TiDB.FailureMembers)
	replicas := tc.Spec.TiDB.Replicas + int32(failureCount)
	if *tidbSet.Spec.Replicas != replicas {
		klog.Infof("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, tidbSetName, *tidbSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tidbSet.Status.ReadyReplicas != tc.Spec.TiDB.Replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, tidbSetName, tidbSet.Status.ReadyReplicas, tc.Spec.TiDB.Replicas)
		return false, nil
	}
	if len(tc.Status.TiDB.Members) != int(tc.Spec.TiDB.Replicas) {
		klog.Infof("tidbcluster: %s/%s .status.TiDB.Members count(%d) != %d",
			ns, tcName, len(tc.Status.TiDB.Members), tc.Spec.TiDB.Replicas)
		return false, nil
	}
	if tidbSet.Status.ReadyReplicas != tidbSet.Status.Replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, tidbSetName, tidbSet.Status.ReadyReplicas, tidbSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(ctu.kubeCli, ctu.tcStsGetter, ns, tc.Name, label.TiDBLabelVal)
	if !found {
		klog.Infof("statefulset: %s/%s not found containers[name=tidb] or pod %s-0",
			ns, tidbSetName, tidbSetName)
		return false, nil
	}

	if tc.TiDBImage() != c.Image {
		klog.Infof("statefulset: %s/%s .spec.template.spec.containers[name=tidb].image(%s) != %s",
			ns, tidbSetName, c.Image, tc.TiDBImage())
		return false, nil
	}

	_, err = ctu.kubeCli.CoreV1().Services(ns).Get(tidbSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get service: %s/%s", ns, tidbSetName)
		return false, nil
	}
	_, err = ctu.kubeCli.CoreV1().Services(ns).Get(controller.TiDBPeerMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get peer service: %s/%s", ns, controller.TiDBPeerMemberName(tcName))
		return false, nil
	}

	return true, nil
}

func (ctu *CrdTestUtil) tiflashMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	tiflashSetName := controller.TiFlashMemberName(tcName)

	tiflashSet, err := ctu.tcStsGetter.StatefulSets(ns).Get(tiflashSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get statefulset: %s/%s, %v", ns, tiflashSetName, err)
		return false, nil
	}

	if tiflashSet.Status.CurrentRevision != tiflashSet.Status.UpdateRevision {
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(ctu.kubeCli, ctu.asCli), tiflashSet) {
		return false, nil
	}

	if tc.Status.TiFlash.StatefulSet == nil {
		klog.Infof("tidbcluster: %s/%s .status.TiFlash.StatefulSet is nil", ns, tcName)
		return false, nil
	}
	failureCount := len(tc.Status.TiFlash.FailureStores)
	replicas := tc.Spec.TiFlash.Replicas + int32(failureCount)
	if *tiflashSet.Spec.Replicas != replicas {
		klog.Infof("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, tiflashSetName, *tiflashSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tiflashSet.Status.ReadyReplicas != replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, tiflashSetName, tiflashSet.Status.ReadyReplicas, replicas)
		return false, nil
	}
	if len(tc.Status.TiFlash.Stores) != int(replicas) {
		klog.Infof("tidbcluster: %s/%s .status.TiFlash.Stores.count(%d) != %d",
			ns, tcName, len(tc.Status.TiFlash.Stores), replicas)
		return false, nil
	}
	if tiflashSet.Status.ReadyReplicas != tiflashSet.Status.Replicas {
		klog.Infof("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, tiflashSetName, tiflashSet.Status.ReadyReplicas, tiflashSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(ctu.kubeCli, ctu.tcStsGetter, ns, tc.Name, label.TiFlashLabelVal)
	if !found {
		klog.Infof("statefulset: %s/%s not found containers[name=tiflash] or pod %s-0",
			ns, tiflashSetName, tiflashSetName)
		return false, nil
	}

	if tc.TiFlashImage() != c.Image {
		klog.Infof("statefulset: %s/%s .spec.template.spec.containers[name=tiflash].image(%s) != %s",
			ns, tiflashSetName, c.Image, tc.TiFlashImage())
		return false, nil
	}

	for _, store := range tc.Status.TiFlash.Stores {
		if store.State != v1alpha1.TiKVStateUp {
			klog.Infof("tidbcluster: %s/%s's store(%s) state != %s", ns, tcName, store.ID, v1alpha1.TiKVStateUp)
			return false, nil
		}
	}

	tiflashPeerServiceName := controller.TiFlashPeerMemberName(tcName)
	if _, err := ctu.kubeCli.CoreV1().Services(ns).Get(tiflashPeerServiceName, metav1.GetOptions{}); err != nil {
		klog.Errorf("failed to get peer service: %s/%s", ns, tiflashPeerServiceName)
		return false, nil
	}

	return true, nil
}

func (ctu *CrdTestUtil) pumpMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	klog.Infof("begin to check incremental backup cluster[%s] namespace[%s]", tc.Name, tc.Namespace)
	pumpStatefulSetName := fmt.Sprintf("%s-pump", tc.Name)

	pumpStatefulSet, err := ctu.kubeCli.AppsV1().StatefulSets(tc.Namespace).Get(pumpStatefulSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get jobs %s ,%v", pumpStatefulSetName, err)
		return false, nil
	}
	if pumpStatefulSet.Status.Replicas != pumpStatefulSet.Status.ReadyReplicas {
		klog.Errorf("pump replicas is not ready, please wait ! %s ", pumpStatefulSetName)
		return false, nil
	}

	listOps := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			map[string]string{
				label.ComponentLabelKey: "pump",
				label.InstanceLabelKey:  pumpStatefulSet.Labels[label.InstanceLabelKey],
				label.NameLabelKey:      "tidb-cluster",
			},
		).String(),
	}

	pods, err := ctu.kubeCli.CoreV1().Pods(tc.Namespace).List(listOps)
	if err != nil {
		klog.Errorf("failed to get pods via pump labels %s ,%v", pumpStatefulSetName, err)
		return false, nil
	}

	for _, pod := range pods.Items {
		if !ctu.pumpHealth(tc, pod.Name) {
			klog.Errorf("some pods is not health %s", pumpStatefulSetName)
			return false, nil
		}

		klog.Info(pod.Spec.Affinity)
		if pod.Spec.Affinity == nil || pod.Spec.Affinity.PodAntiAffinity == nil || len(pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) != 1 {
			return true, fmt.Errorf("pump pod %s/%s should have affinity set", pod.Namespace, pod.Name)
		}
		klog.Info(pod.Spec.Tolerations)
		foundKey := false
		for _, tor := range pod.Spec.Tolerations {
			if tor.Key == "node-role" {
				foundKey = true
				break
			}
		}
		if !foundKey {
			return true, fmt.Errorf("pump pod %s/%s should have tolerations set", pod.Namespace, pod.Name)
		}
	}
	return true, nil
}

func (ctu *CrdTestUtil) pumpHealth(tc *v1alpha1.TidbCluster, podName string) bool {
	var addr string
	addr = fmt.Sprintf("%s.%s-pump.%s:8250", podName, tc.Name, tc.Namespace)
	pumpHealthURL := fmt.Sprintf("http://%s/status", addr)
	res, err := http.Get(pumpHealthURL)
	if err != nil {
		klog.Errorf("cluster:[%s] call %s failed,error:%v", tc.Name, pumpHealthURL, err)
		return false
	}
	if res.StatusCode >= 400 {
		klog.Errorf("Error response %v", res.StatusCode)
		return false
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		klog.Errorf("cluster:[%s] read response body failed,error:%v", tc.Name, err)
		return false
	}
	healths := pumpStatus{}
	err = json.Unmarshal(body, &healths)
	if err != nil {
		klog.Errorf("cluster:[%s] unmarshal failed,error:%v", tc.Name, err)
		return false
	}
	for _, status := range healths.StatusMap {
		if status.State != "online" {
			klog.Errorf("cluster:[%s] pump's state is not online", tc.Name)
			return false
		}
	}
	return true
}

func (ctu *CrdTestUtil) CleanResourcesOrDie(resource, namespace string) {
	cmd := fmt.Sprintf("kubectl delete %s --all -n %s", resource, namespace)
	data, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%v, resp: %s", err, data)
		slack.NotifyAndPanic(err)
	}
}

func (ctu *CrdTestUtil) CreateSecretOrDie(secret *corev1.Secret) {
	_, err := ctu.kubeCli.CoreV1().Secrets(secret.Namespace).Create(secret)
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (ctu *CrdTestUtil) WaitForTiKVGroupReady(tg *v1alpha1.TiKVGroup, timeout, pollInterval time.Duration) error {
	if tg == nil {
		return fmt.Errorf("tikvgroup is nil, cannot call WaitForTiKVGroupReady")
	}
	err := wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		var local *v1alpha1.TiKVGroup
		var err error
		if local, err = ctu.cli.PingcapV1alpha1().TiKVGroups(tg.Namespace).Get(tg.Name, metav1.GetOptions{}); err != nil {
			klog.Errorf("failed to get tikvgroup: %s/%s, %v", tg.Namespace, tg.Name, err)
			return false, nil
		}
		if b, err := ctu.tikvMembersReadyFn(local); !b && err == nil {
			return false, nil
		}
		return true, nil
	})
	return err
}
