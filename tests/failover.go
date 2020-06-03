// Copyright 2019 PingCAP, Inc.
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
	"database/sql"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	// To register MySQL driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	"github.com/pingcap/tidb-operator/tests/pkg/ops"
	"github.com/pingcap/tidb-operator/tests/slack"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

func (oa *operatorActions) DeletePDDataThenCheckFailover(info *TidbClusterConfig, pdFailoverPeriod time.Duration) error {
	const failoverTimeout = 5 * time.Minute
	ns := info.Namespace
	tcName := info.ClusterName
	podName := fmt.Sprintf("%s-pd-0", tcName)

	var err error
	var result []byte
	err = wait.Poll(10*time.Second, time.Minute, func() (bool, error) {
		deletePDDataCmd := fmt.Sprintf("kubectl exec -n %s %s -- rm -rf /var/lib/pd/member", ns, podName)
		result, err = exec.Command("/bin/sh", "-c", deletePDDataCmd).CombinedOutput()
		if err != nil {
			klog.Error(err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete pod %s/%s data, %s", ns, podName, string(result))
	}
	klog.Infof("delete pod %s/%s data successfully", ns, podName)

	oldPD, err := oa.kubeCli.CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
	if err != nil {
		klog.Error(err.Error())
		return err
	}
	// first we ensured that pd failover new pod, and failure member/pod should be deleted.
	err = wait.Poll(10*time.Second, failoverTimeout+pdFailoverPeriod, func() (bool, error) {
		tc, err := oa.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
		if err != nil {
			klog.Error(err)
			return false, nil
		}

		// ensure oldPD is deleted
		newPd, err := oa.kubeCli.CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
		if err != nil {
			klog.Error(err)
			return false, nil
		}
		if string(oldPD.UID) == string(newPd.UID) {
			klog.Infof("oldPD should be deleted and newPD should be created")
			return false, nil
		}

		// ensure failure member has deleted state
		if len(tc.Status.PD.FailureMembers) == 1 {
			klog.Infof("%#v", tc.Status.PD.FailureMembers)
			for _, failureMember := range tc.Status.PD.FailureMembers {
				if failureMember.MemberDeleted {
					return true, nil
				}
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to check pd %s/%s failover", ns, podName)
	}
	klog.Infof("check pd pod %s/%s failover marked successfully, new pod verified", ns, podName)

	// Then we ensure pd failover recovery
	err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		tc, err := oa.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
		if err != nil {
			klog.Error(err)
			return false, nil
		}

		if tc.Status.PD.FailureMembers != nil && len(tc.Status.PD.FailureMembers) > 0 {
			klog.Error("pd failover should empty failure members in recovery")
			return false, nil
		}
		pdSpecReplicas := tc.Spec.PD.Replicas
		pdsts, err := oa.kubeCli.AppsV1().StatefulSets(ns).Get(fmt.Sprintf("%s-pd", tc.Name), metav1.GetOptions{})
		if err != nil {
			klog.Error(err.Error())
			return false, nil
		}
		if *pdsts.Spec.Replicas != pdSpecReplicas {
			klog.Errorf("pdsts replicas[%d] should equal pdspec replicas[%d]", pdSpecReplicas, *pdsts.Spec.Replicas)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("check pd cluster %s/%s recovery failed after failover", ns, tcName)
	}
	klog.Infof("pd cluster have been recovered")

	err = oa.CheckTidbClusterStatus(info)
	if err != nil {
		return err
	}

	klog.Infof("recover %s/%s successfully", ns, podName)
	return nil
}

func (oa *operatorActions) DeletePDDataThenCheckFailoverOrDie(info *TidbClusterConfig, pdFailoverPeriod time.Duration) {
	if err := oa.DeletePDDataThenCheckFailover(info, pdFailoverPeriod); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) TruncateSSTFileThenCheckFailover(info *TidbClusterConfig, tikvFailoverPeriod time.Duration) error {
	const failoverTimeout = 5 * time.Minute

	cli := client.Union(oa.kubeCli, oa.cli)
	tikvOps := ops.TiKVOps{ClientOps: ops.ClientOps{Client: cli}}

	// checkout latest tidb cluster
	tc, err := cli.PingcapV1alpha1().TidbClusters(info.Namespace).Get(info.ClusterName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get the cluster: ns=%s tc=%s err=%s", info.Namespace, info.ClusterName, err.Error())
		return err
	}

	// checkout pd config
	pdCfg, err := oa.pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled()).GetConfig()
	if err != nil {
		klog.Errorf("failed to get the pd config: tc=%s err=%s", info.ClusterName, err.Error())
		return err
	}
	maxStoreDownTime, err := time.ParseDuration(pdCfg.Schedule.MaxStoreDownTime)
	if err != nil {
		return err
	}
	klog.Infof("truncate sst file failover config: maxStoreDownTime=%v tikvFailoverPeriod=%v", maxStoreDownTime, tikvFailoverPeriod)

	// find an up store
	var store v1alpha1.TiKVStore
	var podName string
	for _, v := range tc.Status.TiKV.Stores {
		if v.State != v1alpha1.TiKVStateUp {
			continue
		}
		store = v
		podName = v.PodName
		break
	}
	if len(store.ID) == 0 {
		klog.Errorf("failed to find an up store")
		return errors.New("no up store for truncating sst file")
	}
	klog.Infof("truncate sst file target store: id=%s pod=%s", store.ID, store.PodName)

	oa.EmitEvent(info, fmt.Sprintf("TruncateSSTFile: tikv: %s", store.PodName))
	klog.Infof("deleting pod: [%s/%s] and wait 1 minute for the pod to terminate", info.Namespace, store.PodName)
	err = cli.CoreV1().Pods(info.Namespace).Delete(store.PodName, nil)
	if err != nil {
		klog.Errorf("failed to get delete the pod: ns=%s tc=%s pod=%s err=%s",
			info.Namespace, info.ClusterName, store.PodName, err.Error())
		return err
	}

	time.Sleep(1 * time.Minute)

	// truncate the sst file and wait for failover
	err = tikvOps.TruncateSSTFile(ops.TruncateOptions{
		Namespace: info.Namespace,
		Cluster:   info.ClusterName,
		Store:     store.ID,
	})
	if err != nil {
		klog.Errorf("failed to truncate the sst file: ns=%s tc=%s store=%s err=%s",
			info.Namespace, info.ClusterName, store.ID, err.Error())
		return err
	}
	oa.EmitEvent(info, fmt.Sprintf("TruncateSSTFile: tikv: %s/%s", info.Namespace, store.PodName))

	// delete tikv pod
	klog.Infof("deleting pod: [%s/%s] again", info.Namespace, store.PodName)
	err = wait.Poll(10*time.Second, time.Minute, func() (bool, error) {
		err = oa.kubeCli.CoreV1().Pods(info.Namespace).Delete(store.PodName, &metav1.DeleteOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		klog.Error(err.Error())
		return err
	}

	tikvOps.SetPoll(DefaultPollInterval, maxStoreDownTime+tikvFailoverPeriod+failoverTimeout)

	err = tikvOps.PollTiDBCluster(info.Namespace, info.ClusterName,
		func(tc *v1alpha1.TidbCluster, err error) (bool, error) {
			_, ok := tc.Status.TiKV.FailureStores[store.ID]
			klog.Infof("cluster: [%s/%s] check if target store failed: %t",
				info.Namespace, info.ClusterName, ok)
			if !ok {
				return false, nil
			}
			return true, nil
		})
	if err != nil {
		klog.Errorf("failed to check truncate sst file: %v", err)
		return err
	}

	if err := wait.Poll(1*time.Minute, 30*time.Minute, func() (bool, error) {
		if err := tikvOps.RecoverSSTFile(info.Namespace, podName); err != nil {
			klog.Errorf("failed to recovery sst file %s/%s, %v", info.Namespace, podName, err)
			return false, nil
		}

		return true, nil
	}); err != nil {
		return err
	}

	klog.Infof("deleting pod: [%s/%s] again", info.Namespace, store.PodName)
	err = wait.Poll(10*time.Second, time.Minute, func() (bool, error) {
		err = oa.kubeCli.CoreV1().Pods(info.Namespace).Delete(store.PodName, &metav1.DeleteOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	ns := info.Namespace
	tcName := info.ClusterName
	return wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		tc, err := oa.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
		if err != nil {
			klog.Error(err.Error())
			return false, nil
		}
		if tc.Status.TiKV.FailureStores == nil || len(tc.Status.TiKV.FailureStores) < 1 {
			return true, nil
		}
		tc.Status.TiKV.FailureStores = nil
		tc, err = oa.cli.PingcapV1alpha1().TidbClusters(ns).Update(tc)
		if err != nil {
			klog.Error(err.Error())
		}
		return false, nil
	})
}

func (oa *operatorActions) TruncateSSTFileThenCheckFailoverOrDie(info *TidbClusterConfig, tikvFailoverPeriod time.Duration) {
	if err := oa.TruncateSSTFileThenCheckFailover(info, tikvFailoverPeriod); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CheckFailoverPending(info *TidbClusterConfig, node string, faultPoint *time.Time) (bool, error) {
	affectedPods, err := oa.getPodsByNode(info, node)
	if err != nil {
		klog.Infof("cluster:[%s] query pods failed,error:%v", info.FullName(), err)
		return false, nil
	}
	tc, err := oa.cli.PingcapV1alpha1().TidbClusters(info.Namespace).Get(info.ClusterName, metav1.GetOptions{})
	if err != nil {
		klog.Infof("pending failover,failed to get tidbcluster:[%s], error: %v", info.FullName(), err)
		if strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers") {
			klog.Info("create new client")
			newCli, _, _, _, _ := client.NewCliOrDie()
			oa.cli = newCli
		}
		return false, nil
	}
	deadline := faultPoint.Add(period)
	if time.Now().Before(deadline) {
		if tc.Status.PD.FailureMembers != nil && len(tc.Status.PD.FailureMembers) > 0 {
			for _, failureMember := range tc.Status.PD.FailureMembers {
				if _, exist := affectedPods[failureMember.PodName]; exist {
					err := fmt.Errorf("cluster: [%s] the pd member[%s] should be mark failure after %s", info.FullName(), failureMember.PodName, deadline.Format(time.RFC3339))
					klog.Errorf(err.Error())
					return false, err
				}
			}
		}
		if tc.Status.TiKV.FailureStores != nil && len(tc.Status.TiKV.FailureStores) > 0 {
			for _, failureStore := range tc.Status.TiKV.FailureStores {
				if _, exist := affectedPods[failureStore.PodName]; exist {
					err := fmt.Errorf("cluster: [%s] the tikv store[%s] should be mark failure after %s", info.FullName(), failureStore.PodName, deadline.Format(time.RFC3339))
					klog.Errorf(err.Error())
					// There may have been a failover before
					return false, nil
				}
			}

		}
		if tc.Status.TiDB.FailureMembers != nil && len(tc.Status.TiDB.FailureMembers) > 0 {
			for _, failureMember := range tc.Status.TiDB.FailureMembers {
				if _, exist := affectedPods[failureMember.PodName]; exist {
					err := fmt.Errorf("cluster: [%s] the tidb member[%s] should be mark failure after %s", info.FullName(), failureMember.PodName, deadline.Format(time.RFC3339))
					klog.Errorf(err.Error())
					return false, err
				}
			}
		}

		klog.Infof("cluster: [%s] operator's failover feature is pending", info.FullName())
		return false, nil
	}
	return true, nil
}

func (oa *operatorActions) CheckFailoverPendingOrDie(clusters []*TidbClusterConfig, node string, faultPoint *time.Time) {
	if err := wait.Poll(1*time.Minute, 30*time.Minute, func() (bool, error) {
		var passes []bool
		for i := range clusters {
			pass, err := oa.CheckFailoverPending(clusters[i], node, faultPoint)
			if err != nil {
				return pass, err
			}
			passes = append(passes, pass)
		}
		for _, pass := range passes {
			if !pass {
				return false, nil
			}
		}
		return true, nil
	}); err != nil {
		slack.NotifyAndPanic(fmt.Errorf("failed to check failover pending"))
	}
}

func (oa *operatorActions) CheckFailover(info *TidbClusterConfig, node string) (bool, error) {
	affectedPods, err := oa.getPodsByNode(info, node)
	if err != nil {
		klog.Infof("cluster:[%s] query pods failed,error:%v", info.FullName(), err)
		return false, nil
	}

	if len(affectedPods) == 0 {
		klog.Infof("the cluster:[%s] can not be affected by node:[%s]", info.FullName(), node)
		return true, nil
	}

	tc, err := oa.cli.PingcapV1alpha1().TidbClusters(info.Namespace).Get(info.ClusterName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("query tidbcluster: [%s] failed, error: %v", info.FullName(), err)
		return false, nil
	}

	for _, affectedPod := range affectedPods {
		switch affectedPod.Labels[label.ComponentLabelKey] {
		case label.PDLabelVal:
			if !oa.pdFailover(affectedPod, tc) {
				return false, nil
			}
		case label.TiKVLabelVal:
			if !oa.tikvFailover(affectedPod, tc) {
				return false, nil
			}
		case label.TiDBLabelVal:
			if !oa.tidbFailover(affectedPod, tc) {
				return false, nil
			}
		}
	}

	klog.Infof("cluster: [%s]'s failover feature has complete", info.FullName())
	return true, nil
}

func (oa *operatorActions) getPodsByNode(info *TidbClusterConfig, node string) (map[string]*corev1.Pod, error) {
	selector, err := label.New().Instance(info.ClusterName).Selector()
	if err != nil {
		klog.Errorf("cluster:[%s] create selector failed, error:%v", info.FullName(), err)
		return nil, err
	}
	pods, err := oa.kubeCli.CoreV1().Pods(info.Namespace).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		klog.Errorf("cluster:[%s] query pods failed, error:%v", info.FullName(), err)
		return nil, err
	}
	podsOfNode := map[string]*corev1.Pod{}
	for i, pod := range pods.Items {
		if pod.Spec.NodeName == node {
			podsOfNode[pod.Name] = &pods.Items[i]
		}
	}

	return podsOfNode, nil
}

func (oa *operatorActions) CheckFailoverOrDie(clusters []*TidbClusterConfig, faultNode string) {
	if err := wait.Poll(1*time.Minute, 60*time.Minute, func() (bool, error) {
		var passes []bool
		for i := range clusters {
			pass, err := oa.CheckFailover(clusters[i], faultNode)
			if err != nil {
				return pass, err
			}
			passes = append(passes, pass)
		}
		for _, pass := range passes {
			if !pass {
				return false, nil
			}
		}
		return true, nil
	}); err != nil {
		slack.NotifyAndPanic(fmt.Errorf("failed to check failover"))
	}
}

func (oa *operatorActions) CheckRecover(cluster *TidbClusterConfig) (bool, error) {
	tc, err := oa.cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
	if err != nil {
		return false, nil
	}

	if tc.Status.PD.FailureMembers != nil && len(tc.Status.PD.FailureMembers) > 0 {
		klog.Infof("cluster: [%s]'s pd FailureMembers is not nil, continue to wait", cluster.FullName())
		return false, nil
	}

	if tc.Status.TiDB.FailureMembers != nil && len(tc.Status.TiDB.FailureMembers) > 0 {
		klog.Infof("cluster: [%s]'s tidb FailureMembers is not nil, continue to wait", cluster.FullName())
		return false, nil
	}

	// recover tikv manually
	if tc.Status.TiKV.FailureStores != nil {
		tc.Status.TiKV.FailureStores = nil
		tc, err = oa.cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Update(tc)
		if err != nil {
			klog.Errorf("failed to set status.tikv.failureStore to nil, %v", err)
			return false, nil
		}
	}

	return true, nil
}

func (oa *operatorActions) CheckRecoverOrDie(clusters []*TidbClusterConfig) {
	if err := wait.Poll(DefaultPollInterval, DefaultPollTimeout, func() (bool, error) {
		var passes []bool
		for i := range clusters {
			pass, err := oa.CheckRecover(clusters[i])
			if err != nil {
				return pass, err
			}
			passes = append(passes, pass)
		}
		for _, pass := range passes {
			if !pass {
				return false, nil
			}
		}
		return true, nil
	}); err != nil {
		slack.NotifyAndPanic(fmt.Errorf("failed to check recover"))
	}
}

func (oa *operatorActions) pdFailover(pod *corev1.Pod, tc *v1alpha1.TidbCluster) bool {
	failure := false
	for _, failureMember := range tc.Status.PD.FailureMembers {
		if failureMember.PodName == pod.GetName() {
			failure = true
			break
		}
	}
	if !failure {
		klog.Infof("tidbCluster:[%s/%s]'s member:[%s] have not become failuremember", tc.Namespace, tc.Name, pod.Name)
		return false
	}

	for _, member := range tc.Status.PD.Members {
		if member.Name == pod.GetName() {
			klog.Infof("tidbCluster:[%s/%s]'s status.members still have pd member:[%s]", tc.Namespace, tc.Name, pod.Name)
			return false
		}
	}

	if tc.Status.PD.Synced && len(tc.Status.PD.Members) == int(tc.Spec.PD.Replicas) {
		return true
	}

	klog.Infof("cluster: [%s/%s] pd:[%s] failover still not complete", tc.Namespace, tc.Name, pod.GetName())

	return false
}

// TODO we should confirm the tombstone exists, important!!!!!!
// 		for example: offline the same pod again and again, and see it in the tombstone stores
// 					 offline two pods, and see them in the tombstone stores
func (oa *operatorActions) tikvFailover(pod *corev1.Pod, tc *v1alpha1.TidbCluster) bool {
	failure := false
	for _, failureStore := range tc.Status.TiKV.FailureStores {
		if failureStore.PodName == pod.GetName() {
			failure = true
			break
		}
	}
	if !failure {
		klog.Infof("tidbCluster:[%s/%s]'s store pod:[%s] have not become failuremember", tc.Namespace, tc.Name, pod.Name)
		return false
	}

	healthCount := 0
	for _, store := range tc.Status.TiKV.Stores {
		if store.State == v1alpha1.TiKVStateUp {
			healthCount++
		}
	}
	if tc.Status.TiKV.Synced && healthCount >= int(tc.Spec.TiKV.Replicas) {
		return true
	}

	klog.Infof("cluster: [%s/%s] tikv:[%s] failover still not complete", tc.Namespace, tc.Name, pod.GetName())
	return false
}

func (oa *operatorActions) tidbFailover(pod *corev1.Pod, tc *v1alpha1.TidbCluster) bool {
	failure := false
	for _, failureMember := range tc.Status.TiDB.FailureMembers {
		if failureMember.PodName == pod.GetName() {
			klog.Infof("tidbCluster:[%s/%s]'s store pod:[%s] have become failuremember", tc.Namespace, tc.Name, pod.Name)
			failure = true
			break
		}
	}
	if !failure {
		return false
	}

	healthCount := 0
	for _, member := range tc.Status.TiDB.Members {
		if member.Health {
			healthCount++
		}
	}

	if healthCount == int(tc.Spec.TiDB.Replicas) {
		return true
	}
	klog.Infof("cluster: [%s/%s] tidb:[%s] failover still not complete", tc.Namespace, tc.Name, pod.GetName())
	return false
}

func (oa *operatorActions) GetPodUIDMap(info *TidbClusterConfig) (map[string]types.UID, error) {
	result := map[string]types.UID{}

	selector, err := label.New().Instance(info.ClusterName).Selector()
	if err != nil {
		return nil, err
	}
	pods, err := oa.kubeCli.CoreV1().Pods(info.Namespace).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	for _, pod := range pods.Items {
		result[pod.GetName()] = pod.GetUID()
	}

	return result, nil
}

func (oa *operatorActions) GetNodeMap(info *TidbClusterConfig, component string) (map[string][]string, error) {
	nodeMap := make(map[string][]string)
	selector := label.New().Instance(info.ClusterName).Component(component).Labels()
	podList, err := oa.kubeCli.CoreV1().Pods(info.Namespace).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selector).String(),
	})
	if err != nil {
		return nil, err
	}

	for _, pod := range podList.Items {
		nodeName := pod.Spec.NodeName
		if len(nodeMap[nodeName]) == 0 {
			nodeMap[nodeName] = make([]string, 0)
		}
		nodeMap[nodeName] = append(nodeMap[nodeName], pod.GetName())
		sort.Strings(nodeMap[nodeName])
	}

	return nodeMap, nil
}

func (oa *operatorActions) CheckKubeletDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig, faultNode string) {
	klog.Infof("check k8s/operator/tidbCluster status when kubelet down")
	time.Sleep(10 * time.Minute)
	KeepOrDie(3*time.Second, 10*time.Minute, func() error {
		err := oa.CheckK8sAvailable(nil, nil)
		if err != nil {
			return err
		}
		klog.V(4).Infof("k8s cluster is available.")
		err = oa.CheckOperatorAvailable(operatorConfig)
		if err != nil {
			return err
		}
		klog.V(4).Infof("tidb operator is available.")
		err = oa.CheckTidbClustersAvailable(clusters)
		if err != nil {
			return err
		}
		klog.V(4).Infof("all clusters are available")
		return nil
	})
}

func (oa *operatorActions) CheckEtcdDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig, faultNode string) {
	klog.Infof("check k8s/operator/tidbCluster status when etcd down")
	// kube-apiserver may block 15 min
	time.Sleep(20 * time.Minute)
	KeepOrDie(3*time.Second, 10*time.Minute, func() error {
		err := oa.CheckK8sAvailable(nil, nil)
		if err != nil {
			return err
		}
		klog.V(4).Infof("k8s cluster is available.")
		err = oa.CheckOperatorAvailable(operatorConfig)
		if err != nil {
			return err
		}
		klog.V(4).Infof("tidb operator is available.")
		err = oa.CheckTidbClustersAvailable(clusters)
		if err != nil {
			return err
		}
		klog.V(4).Infof("all clusters are available")
		return nil
	})
}

func (oa *operatorActions) CheckKubeProxyDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig) {
	klog.Infof("checking k8s/tidbCluster status when kube-proxy down")

	KeepOrDie(3*time.Second, 10*time.Minute, func() error {
		err := oa.CheckK8sAvailable(nil, nil)
		if err != nil {
			return err

		}
		klog.V(4).Infof("k8s cluster is available.")

		err = oa.CheckOperatorAvailable(operatorConfig)
		if err != nil {
			return err
		}
		klog.V(4).Infof("tidb operator is available.")

		err = oa.CheckTidbClustersAvailable(clusters)
		if err != nil {
			return err
		}
		klog.V(4).Infof("all clusters are available.")
		return nil
	})
}

func (oa *operatorActions) CheckKubeSchedulerDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig) {
	klog.Infof("verify kube-scheduler is not avaiavble")

	if err := waitForComponentStatus(oa.kubeCli, "scheduler", corev1.ComponentHealthy, corev1.ConditionFalse); err != nil {
		slack.NotifyAndPanic(fmt.Errorf("failed to stop kube-scheduler: %v", err))
	}

	klog.Infof("checking operator/tidbCluster status when kube-scheduler is not available")

	KeepOrDie(3*time.Second, 10*time.Minute, func() error {
		err := oa.CheckOperatorAvailable(operatorConfig)
		if err != nil {
			return err
		}
		klog.V(4).Infof("tidb operator is available.")

		err = oa.CheckTidbClustersAvailable(clusters)
		if err != nil {
			return err
		}
		klog.V(4).Infof("all clusters are available.")
		return nil
	})
}

func (oa *operatorActions) CheckKubeControllerManagerDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig) {
	klog.Infof("verify kube-controller-manager is not avaiavble")

	if err := waitForComponentStatus(oa.kubeCli, "controller-manager", corev1.ComponentHealthy, corev1.ConditionFalse); err != nil {
		slack.NotifyAndPanic(fmt.Errorf("failed to stop kube-controller-manager: %v", err))
	}

	klog.Infof("checking operator/tidbCluster status when kube-controller-manager is not available")

	KeepOrDie(3*time.Second, 10*time.Minute, func() error {
		err := oa.CheckOperatorAvailable(operatorConfig)
		if err != nil {
			return err
		}
		klog.V(4).Infof("tidb operator is available.")

		err = oa.CheckTidbClustersAvailable(clusters)
		if err != nil {
			return err
		}
		klog.V(4).Infof("all clusters are available.")
		return nil
	})
}

func (oa *operatorActions) CheckOneApiserverDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig, faultNode string) {
	klog.Infof("check k8s/operator/tidbCluster status when one apiserver down")
	affectedPods := map[string]*corev1.Pod{}
	apiserverPod, err := GetKubeApiserverPod(oa.kubeCli, faultNode)
	if err != nil {
		slack.NotifyAndPanic(fmt.Errorf("can't find apiserver in k8s cluster"))
	}
	if apiserverPod != nil {
		affectedPods[apiserverPod.GetName()] = apiserverPod
	}

	controllerPod, err := GetKubeControllerManagerPod(oa.kubeCli, faultNode)
	if err != nil {
		slack.NotifyAndPanic(fmt.Errorf("can't find kube-controller-manager in k8s cluster"))
	}
	if controllerPod != nil {
		affectedPods[controllerPod.GetName()] = controllerPod
	}

	schedulerPod, err := GetKubeSchedulerPod(oa.kubeCli, faultNode)
	if err != nil {
		slack.NotifyAndPanic(fmt.Errorf("can't find kube-scheduler in k8s cluster"))
	}
	if schedulerPod != nil {
		affectedPods[schedulerPod.GetName()] = schedulerPod
	}

	dnsPod, err := GetKubeDNSPod(oa.kubeCli, faultNode)
	if err != nil {
		slack.NotifyAndPanic(fmt.Errorf("can't find kube-dns in k8s cluster"))
	}
	if dnsPod != nil {
		affectedPods[dnsPod.GetName()] = dnsPod
	}

	proxyPod, err := GetKubeProxyPod(oa.kubeCli, faultNode)
	if err != nil {
		slack.NotifyAndPanic(fmt.Errorf("can't find kube-proxy in k8s cluster"))
	}
	if proxyPod != nil {
		affectedPods[proxyPod.GetName()] = proxyPod
	}
	KeepOrDie(3*time.Second, 10*time.Minute, func() error {
		err := oa.CheckK8sAvailable(map[string]string{faultNode: faultNode}, affectedPods)
		if err != nil {
			klog.Errorf("CheckK8sAvailable Failed, err: %v", err)
			return err
		}
		klog.V(4).Infof("k8s cluster is available.")
		err = oa.CheckOperatorAvailable(operatorConfig)
		if err != nil {
			klog.Errorf("CheckOperatorAvailable Failed, err: %v", err)
			return err
		}
		klog.V(4).Infof("tidb operator is available.")
		err = oa.CheckTidbClustersAvailable(clusters)
		if err != nil {
			klog.Errorf("CheckTidbClustersAvailable Failed, err: %v", err)
			return err
		}
		klog.V(4).Infof("all clusters is available")
		return nil
	})
}

func (oa *operatorActions) CheckAllApiserverDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig) {
	KeepOrDie(3*time.Second, 10*time.Minute, func() error {
		err := oa.CheckTidbClustersAvailable(clusters)
		if err != nil {
			return err
		}
		klog.V(4).Infof("all clusters is available")
		return nil
	})
}

func (oa *operatorActions) CheckOperatorDownOrDie(clusters []*TidbClusterConfig) {
	klog.Infof("checking k8s/tidbCluster status when operator down")

	KeepOrDie(3*time.Second, 10*time.Minute, func() error {
		err := oa.CheckK8sAvailable(nil, nil)
		if err != nil {
			return err
		}

		return oa.CheckTidbClustersAvailable(clusters)
	})
}

func (oa *operatorActions) CheckK8sAvailableOrDie(excludeNodes map[string]string, excludePods map[string]*corev1.Pod) {
	if err := oa.CheckK8sAvailable(excludeNodes, excludePods); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CheckK8sAvailable(excludeNodes map[string]string, excludePods map[string]*corev1.Pod) error {
	return wait.Poll(3*time.Second, 10*time.Minute, func() (bool, error) {
		nodes, err := oa.kubeCli.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			klog.Errorf("failed to list nodes,error:%v", err)
			return false, nil
		}
		for _, node := range nodes.Items {
			if _, exist := excludeNodes[node.GetName()]; exist {
				continue
			}
			for _, condition := range node.Status.Conditions {
				if condition.Type == corev1.NodeReady && condition.Status != corev1.ConditionTrue {
					klog.Infof("node: [%s] is not in running", node.GetName())
					return false, nil
				}
			}
		}
		systemPods, err := oa.kubeCli.CoreV1().Pods("kube-system").List(metav1.ListOptions{})
		if err != nil {
			klog.Errorf("failed to list kube-system pods,error:%v", err)
			return false, nil
		}
		for _, pod := range systemPods.Items {
			if _, exist := excludePods[pod.GetName()]; exist {
				continue
			}
			podState := GetPodStatus(&pod)
			if podState != string(corev1.PodRunning) {
				klog.Infof("pod:[%s/%s] is unavailable,state is %s", pod.GetNamespace(), pod.GetName(), podState)
				return false, nil
			}
		}
		return true, nil
	})
}

func (oa *operatorActions) CheckOperatorAvailable(operatorConfig *OperatorConfig) error {
	var errCount int
	var e error
	return wait.Poll(10*time.Second, 3*time.Minute, func() (bool, error) {
		if errCount >= 10 {
			return true, e
		}
		controllerDeployment, err := oa.kubeCli.AppsV1().Deployments(operatorConfig.Namespace).Get(tidbControllerName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get deployment：%s failed,error:%v", tidbControllerName, err)
			return false, nil
		}
		if controllerDeployment.Status.AvailableReplicas != *controllerDeployment.Spec.Replicas {
			e = fmt.Errorf("the %s is not available", tidbControllerName)
			klog.Error(e)
			errCount++
			return false, nil
		}
		schedulerDeployment, err := oa.kubeCli.AppsV1().Deployments(operatorConfig.Namespace).Get(tidbSchedulerName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get deployment：%s failed,error:%v", tidbSchedulerName, err)
			return false, nil
		}
		if schedulerDeployment.Status.AvailableReplicas != *schedulerDeployment.Spec.Replicas {
			e = fmt.Errorf("the %s is not available", tidbSchedulerName)
			klog.Error(e)
			errCount++
			return false, nil
		}
		return true, nil
	})
}

func (oa *operatorActions) CheckTidbClustersAvailable(infos []*TidbClusterConfig) error {
	return wait.Poll(3*time.Second, DefaultPollTimeout, func() (bool, error) {
		for _, info := range infos {
			succ, err := oa.addDataToCluster(info)
			if err != nil {
				return false, err
			}
			if !succ {
				return false, nil
			}
		}
		return true, nil
	})

}

func (oa *operatorActions) CheckTidbClustersAvailableOrDie(infos []*TidbClusterConfig) {
	if err := oa.CheckTidbClustersAvailable(infos); err != nil {
		slack.NotifyAndPanic(err)
	}
}

var testTableName = "testTable"

func (oa *operatorActions) addDataToCluster(info *TidbClusterConfig) (bool, error) {
	dsn, cancel, err := oa.getTiDBDSN(info.Namespace, info.ClusterName, "test", info.Password)
	if err != nil {
		klog.Errorf("failed to get TiDB DSN: %v", err)
		return false, nil
	}
	defer cancel()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		klog.Errorf("cluster:[%s] can't open connection to mysql: %v", info.FullName(), err)
		return false, nil
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE TABLE %s (name VARCHAR(64))", testTableName))
	if err != nil && !tableAlreadyExist(err) {
		klog.Errorf("cluster:[%s] can't create table to mysql: %v", info.FullName(), err)
		return false, nil
	}

	_, err = db.Exec(fmt.Sprintf("INSERT INTO %s VALUES (?)", testTableName), "testValue")
	if err != nil {
		klog.Errorf("cluster:[%s] can't insert data to mysql: %v", info.FullName(), err)
		return false, nil
	}

	return true, nil
}

func (oa *operatorActions) WaitPodOnNodeReadyOrDie(clusters []*TidbClusterConfig, faultNode string) {

	err := wait.Poll(5*time.Second, 30*time.Minute, func() (done bool, err error) {
		for _, cluster := range clusters {
			listOptions := metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(
					label.New().Instance(cluster.ClusterName).Labels()).String(),
			}
			pods, err := oa.kubeCli.CoreV1().Pods(cluster.Namespace).List(listOptions)
			if err != nil {
				klog.Error(err.Error())
				return false, nil
			}
			for _, pod := range pods.Items {
				if pod.Spec.NodeName == faultNode {
					if !podutil.IsPodReady(&pod) {
						return false, nil
					}
				}
			}
		}
		return true, nil
	})
	if err != nil {
		slack.NotifyAndPanic(fmt.Errorf("pod on node[%s] not ready, err:%v", faultNode, err))
	}
}

func GetPodStatus(pod *corev1.Pod) string {
	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}
	if !initializing {
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			}
		}
	}

	if pod.DeletionTimestamp != nil && pod.Status.Reason == NodeUnreachablePodReason {
		reason = "Unknown"
	} else if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}

	return reason
}

func tableAlreadyExist(err error) bool {
	return strings.Contains(err.Error(), "already exists")
}
