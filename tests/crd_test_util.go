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
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/e2e/util/proxiedpdclient"
	"github.com/pingcap/tidb-operator/tests/e2e/util/proxiedtidbclient"
	utilstatefulset "github.com/pingcap/tidb-operator/tests/e2e/util/statefulset"
	tcUtil "github.com/pingcap/tidb-operator/tests/e2e/util/tidbcluster"
	"github.com/pingcap/tidb-operator/tests/slack"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/log"
	ctrlCli "sigs.k8s.io/controller-runtime/pkg/client"
)

// CrdTestUtil is a util struct for manipulating custom resources such as TidbCluster
type CrdTestUtil struct {
	// TODO: rename cli to pingcap cli
	cli         versioned.Interface
	kubeCli     kubernetes.Interface
	genericCli  ctrlCli.Client
	asCli       asclientset.Interface
	fw          portforward.PortForward
	pdControl   pdapi.PDControlInterface
	tidbControl controller.TiDBControlInterface
}

// NewCrdTestUtil make a new CrdTestUtil
func NewCrdTestUtil(
	cli versioned.Interface,
	kubeCli kubernetes.Interface,
	asCli asclientset.Interface,
	genericCli ctrlCli.Client,
	fw portforward.PortForward) *CrdTestUtil {
	ctu := &CrdTestUtil{
		cli:        cli,
		kubeCli:    kubeCli,
		genericCli: genericCli,
		asCli:      asCli,
		fw:         fw,
		pdControl:  pdapi.NewDefaultPDControl(kubeCli),
	}
	if fw != nil {
		kubeCfg, err := framework.LoadConfig()
		framework.ExpectNoError(err, "failed to load config")
		ctu.tidbControl = proxiedtidbclient.NewProxiedTiDBClient(fw, kubeCfg.TLSClientConfig.CAData)
	} else {
		ctu.tidbControl = controller.NewDefaultTiDBControl(kubeCli)
	}
	return ctu
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
	err = checkPodsAffinity(pds.Items)
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
	err = checkPodsAffinity(tikvs.Items)
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
	return checkPodsAffinity(tidbs.Items)
}

func checkPodsAffinity(allPods []corev1.Pod) error {
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

// WaitForTidbClusterReady waits for tidb components ready, or timeout
func (ctu *CrdTestUtil) WaitForTidbClusterReady(tc *v1alpha1.TidbCluster, timeout, pollInterval time.Duration) error {
	if tc == nil {
		return fmt.Errorf("tidbcluster is nil, cannot call WaitForTidbClusterReady")
	}
	return wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		var local *v1alpha1.TidbCluster
		var err error
		if local, err = ctu.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, metav1.GetOptions{}); err != nil {
			log.Logf("failed to get tidbcluster: %s/%s, %v", tc.Namespace, tc.Name, err)
			return false, nil
		}

		if b, err := ctu.pdMembersReadyFn(local); !b && err == nil {
			log.Logf("pd members are not ready for tc %q", tc.Name)
			return false, nil
		}
		log.Logf("pd members are ready for tc %q", tc.Name)

		if b, err := ctu.tikvMembersReadyFn(local); !b && err == nil {
			log.Logf("tikv members are not ready for tc %q", tc.Name)
			return false, nil
		}
		log.Logf("tikv members are ready for tc %q", tc.Name)

		if b, err := ctu.tidbMembersReadyFn(local); !b && err == nil {
			log.Logf("tidb members are not ready for tc %q", tc.Name)
			return false, nil
		}
		log.Logf("tidb members are ready for tc %q", tc.Name)

		if tc.Spec.TiFlash != nil && tc.Spec.TiFlash.Replicas > int32(0) {
			if b, err := ctu.tiflashMembersReadyFn(local); !b && err == nil {
				log.Logf("tiflash members are not ready for tc %q", tc.Name)
				return false, nil
			}
			log.Logf("tiflash members are ready for tc %q", tc.Name)
		} else {
			log.Logf("no tiflash in tc spec")
		}

		if tc.Spec.Pump != nil {
			if b, err := ctu.pumpMembersReadyFn(local); !b && err == nil {
				log.Logf("pump members are not ready for tc %q", tc.Name)
				return false, nil
			}
			log.Logf("pump members are ready for tc %q", tc.Name)
		} else {
			log.Logf("no pump in tc spec")
		}

		log.Logf("TidbCluster is ready")
		return true, nil
	})
}

// getMemberContainer gets member container
func getMemberContainer(genericCli ctrlCli.Client, namespace, tcName, component string) (*corev1.Container, bool) {
	stsName := fmt.Sprintf("%s-%s", tcName, component)
	sts, err := tcUtil.GetSts(genericCli, namespace, stsName)
	if err != nil {
		log.Logf("failed to get StatefulSet %s/%s", namespace, stsName)
		return nil, false
	}
	return getStsContainer(genericCli, sts, component)
}

func getStsContainer(genericCli ctrlCli.Client, sts *appsv1.StatefulSet, containerName string) (*corev1.Container, bool) {
	selector := labels.SelectorFromSet(sts.Spec.Selector.MatchLabels)
	podList, err := tcUtil.ListPods(genericCli, sts.Namespace, selector)
	if err != nil {
		log.Logf("failed to list Pod in ns %q with selector: %+v", sts.Namespace, selector)
		return nil, false
	}
	if len(podList.Items) == 0 {
		log.Logf("ERROR: no pods found for component %s of cluster %s/%s", containerName, sts.Namespace, sts.Name)
		return nil, false
	}
	pod := podList.Items[0]
	if len(pod.Spec.Containers) == 0 {
		log.Logf("ERROR: no containers found for component %s of cluster %s/%s", containerName, sts.Namespace, sts.Name)
		return nil, false
	}
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			return &container, true
		}
	}
	return nil, false
}

func (ctu *CrdTestUtil) pdMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	pdSetName := controller.PDMemberName(tcName)

	pdSet, err := tcUtil.GetSts(ctu.genericCli, ns, pdSetName)
	if err != nil {
		log.Logf("failed to get StatefulSet %s/%s", ns, pdSetName)
		return false, err
	}

	if pdSet.Status.CurrentRevision != pdSet.Status.UpdateRevision {
		log.Logf("pd sts .Status.CurrentRevision (%s) != .Status.UpdateRevision (%s)", pdSet.Status.CurrentRevision, pdSet.Status.UpdateRevision)
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(ctu.kubeCli, ctu.asCli), pdSet) {
		return false, nil
	}

	if tc.Status.PD.StatefulSet == nil {
		log.Logf("tidbcluster: %s/%s .status.PD.StatefulSet is nil", ns, tcName)
		return false, nil
	}
	failureCount := len(tc.Status.PD.FailureMembers)
	replicas := tc.Spec.PD.Replicas + int32(failureCount)
	if *pdSet.Spec.Replicas != replicas {
		log.Logf("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, pdSetName, *pdSet.Spec.Replicas, replicas)
		return false, nil
	}
	if pdSet.Status.ReadyReplicas != tc.Spec.PD.Replicas {
		log.Logf("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, pdSetName, pdSet.Status.ReadyReplicas, tc.Spec.PD.Replicas)
		return false, nil
	}
	if len(tc.Status.PD.Members) != int(tc.Spec.PD.Replicas) {
		log.Logf("tidbcluster: %s/%s .status.PD.Members count(%d) != %d",
			ns, tcName, len(tc.Status.PD.Members), tc.Spec.PD.Replicas)
		return false, nil
	}
	if pdSet.Status.ReadyReplicas != pdSet.Status.Replicas {
		log.Logf("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, pdSetName, pdSet.Status.ReadyReplicas, pdSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(ctu.genericCli, ns, tc.Name, label.PDLabelVal)
	if !found {
		log.Logf("statefulset: %s/%s not found containers[name=pd] or pod %s-0",
			ns, pdSetName, pdSetName)
		return false, nil
	}

	if tc.PDImage() != c.Image {
		log.Logf("statefulset: %s/%s .spec.template.spec.containers[name=pd].image(%s) != %s",
			ns, pdSetName, c.Image, tc.PDImage())
		return false, nil
	}

	for _, member := range tc.Status.PD.Members {
		if !member.Health {
			log.Logf("tidbcluster: %s/%s pd member(%s/%s) is not health",
				ns, tcName, member.ID, member.Name)
			return false, nil
		}
	}

	pdServiceName := controller.PDMemberName(tcName)
	pdPeerServiceName := controller.PDPeerMemberName(tcName)
	if _, err := ctu.kubeCli.CoreV1().Services(ns).Get(pdServiceName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get service: %s/%s", ns, pdServiceName)
		return false, nil
	}
	if _, err := ctu.kubeCli.CoreV1().Services(ns).Get(pdPeerServiceName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get peer service: %s/%s", ns, pdPeerServiceName)
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
	} else {
		return false, fmt.Errorf("failed to parse obj to TidbCluster")
	}

	tikvSet, err := tcUtil.GetSts(ctu.genericCli, ns, tikvSetName)
	if err != nil {
		log.Logf("failed to get StatefulSet %s/%s", ns, tikvSetName)
		return false, err
	}

	if tikvSet.Status.CurrentRevision != tikvSet.Status.UpdateRevision {
		log.Logf("tikv sts .Status.CurrentRevision (%s) != .Status.UpdateRevision (%s)", tikvSet.Status.CurrentRevision, tikvSet.Status.UpdateRevision)
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
	}

	if tikvStatus.StatefulSet == nil {
		log.Logf("%s/%s .status.StatefulSet is nil", ns, name)
		return false, nil
	}
	if *tikvSet.Spec.Replicas != replicas {
		log.Logf("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, tikvSetName, *tikvSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tikvSet.Status.ReadyReplicas != replicas {
		log.Logf("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, tikvSetName, tikvSet.Status.ReadyReplicas, replicas)
		return false, nil
	}
	if storeCounts != replicas {
		log.Logf("%s/%s .status.TiKV.Stores.count(%d) != %d",
			ns, name, storeCounts, replicas)
		return false, nil
	}
	if tikvSet.Status.ReadyReplicas != tikvSet.Status.Replicas {
		log.Logf("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, tikvSetName, tikvSet.Status.ReadyReplicas, tikvSet.Status.Replicas)
		return false, nil
	}

	c, found := getStsContainer(ctu.genericCli, tikvSet, label.TiKVLabelVal)
	if !found {
		log.Logf("statefulset: %s/%s not found containers[name=tikv] or pod %s-0",
			ns, tikvSetName, tikvSetName)
		return false, nil
	}

	if image != c.Image {
		log.Logf("statefulset: %s/%s .spec.template.spec.containers[name=tikv].image(%s) != %s",
			ns, tikvSetName, c.Image, image)
		return false, nil
	}

	for _, store := range stores {
		if store.State != v1alpha1.TiKVStateUp {
			log.Logf("%s/%s's store(%s) state != %s", ns, name, store.ID, v1alpha1.TiKVStateUp)
			return false, nil
		}
	}
	if _, err := ctu.kubeCli.CoreV1().Services(ns).Get(tikvPeerServiceName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get peer service: %s/%s", ns, tikvPeerServiceName)
		return false, nil
	}
	return true, nil
}

func (ctu *CrdTestUtil) tidbMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	tidbSetName := controller.TiDBMemberName(tcName)

	tidbSet, err := tcUtil.GetSts(ctu.genericCli, ns, tidbSetName)
	if err != nil {
		log.Logf("failed to get StatefulSet %s/%s", ns, tidbSetName)
		return false, err
	}

	if tidbSet.Status.CurrentRevision != tidbSet.Status.UpdateRevision {
		log.Logf("tidb sts .Status.CurrentRevision (%s) != tidb sts .Status.UpdateRevision (%s)", tidbSet.Status.CurrentRevision, tidbSet.Status.UpdateRevision)
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(ctu.kubeCli, ctu.asCli), tidbSet) {
		return false, nil
	}

	if tc.Status.TiDB.StatefulSet == nil {
		log.Logf("tidbcluster: %s/%s .status.TiDB.StatefulSet is nil", ns, tcName)
		return false, nil
	}
	failureCount := len(tc.Status.TiDB.FailureMembers)
	replicas := tc.Spec.TiDB.Replicas + int32(failureCount)
	if *tidbSet.Spec.Replicas != replicas {
		log.Logf("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, tidbSetName, *tidbSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tidbSet.Status.ReadyReplicas != tc.Spec.TiDB.Replicas {
		log.Logf("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, tidbSetName, tidbSet.Status.ReadyReplicas, tc.Spec.TiDB.Replicas)
		return false, nil
	}
	if len(tc.Status.TiDB.Members) != int(tc.Spec.TiDB.Replicas) {
		log.Logf("tidbcluster: %s/%s .status.TiDB.Members count(%d) != %d",
			ns, tcName, len(tc.Status.TiDB.Members), tc.Spec.TiDB.Replicas)
		return false, nil
	}
	if tidbSet.Status.ReadyReplicas != tidbSet.Status.Replicas {
		log.Logf("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, tidbSetName, tidbSet.Status.ReadyReplicas, tidbSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(ctu.genericCli, ns, tc.Name, label.TiDBLabelVal)
	if !found {
		log.Logf("statefulset: %s/%s not found containers[name=tidb] or pod %s-0",
			ns, tidbSetName, tidbSetName)
		return false, nil
	}

	if tc.TiDBImage() != c.Image {
		log.Logf("statefulset: %s/%s .spec.template.spec.containers[name=tidb].image(%s) != %s",
			ns, tidbSetName, c.Image, tc.TiDBImage())
		return false, nil
	}

	_, err = ctu.kubeCli.CoreV1().Services(ns).Get(tidbSetName, metav1.GetOptions{})
	if err != nil {
		log.Logf("failed to get service: %s/%s", ns, tidbSetName)
		return false, nil
	}
	_, err = ctu.kubeCli.CoreV1().Services(ns).Get(controller.TiDBPeerMemberName(tcName), metav1.GetOptions{})
	if err != nil {
		log.Logf("failed to get peer service: %s/%s", ns, controller.TiDBPeerMemberName(tcName))
		return false, nil
	}

	return true, nil
}

func (ctu *CrdTestUtil) tiflashMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	tiflashSetName := controller.TiFlashMemberName(tcName)

	tiflashSet, err := tcUtil.GetSts(ctu.genericCli, ns, tiflashSetName)
	if err != nil {
		log.Logf("failed to get StatefulSet %s/%s", ns, tiflashSetName)
		return false, err
	}

	if tiflashSet.Status.CurrentRevision != tiflashSet.Status.UpdateRevision {
		log.Logf("tiflash sts .Status.CurrentRevision (%s) != .Status.UpdateRevision (%s)", tiflashSet.Status.CurrentRevision, tiflashSet.Status.UpdateRevision)
		return false, nil
	}

	if !utilstatefulset.IsAllDesiredPodsRunningAndReady(helper.NewHijackClient(ctu.kubeCli, ctu.asCli), tiflashSet) {
		return false, nil
	}

	if tc.Status.TiFlash.StatefulSet == nil {
		log.Logf("tidbcluster: %s/%s .status.TiFlash.StatefulSet is nil", ns, tcName)
		return false, nil
	}
	failureCount := len(tc.Status.TiFlash.FailureStores)
	replicas := tc.Spec.TiFlash.Replicas + int32(failureCount)
	if *tiflashSet.Spec.Replicas != replicas {
		log.Logf("statefulset: %s/%s .spec.Replicas(%d) != %d",
			ns, tiflashSetName, *tiflashSet.Spec.Replicas, replicas)
		return false, nil
	}
	if tiflashSet.Status.ReadyReplicas != replicas {
		log.Logf("statefulset: %s/%s .status.ReadyReplicas(%d) != %d",
			ns, tiflashSetName, tiflashSet.Status.ReadyReplicas, replicas)
		return false, nil
	}
	if len(tc.Status.TiFlash.Stores) != int(replicas) {
		log.Logf("tidbcluster: %s/%s .status.TiFlash.Stores.count(%d) != %d",
			ns, tcName, len(tc.Status.TiFlash.Stores), replicas)
		return false, nil
	}
	if tiflashSet.Status.ReadyReplicas != tiflashSet.Status.Replicas {
		log.Logf("statefulset: %s/%s .status.ReadyReplicas(%d) != .status.Replicas(%d)",
			ns, tiflashSetName, tiflashSet.Status.ReadyReplicas, tiflashSet.Status.Replicas)
		return false, nil
	}

	c, found := getMemberContainer(ctu.genericCli, ns, tc.Name, label.TiFlashLabelVal)
	if !found {
		log.Logf("statefulset: %s/%s not found containers[name=tiflash] or pod %s-0",
			ns, tiflashSetName, tiflashSetName)
		return false, nil
	}

	if tc.TiFlashImage() != c.Image {
		log.Logf("statefulset: %s/%s .spec.template.spec.containers[name=tiflash].image(%s) != %s",
			ns, tiflashSetName, c.Image, tc.TiFlashImage())
		return false, nil
	}

	for _, store := range tc.Status.TiFlash.Stores {
		if store.State != v1alpha1.TiKVStateUp {
			log.Logf("tidbcluster: %s/%s's store(%s) state != %s", ns, tcName, store.ID, v1alpha1.TiKVStateUp)
			return false, nil
		}
	}

	tiflashPeerServiceName := controller.TiFlashPeerMemberName(tcName)
	if _, err := ctu.kubeCli.CoreV1().Services(ns).Get(tiflashPeerServiceName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get peer service: %s/%s", ns, tiflashPeerServiceName)
		return false, nil
	}

	return true, nil
}

func (ctu *CrdTestUtil) pumpMembersReadyFn(tc *v1alpha1.TidbCluster) (bool, error) {
	log.Logf("begin to check incremental backup cluster[%s] namespace[%s]", tc.Name, tc.Namespace)
	pumpStatefulSetName := fmt.Sprintf("%s-pump", tc.Name)

	pumpStatefulSet, err := ctu.kubeCli.AppsV1().StatefulSets(tc.Namespace).Get(pumpStatefulSetName, metav1.GetOptions{})
	if err != nil {
		log.Logf("failed to get jobs %s ,%v", pumpStatefulSetName, err)
		return false, nil
	}
	if pumpStatefulSet.Status.Replicas != pumpStatefulSet.Status.ReadyReplicas {
		log.Logf("pump replicas is not ready, please wait ! %s ", pumpStatefulSetName)
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
		log.Logf("failed to get pods via pump labels %s ,%v", pumpStatefulSetName, err)
		return false, nil
	}

	for _, pod := range pods.Items {
		if !ctu.pumpHealth(tc, pod.Name) {
			log.Logf("some pods is not health %s", pumpStatefulSetName)
			return false, nil
		}

		log.Logf("pod.Spec.Affinity: %v", pod.Spec.Affinity)
		if pod.Spec.Affinity == nil || pod.Spec.Affinity.PodAntiAffinity == nil || len(pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) != 1 {
			return true, fmt.Errorf("pump pod %s/%s should have affinity set", pod.Namespace, pod.Name)
		}
		log.Logf("pod.Spec.Tolerations: %v", pod.Spec.Tolerations)
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
	addr := fmt.Sprintf("%s.%s-pump.%s:8250", podName, tc.Name, tc.Namespace)
	pumpHealthURL := fmt.Sprintf("http://%s/status", addr)
	res, err := http.Get(pumpHealthURL)
	if err != nil {
		log.Logf("cluster:[%s] call %s failed,error:%v", tc.Name, pumpHealthURL, err)
		return false
	}
	if res.StatusCode >= 400 {
		log.Logf("Error response %v", res.StatusCode)
		return false
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Logf("cluster:[%s] read response body failed,error:%v", tc.Name, err)
		return false
	}
	healths := pumpStatus{}
	err = json.Unmarshal(body, &healths)
	if err != nil {
		log.Logf("cluster:[%s] unmarshal failed,error:%v", tc.Name, err)
		return false
	}
	for _, status := range healths.StatusMap {
		if status.State != "online" {
			log.Logf("cluster:[%s] pump's state is not online", tc.Name)
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

// TODO: re-evaluate this, because it spans about 700 lines including all sub functions!
func (ctu *CrdTestUtil) CheckTidbClusterStatus(info *TidbClusterConfig) error {
	log.Logf("checking tidb cluster [%s/%s] status", info.Namespace, info.ClusterName)
	if info.Clustrer != nil {
		return ctu.WaitForTidbClusterReady(info.Clustrer, 120*time.Minute, 1*time.Minute)
	}

	ns := info.Namespace
	tcName := info.ClusterName
	// TODO: remove redundant checks already in WaitForTidbClusterReady
	if err := wait.Poll(DefaultPollInterval, 10*time.Minute, func() (bool, error) {
		var tc *v1alpha1.TidbCluster
		var err error
		if tc, err = ctu.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{}); err != nil {
			log.Logf("failed to get tidbcluster: %s/%s, %v", ns, tcName, err)
			return false, nil
		}

		if b, err := ctu.pdMembersReadyFn(tc); !b && err == nil {
			return false, nil
		}
		if b, err := ctu.tikvMembersReadyFn(tc); !b && err == nil {
			return false, nil
		}

		log.Logf("check tidb cluster begin tidbMembersReadyFn")
		if b, err := ctu.tidbMembersReadyFn(tc); !b && err == nil {
			return false, nil
		}

		log.Logf("check tidb cluster begin reclaimPolicySyncFn")
		if b, err := ctu.reclaimPolicySyncFn(tc); !b && err == nil {
			return false, nil
		}

		log.Logf("check tidb cluster begin metaSyncFn")
		if b, err := ctu.metaSyncFn(tc); !b && err == nil {
			return false, nil
		} else if err != nil {
			log.Logf("%v", err)
			return false, nil
		}

		log.Logf("check tidb cluster begin schedulerHAFn")
		if b, err := ctu.schedulerHAFn(tc); !b && err == nil {
			return false, nil
		}

		log.Logf("check all pd and tikv instances have not pod scheduling annotation")
		if info.OperatorTag != "v1.0.0" {
			if b, err := ctu.podsScheduleAnnHaveDeleted(tc); !b && err == nil {
				return false, nil
			}
		}

		log.Logf("check store labels")
		if b, err := ctu.storeLabelsIsSet(tc, info.TopologyKey); !b && err == nil {
			return false, nil
		} else if err != nil {
			return false, err
		}

		log.Logf("check tidb cluster begin passwordIsSet")
		if b, err := ctu.passwordIsSet(info); !b && err == nil {
			return false, nil
		}

		if info.Monitor {
			log.Logf("check tidb monitor normal")
			if b, err := ctu.monitorNormal(info); !b && err == nil {
				return false, nil
			}
		}
		if info.EnableConfigMapRollout {
			log.Logf("check tidb cluster configuration synced")
			if b, err := ctu.checkTidbClusterConfigUpdated(tc, info); !b && err == nil {
				return false, nil
			}
		}
		if info.EnablePVReclaim {
			log.Logf("check reclaim pvs success when scale in pd or tikv")
			if b, err := ctu.checkReclaimPVSuccess(tc); !b && err == nil {
				return false, nil
			}
		}
		return true, nil
	}); err != nil {
		log.Logf("ERROR: check tidb cluster status failed: %s", err.Error())
		return fmt.Errorf("failed to waiting for tidbcluster %s/%s ready in 120 minutes", ns, tcName)
	}

	return nil
}

func (ctu *CrdTestUtil) CheckTidbClusterStatusOrDie(info *TidbClusterConfig) {
	if err := ctu.CheckTidbClusterStatus(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (ctu *CrdTestUtil) reclaimPolicySyncFn(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Labels(),
		).String(),
	}
	var pvcList *corev1.PersistentVolumeClaimList
	var err error
	if pvcList, err = ctu.kubeCli.CoreV1().PersistentVolumeClaims(ns).List(listOptions); err != nil {
		log.Logf("failed to list pvs for tidbcluster %s/%s, %v", ns, tcName, err)
		return false, nil
	}

	for _, pvc := range pvcList.Items {
		pvName := pvc.Spec.VolumeName
		if pv, err := ctu.kubeCli.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{}); err != nil {
			log.Logf("failed to get pv: %s, error: %v", pvName, err)
			return false, nil
		} else if pv.Spec.PersistentVolumeReclaimPolicy != *tc.Spec.PVReclaimPolicy {
			log.Logf("pv: %s's reclaimPolicy is not Retain", pvName)
			return false, nil
		}
	}

	return true, nil
}

func (ctu *CrdTestUtil) getPDClient(tc *v1alpha1.TidbCluster) (pdapi.PDClient, context.CancelFunc, error) {
	if ctu.fw != nil {
		return proxiedpdclient.NewProxiedPDClientFromTidbCluster(ctu.kubeCli, ctu.fw, tc)
	}
	return controller.GetPDClient(ctu.pdControl, tc), dummyCancel, nil
}

func (ctu *CrdTestUtil) getTiDBDSN(ns, tcName, databaseName, password string) (string, context.CancelFunc, error) {
	if ctu.fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(ctu.fw, ns, fmt.Sprintf("svc/%s", controller.TiDBMemberName(tcName)), 4000)
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("root:%s@(%s:%d)/%s?charset=utf8", password, localHost, localPort, databaseName), cancel, nil
	}
	return fmt.Sprintf("root:%s@(%s-tidb.%s:4000)/%s?charset=utf8", password, tcName, ns, databaseName), dummyCancel, nil
}

func (ctu *CrdTestUtil) metaSyncFn(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	pdClient, cancel, err := ctu.getPDClient(tc)
	if err != nil {
		log.Logf("failed to create external PD client for tidb cluster %q: %v", tc.GetName(), err)
		return false, nil
	}
	defer cancel()
	var cluster *metapb.Cluster
	if cluster, err = pdClient.GetCluster(); err != nil {
		log.Logf("failed to get cluster from pdControl: %s/%s, error: %v", ns, tcName, err)
		return false, nil
	}

	clusterID := strconv.FormatUint(cluster.Id, 10)
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Labels(),
		).String(),
	}

	var podList *corev1.PodList
	if podList, err = ctu.kubeCli.CoreV1().Pods(ns).List(listOptions); err != nil {
		log.Logf("failed to list pods for tidbcluster %s/%s, %v", ns, tcName, err)
		return false, nil
	}

outerLoop:
	for _, pod := range podList.Items {
		podName := pod.GetName()
		if pod.Labels[label.ClusterIDLabelKey] != clusterID {
			log.Logf("tidbcluster %s/%s's pod %s's label %s not equals %s ",
				ns, tcName, podName, label.ClusterIDLabelKey, clusterID)
			return false, nil
		}

		component := pod.Labels[label.ComponentLabelKey]
		switch component {
		case label.PDLabelVal:
			var memberID string
			members, err := pdClient.GetMembers()
			if err != nil {
				log.Logf("failed to get members for tidbcluster %s/%s, %v", ns, tcName, err)
				return false, nil
			}
			for _, member := range members.Members {
				if member.Name == podName {
					memberID = strconv.FormatUint(member.GetMemberId(), 10)
					break
				}
			}
			if memberID == "" {
				log.Logf("tidbcluster: %s/%s's pod %s label [%s] is empty",
					ns, tcName, podName, label.MemberIDLabelKey)
				return false, nil
			}
			if pod.Labels[label.MemberIDLabelKey] != memberID {
				return false, fmt.Errorf("tidbcluster: %s/%s's pod %s label [%s] not equals %s",
					ns, tcName, podName, label.MemberIDLabelKey, memberID)
			}
		case label.TiKVLabelVal:
			var storeID string
			stores, err := pdClient.GetStores()
			if err != nil {
				log.Logf("failed to get stores for tidbcluster %s/%s, %v", ns, tcName, err)
				return false, nil
			}
			for _, store := range stores.Stores {
				addr := store.Store.GetAddress()
				if strings.Split(addr, ".")[0] == podName {
					storeID = strconv.FormatUint(store.Store.GetId(), 10)
					break
				}
			}
			if storeID == "" {
				log.Logf("tidbcluster: %s/%s's pod %s label [%s] is empty",
					tc.GetNamespace(), tc.GetName(), podName, label.StoreIDLabelKey)
				return false, nil
			}
			if pod.Labels[label.StoreIDLabelKey] != storeID {
				return false, fmt.Errorf("tidbcluster: %s/%s's pod %s label [%s] not equals %s",
					ns, tcName, podName, label.StoreIDLabelKey, storeID)
			}
		case label.TiDBLabelVal:
			continue outerLoop
		default:
			continue outerLoop
		}

		var pvcName string
		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				pvcName = vol.PersistentVolumeClaim.ClaimName
				break
			}
		}
		if pvcName == "" {
			return false, fmt.Errorf("pod: %s/%s's pvcName is empty", ns, podName)
		}

		var pvc *corev1.PersistentVolumeClaim
		if pvc, err = ctu.kubeCli.CoreV1().PersistentVolumeClaims(ns).Get(pvcName, metav1.GetOptions{}); err != nil {
			log.Logf("failed to get pvc %s/%s for pod %s/%s", ns, pvcName, ns, podName)
			return false, nil
		}
		if pvc.Labels[label.ClusterIDLabelKey] != clusterID {
			return false, fmt.Errorf("tidbcluster: %s/%s's pvc %s label [%s] not equals %s ",
				ns, tcName, pvcName, label.ClusterIDLabelKey, clusterID)
		}
		if pvc.Labels[label.MemberIDLabelKey] != pod.Labels[label.MemberIDLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pvc %s label [%s=%s] not equals pod lablel [%s=%s]",
				ns, tcName, pvcName,
				label.MemberIDLabelKey, pvc.Labels[label.MemberIDLabelKey],
				label.MemberIDLabelKey, pod.Labels[label.MemberIDLabelKey])
		}
		if pvc.Labels[label.StoreIDLabelKey] != pod.Labels[label.StoreIDLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pvc %s label[%s=%s] not equals pod lable[%s=%s]",
				ns, tcName, pvcName,
				label.StoreIDLabelKey, pvc.Labels[label.StoreIDLabelKey],
				label.StoreIDLabelKey, pod.Labels[label.StoreIDLabelKey])
		}
		if pvc.Annotations[label.AnnPodNameKey] != podName {
			return false, fmt.Errorf("tidbcluster: %s/%s's pvc %s annotations [%s] not equals podName: %s",
				ns, tcName, pvcName, label.AnnPodNameKey, podName)
		}

		pvName := pvc.Spec.VolumeName
		var pv *corev1.PersistentVolume
		if pv, err = ctu.kubeCli.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{}); err != nil {
			log.Logf("failed to get pv for pvc %s/%s, %v", ns, pvcName, err)
			return false, nil
		}
		if pv.Labels[label.NamespaceLabelKey] != ns {
			return false, fmt.Errorf("tidbcluster: %s/%s 's pv %s label [%s] not equals %s",
				ns, tcName, pvName, label.NamespaceLabelKey, ns)
		}
		if pv.Labels[label.ComponentLabelKey] != pod.Labels[label.ComponentLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s=%s] not equals pod label[%s=%s]",
				ns, tcName, pvName,
				label.ComponentLabelKey, pv.Labels[label.ComponentLabelKey],
				label.ComponentLabelKey, pod.Labels[label.ComponentLabelKey])
		}
		if pv.Labels[label.NameLabelKey] != pod.Labels[label.NameLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s=%s] not equals pod label [%s=%s]",
				ns, tcName, pvName,
				label.NameLabelKey, pv.Labels[label.NameLabelKey],
				label.NameLabelKey, pod.Labels[label.NameLabelKey])
		}
		if pv.Labels[label.ManagedByLabelKey] != pod.Labels[label.ManagedByLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s=%s] not equals pod label [%s=%s]",
				ns, tcName, pvName,
				label.ManagedByLabelKey, pv.Labels[label.ManagedByLabelKey],
				label.ManagedByLabelKey, pod.Labels[label.ManagedByLabelKey])
		}
		if pv.Labels[label.InstanceLabelKey] != pod.Labels[label.InstanceLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s=%s] not equals pod label [%s=%s]",
				ns, tcName, pvName,
				label.InstanceLabelKey, pv.Labels[label.InstanceLabelKey],
				label.InstanceLabelKey, pod.Labels[label.InstanceLabelKey])
		}
		if pv.Labels[label.ClusterIDLabelKey] != clusterID {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s] not equals %s",
				ns, tcName, pvName, label.ClusterIDLabelKey, clusterID)
		}
		if pv.Labels[label.MemberIDLabelKey] != pod.Labels[label.MemberIDLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s=%s] not equals pod label [%s=%s]",
				ns, tcName, pvName,
				label.MemberIDLabelKey, pv.Labels[label.MemberIDLabelKey],
				label.MemberIDLabelKey, pod.Labels[label.MemberIDLabelKey])
		}
		if pv.Labels[label.StoreIDLabelKey] != pod.Labels[label.StoreIDLabelKey] {
			return false, fmt.Errorf("tidbcluster: %s/%s's pv %s label [%s=%s] not equals pod label [%s=%s]",
				ns, tcName, pvName,
				label.StoreIDLabelKey, pv.Labels[label.StoreIDLabelKey],
				label.StoreIDLabelKey, pod.Labels[label.StoreIDLabelKey])
		}
		if pv.Annotations[label.AnnPodNameKey] != podName {
			return false, fmt.Errorf("tidbcluster:[%s/%s's pv %s annotations [%s] not equals %s",
				ns, tcName, pvName, label.AnnPodNameKey, podName)
		}
	}

	return true, nil
}

func (ctu *CrdTestUtil) schedulerHAFn(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	fn := func(component string) (bool, error) {
		nodeMap := make(map[string][]string)
		listOptions := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(
				label.New().Instance(tcName).Component(component).Labels()).String(),
		}
		var podList *corev1.PodList
		var err error
		if podList, err = ctu.kubeCli.CoreV1().Pods(ns).List(listOptions); err != nil {
			log.Logf("failed to list pods for tidbcluster %s/%s, %v", ns, tcName, err)
			return false, nil
		}

		totalCount := len(podList.Items)
		for _, pod := range podList.Items {
			nodeName := pod.Spec.NodeName
			if len(nodeMap[nodeName]) == 0 {
				nodeMap[nodeName] = make([]string, 0)
			}
			nodeMap[nodeName] = append(nodeMap[nodeName], pod.GetName())
			if len(nodeMap[nodeName]) > totalCount/2 {
				return false, fmt.Errorf("node %s have %d pods, greater than %d/2",
					nodeName, len(nodeMap[nodeName]), totalCount)
			}
		}
		return true, nil
	}

	components := []string{label.PDLabelVal, label.TiKVLabelVal}
	for _, com := range components {
		if b, err := fn(com); err != nil {
			return false, err
		} else if !b && err == nil {
			return false, nil
		}
	}

	return true, nil
}

func (ctu *CrdTestUtil) podsScheduleAnnHaveDeleted(tc *v1alpha1.TidbCluster) (bool, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Labels()).String(),
	}

	pvcList, err := ctu.kubeCli.CoreV1().PersistentVolumeClaims(ns).List(listOptions)
	if err != nil {
		log.Logf("failed to list pvcs for tidb cluster %s/%s, err: %v", ns, tcName, err)
		return false, nil
	}

	for _, pvc := range pvcList.Items {
		pvcName := pvc.GetName()
		l := label.Label(pvc.Labels)
		if !(l.IsPD() || l.IsTiKV()) {
			continue
		}

		if _, exist := pvc.Annotations[label.AnnPVCPodScheduling]; exist {
			log.Logf("tidb cluster %s/%s pvc %s has pod scheduling annotation", ns, tcName, pvcName)
			return false, nil
		}
	}

	return true, nil
}

func (ctu *CrdTestUtil) storeLabelsIsSet(tc *v1alpha1.TidbCluster, topologyKey string) (bool, error) {
	pdClient, cancel, err := ctu.getPDClient(tc)
	if err != nil {
		log.Logf("failed to create external PD client for tidb cluster %q: %v", tc.GetName(), err)
		return false, nil
	}
	defer cancel()
	for _, store := range tc.Status.TiKV.Stores {
		storeID, err := strconv.ParseUint(store.ID, 10, 64)
		if err != nil {
			return false, err
		}
		storeInfo, err := pdClient.GetStore(storeID)
		if err != nil {
			return false, nil
		}
		if len(storeInfo.Store.Labels) == 0 {
			return false, nil
		}
		for _, label := range storeInfo.Store.Labels {
			if label.Key != topologyKey {
				return false, nil
			}
		}
	}
	return true, nil
}

func (ctu *CrdTestUtil) passwordIsSet(clusterInfo *TidbClusterConfig) (bool, error) {
	ns := clusterInfo.Namespace
	tcName := clusterInfo.ClusterName
	jobName := tcName + "-tidb-initializer"

	var job *batchv1.Job
	var err error
	if job, err = ctu.kubeCli.BatchV1().Jobs(ns).Get(jobName, metav1.GetOptions{}); err != nil {
		log.Logf("failed to get job %s/%s, %v", ns, jobName, err)
		return false, nil
	}
	if job.Status.Succeeded < 1 {
		log.Logf("tidbcluster: %s/%s password setter job not finished", ns, tcName)
		return false, nil
	}

	var db *sql.DB
	dsn, cancel, err := ctu.getTiDBDSN(ns, tcName, "test", clusterInfo.Password)
	if err != nil {
		log.Logf("failed to get TiDB DSN: %v", err)
		return false, nil
	}
	defer cancel()
	if db, err = sql.Open("mysql", dsn); err != nil {
		log.Logf("can't open connection to mysql: %s, %v", dsn, err)
		return false, nil
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Logf("can't connect to mysql: %s with password %s, %v", dsn, clusterInfo.Password, err)
		return false, nil
	}

	return true, nil
}

func (ctu *CrdTestUtil) monitorNormal(clusterInfo *TidbClusterConfig) (bool, error) {
	ns := clusterInfo.Namespace
	tcName := clusterInfo.ClusterName
	monitorDeploymentName := fmt.Sprintf("%s-monitor", tcName)
	monitorDeployment, err := ctu.kubeCli.AppsV1().Deployments(ns).Get(monitorDeploymentName, metav1.GetOptions{})
	if err != nil {
		log.Logf("get monitor deployment: [%s/%s] failed", ns, monitorDeploymentName)
		return false, nil
	}
	if monitorDeployment.Status.ReadyReplicas < 1 {
		log.Logf("monitor ready replicas %d < 1", monitorDeployment.Status.ReadyReplicas)
		return false, nil
	}
	if err := ctu.checkPrometheus(clusterInfo); err != nil {
		log.Logf("check [%s/%s]'s prometheus data failed: %v", ns, monitorDeploymentName, err)
		return false, nil
	}

	if err := ctu.checkGrafanaData(clusterInfo); err != nil {
		log.Logf("check [%s/%s]'s grafana data failed: %v", ns, monitorDeploymentName, err)
		return false, nil
	}
	return true, nil
}

func (ctu *CrdTestUtil) checkTidbClusterConfigUpdated(tc *v1alpha1.TidbCluster, clusterInfo *TidbClusterConfig) (bool, error) {
	if ok := ctu.checkPdConfigUpdated(tc, clusterInfo); !ok {
		return false, nil
	}
	if ok := ctu.checkTiKVConfigUpdated(tc, clusterInfo); !ok {
		return false, nil
	}
	if ok := ctu.checkTiDBConfigUpdated(tc, clusterInfo); !ok {
		return false, nil
	}
	return true, nil
}

func (ctu *CrdTestUtil) checkPdConfigUpdated(tc *v1alpha1.TidbCluster, clusterInfo *TidbClusterConfig) bool {
	pdClient, cancel, err := ctu.getPDClient(tc)
	if err != nil {
		log.Logf("failed to create external PD client for tidb cluster %q: %v", tc.GetName(), err)
		return false
	}
	defer cancel()
	config, err := pdClient.GetConfig()
	if err != nil {
		log.Logf("failed to get PD configuraion from tidb cluster [%s/%s]", tc.Namespace, tc.Name)
		return false
	}
	if len(clusterInfo.PDLogLevel) > 0 && clusterInfo.PDLogLevel != config.Log.Level {
		log.Logf("check [%s/%s] PD logLevel configuration updated failed: desired [%s], actual [%s] not equal",
			tc.Namespace,
			tc.Name,
			clusterInfo.PDLogLevel,
			config.Log.Level)
		return false
	}
	// TODO: fix #487 PD configuration update for persisted configurations
	//if clusterInfo.PDMaxReplicas > 0 && config.Replication.MaxReplicas != uint64(clusterInfo.PDMaxReplicas) {
	//	log.Logf("check [%s/%s] PD maxReplicas configuration updated failed: desired [%d], actual [%d] not equal",
	//		tc.Namespace,
	//		tc.Name,
	//		clusterInfo.PDMaxReplicas,
	//		config.Replication.MaxReplicas)
	//	return false
	//}
	return true
}

func (ctu *CrdTestUtil) checkTiDBConfigUpdated(tc *v1alpha1.TidbCluster, clusterInfo *TidbClusterConfig) bool {
	ordinals, err := util.GetPodOrdinals(tc, v1alpha1.TiDBMemberType)
	if err != nil {
		log.Logf("failed to get pod ordinals for tidb cluster %s/%s (member: %v)", tc.Namespace, tc.Name, v1alpha1.TiDBMemberType)
		return false
	}
	for i := range ordinals {
		config, err := ctu.tidbControl.GetSettings(tc, int32(i))
		if err != nil {
			log.Logf("failed to get TiDB configuration from cluster [%s/%s], ordinal: %d, error: %v", tc.Namespace, tc.Name, i, err)
			return false
		}
		if clusterInfo.TiDBTokenLimit > 0 && uint(clusterInfo.TiDBTokenLimit) != config.TokenLimit {
			log.Logf("check [%s/%s] TiDB instance [%d] configuration updated failed: desired [%d], actual [%d] not equal",
				tc.Namespace, tc.Name, i, clusterInfo.TiDBTokenLimit, config.TokenLimit)
			return false
		}
	}
	return true
}

func (ctu *CrdTestUtil) checkTiKVConfigUpdated(tc *v1alpha1.TidbCluster, clusterInfo *TidbClusterConfig) bool {
	// TODO: check if TiKV configuration updated
	return true
}

func (ctu *CrdTestUtil) checkPrometheus(clusterInfo *TidbClusterConfig) error {
	ns := clusterInfo.Namespace
	tcName := clusterInfo.ClusterName
	return checkPrometheusCommon(tcName, ns, ctu.fw)
}

func (ctu *CrdTestUtil) checkGrafanaData(clusterInfo *TidbClusterConfig) error {
	ns := clusterInfo.Namespace
	tcName := clusterInfo.ClusterName
	grafanaClient, err := checkGrafanaDataCommon(tcName, ns, clusterInfo.GrafanaClient, ctu.fw)
	if err != nil {
		return err
	}
	if clusterInfo.GrafanaClient == nil && grafanaClient != nil {
		clusterInfo.GrafanaClient = grafanaClient
	}
	return nil
}

func (ctu *CrdTestUtil) checkReclaimPVSuccess(tc *v1alpha1.TidbCluster) (bool, error) {
	// check pv reclaim	for pd
	if err := ctu.checkComponentReclaimPVSuccess(tc, label.PDLabelVal); err != nil {
		log.Logf("%v", err)
		return false, nil
	}

	// check pv reclaim for tikv
	if err := ctu.checkComponentReclaimPVSuccess(tc, label.TiKVLabelVal); err != nil {
		log.Logf("%v", err)
		return false, nil
	}
	return true, nil
}

func (ctu *CrdTestUtil) checkComponentReclaimPVSuccess(tc *v1alpha1.TidbCluster, component string) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	var replica int
	switch component {
	case label.PDLabelVal:
		replica = int(tc.Spec.PD.Replicas)
	case label.TiKVLabelVal:
		replica = int(tc.Spec.TiKV.Replicas)
	default:
		return fmt.Errorf("check tidb cluster %s/%s component %s is not supported", ns, tcName, component)
	}

	pvcList, err := ctu.getComponentPVCList(tc, component)
	if err != nil {
		return err
	}

	pvList, err := ctu.getComponentPVList(tc, component)
	if err != nil {
		return err
	}

	if len(pvcList) != replica {
		return fmt.Errorf("tidb cluster %s/%s component %s pvc has not been reclaimed completely, expected: %d, got %d", ns, tcName, component, replica, len(pvcList))
	}

	if len(pvList) != replica {
		return fmt.Errorf("tidb cluster %s/%s component %s pv has not been reclaimed completely, expected: %d, got %d", ns, tcName, component, replica, len(pvList))
	}

	return nil
}

func (ctu *CrdTestUtil) getComponentPVCList(tc *v1alpha1.TidbCluster, component string) ([]corev1.PersistentVolumeClaim, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Component(component).Labels()).String(),
	}

	pvcList, err := ctu.kubeCli.CoreV1().PersistentVolumeClaims(ns).List(listOptions)
	if err != nil {
		err := fmt.Errorf("failed to list pvcs for tidb cluster %s/%s, component: %s, err: %v", ns, tcName, component, err)
		return nil, err
	}
	return pvcList.Items, nil
}

func (ctu *CrdTestUtil) getComponentPVList(tc *v1alpha1.TidbCluster, component string) ([]corev1.PersistentVolume, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			label.New().Instance(tcName).Component(component).Namespace(ns).Labels()).String(),
	}

	pvList, err := ctu.kubeCli.CoreV1().PersistentVolumes().List(listOptions)
	if err != nil {
		err := fmt.Errorf("failed to list pvs for tidb cluster %s/%s, component: %s, err: %v", ns, tcName, component, err)
		return nil, err
	}
	return pvList.Items, nil
}
