package tests

import (
	"fmt"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (oa *operatorActions) CheckFailoverPending(info *TidbClusterConfig, faultPoint *time.Time) (bool, error) {
	tc, err := oa.cli.PingcapV1alpha1().TidbClusters(info.Namespace).Get(info.ClusterName, metav1.GetOptions{})
	if err != nil {
		glog.Infof("pending failover,failed to get tidbcluster:[%s], error: %v", info.FullName(), err)
		if strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers") {
			glog.Info("create new client")
			newCli, _ := client.NewCliOrDie()
			oa.cli = newCli
		}
		return false, nil
	}
	deadline := faultPoint.Add(period)
	if time.Now().Before(deadline) {
		if tc.Status.PD.FailureMembers != nil && len(tc.Status.PD.FailureMembers) > 0 {
			err := fmt.Errorf("cluster: [%s] the pd member should be mark failure after %s", info.FullName(), deadline.Format(time.RFC3339))
			glog.Errorf(err.Error())
			return false, err
		}
		if tc.Status.TiKV.FailureStores != nil && len(tc.Status.TiKV.FailureStores) > 0 {
			err := fmt.Errorf("cluster: [%s] the tikv store should be mark failure after %s", info.FullName(), deadline.Format(time.RFC3339))
			glog.Errorf(err.Error())
			return false, err
		}
		if tc.Status.TiDB.FailureMembers != nil && len(tc.Status.TiDB.FailureMembers) > 0 {
			err := fmt.Errorf("cluster: [%s] the tidb member should be mark failure after %s", info.FullName(), deadline.Format(time.RFC3339))
			glog.Errorf(err.Error())
			return false, err
		}

		glog.Infof("cluster: [%s] operator's failover feature is pending", info.FullName())
		return false, nil
	}
	return true, nil
}

func (oa *operatorActions) CheckFailoverPendingOrDie(clusters []*TidbClusterConfig, faultPoint *time.Time) {
	if err := wait.Poll(1*time.Minute, 30*time.Minute, func() (bool, error) {
		var passes []bool
		for i := range clusters {
			pass, err := oa.CheckFailoverPending(clusters[i], faultPoint)
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
		panic("failed to check failover pending")
	}
}

func (oa *operatorActions) CheckFailover(info *TidbClusterConfig, node string) (bool, error) {
	selector, err := label.New().Instance(info.ClusterName).Selector()
	if err != nil {
		glog.Errorf("cluster:[%s] create selector failed, error:%v", info.FullName(), err)
		return false, nil
	}
	pods, err := oa.kubeCli.CoreV1().Pods(info.Namespace).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		glog.Errorf("cluster:[%s] query pods failed, error:%v", info.FullName(), err)
		return false, nil
	}

	affectedPods := map[string]*corev1.Pod{}
	for i, pod := range pods.Items {
		if pod.Spec.NodeName == node {
			affectedPods[pod.Name] = &pods.Items[i]
		}
	}
	if len(affectedPods) == 0 {
		glog.Infof("the cluster:[%s] can not be affected by node:[%s]", info.FullName(), node)
		return true, nil
	}

	tc, err := oa.cli.PingcapV1alpha1().TidbClusters(info.Namespace).Get(info.ClusterName, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("query tidbcluster: [%s] failed, error: %v", info.FullName(), err)
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

	glog.Infof("cluster: [%s]'s failover feature has complete", info.FullName())
	return true, nil
}

func (oa *operatorActions) CheckFailoverOrDie(clusters []*TidbClusterConfig, faultNode string) {
	if err := wait.Poll(1*time.Minute, 30*time.Minute, func() (bool, error) {
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
		panic("failed to check failover")
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
		glog.Infof("tidbCluster:[%s/%s]'s member:[%s] have not become failuremember", tc.Namespace, tc.Name, pod.Name)
		return false
	}

	for _, member := range tc.Status.PD.Members {
		if member.Name == pod.GetName() {
			glog.Infof("tidbCluster:[%s/%s]'s status.members still have pd member:[%s]", tc.Namespace, tc.Name, pod.Name)
			return false
		}
	}

	if tc.Status.PD.Synced && len(tc.Status.PD.Members) == int(tc.Spec.PD.Replicas) {
		return true
	}

	glog.Infof("cluster: [%s/%s] pd:[%s] failover still not complete", tc.Namespace, tc.Name, pod.GetName())

	return false
}

func (oa *operatorActions) tikvFailover(pod *corev1.Pod, tc *v1alpha1.TidbCluster) bool {
	failure := false
	for _, failureStore := range tc.Status.TiKV.FailureStores {
		if failureStore.PodName == pod.GetName() {
			failure = true
			break
		}
	}
	if !failure {
		glog.Infof("tidbCluster:[%s/%s]'s store pod:[%s] have not become failuremember", tc.Namespace, tc.Name, pod.Name)
		return false
	}

	healthCount := 0
	for _, store := range tc.Status.TiKV.Stores {
		if store.State == v1alpha1.TiKVStateUp {
			healthCount++
		}
	}
	if tc.Status.TiKV.Synced && healthCount == int(tc.Spec.TiKV.Replicas) {
		return true
	}

	glog.Infof("cluster: [%s/%s] tikv:[%s] failover still not complete", tc.Namespace, tc.Name, pod.GetName())
	return false
}

func (oa *operatorActions) tidbFailover(pod *corev1.Pod, tc *v1alpha1.TidbCluster) bool {
	failure := false
	for _, failureMember := range tc.Status.TiDB.FailureMembers {
		if failureMember.PodName == pod.GetName() {
			glog.Infof("tidbCluster:[%s/%s]'s store pod:[%s] have not become failuremember", tc.Namespace, tc.Name, pod.Name)
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
	glog.Infof("cluster: [%s/%s] tidb:[%s] failover still not complete", tc.Namespace, tc.Name, pod.GetName())
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
