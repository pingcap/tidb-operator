package tests

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	// To register MySQL driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	"github.com/pingcap/tidb-operator/tests/pkg/ops"
	"github.com/pingcap/tidb-operator/tests/slack"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (oa *operatorActions) TruncateSSTFileThenCheckFailover(info *TidbClusterConfig, tikvFailoverPeriod time.Duration) error {
	const failoverTimeout = 5 * time.Minute

	cli := client.Union(oa.kubeCli, oa.cli)
	tikvOps := ops.TiKVOps{ClientOps: ops.ClientOps{Client: cli}}

	// checkout latest tidb cluster
	tc, err := cli.PingcapV1alpha1().TidbClusters(info.Namespace).Get(info.ClusterName, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("failed to get the cluster: ns=%s tc=%s err=%s", info.Namespace, info.ClusterName, err.Error())
		return err
	}

	// checkout pd config
	pdCfg, err := oa.pdControl.GetPDClient(tc).GetConfig()
	if err != nil {
		glog.Errorf("failed to get the pd config: tc=%s err=%s", info.ClusterName, err.Error())
		return err
	}
	maxStoreDownTime := pdCfg.Schedule.MaxStoreDownTime.Duration
	glog.Infof("truncate sst file failover config: maxStoreDownTime=%v tikvFailoverPeriod=%v", maxStoreDownTime, tikvFailoverPeriod)

	// find an up store
	var store v1alpha1.TiKVStore
	for _, v := range tc.Status.TiKV.Stores {
		if v.State != v1alpha1.TiKVStateUp {
			continue
		}
		store = v
		break
	}
	if len(store.ID) == 0 {
		glog.Errorf("failed to find an up store")
		return errors.New("no up store for truncating sst file")
	}
	glog.Infof("truncate sst file target store: id=%s pod=%s", store.ID, store.PodName)

	oa.EmitEvent(info, fmt.Sprintf("TruncateSSTFile: tikv: %s", store.PodName))
	glog.Infof("deleting pod: [%s/%s] and wait 1 minute for the pod to terminate", info.Namespace, store.PodName)
	err = cli.CoreV1().Pods(info.Namespace).Delete(store.PodName, nil)
	if err != nil {
		glog.Errorf("failed to get delete the pod: ns=%s tc=%s pod=%s err=%s",
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
		glog.Errorf("failed to truncate the sst file: ns=%s tc=%s store=%s err=%s",
			info.Namespace, info.ClusterName, store.ID, err.Error())
		return err
	}
	oa.EmitEvent(info, fmt.Sprintf("TruncateSSTFile: tikv: %s/%s", info.Namespace, store.PodName))

	// delete tikv pod
	glog.Infof("deleting pod: [%s/%s] again", info.Namespace, store.PodName)
	wait.Poll(10*time.Second, time.Minute, func() (bool, error) {
		err = oa.kubeCli.CoreV1().Pods(info.Namespace).Delete(store.PodName, &metav1.DeleteOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})

	tikvOps.SetPoll(DefaultPollInterval, maxStoreDownTime+tikvFailoverPeriod+failoverTimeout)

	return tikvOps.PollTiDBCluster(info.Namespace, info.ClusterName,
		func(tc *v1alpha1.TidbCluster, err error) (bool, error) {
			_, ok := tc.Status.TiKV.FailureStores[store.ID]
			glog.Infof("cluster: [%s/%s] check if target store failed: %t",
				info.Namespace, info.ClusterName, ok)
			if !ok {
				return false, nil
			}
			return true, nil
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
		glog.Infof("cluster:[%s] query pods failed,error:%v", info.FullName(), err)
		return false, nil
	}
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
			for _, failureMember := range tc.Status.PD.FailureMembers {
				if _, exist := affectedPods[failureMember.PodName]; exist {
					err := fmt.Errorf("cluster: [%s] the pd member[%s] should be mark failure after %s", info.FullName(), failureMember.PodName, deadline.Format(time.RFC3339))
					glog.Errorf(err.Error())
					return false, err
				}
			}
		}
		if tc.Status.TiKV.FailureStores != nil && len(tc.Status.TiKV.FailureStores) > 0 {
			for _, failureStore := range tc.Status.TiKV.FailureStores {
				if _, exist := affectedPods[failureStore.PodName]; exist {
					err := fmt.Errorf("cluster: [%s] the tikv store[%s] should be mark failure after %s", info.FullName(), failureStore.PodName, deadline.Format(time.RFC3339))
					glog.Errorf(err.Error())
					return false, err
				}
			}

		}
		if tc.Status.TiDB.FailureMembers != nil && len(tc.Status.TiDB.FailureMembers) > 0 {
			for _, failureMember := range tc.Status.TiDB.FailureMembers {
				if _, exist := affectedPods[failureMember.PodName]; exist {
					err := fmt.Errorf("cluster: [%s] the tidb member[%s] should be mark failure after %s", info.FullName(), failureMember.PodName, deadline.Format(time.RFC3339))
					glog.Errorf(err.Error())
					return false, err
				}
			}
		}

		glog.Infof("cluster: [%s] operator's failover feature is pending", info.FullName())
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
		glog.Infof("cluster:[%s] query pods failed,error:%v", info.FullName(), err)
		return false, nil
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

func (oa *operatorActions) getPodsByNode(info *TidbClusterConfig, node string) (map[string]*corev1.Pod, error) {
	selector, err := label.New().Instance(info.ClusterName).Selector()
	if err != nil {
		glog.Errorf("cluster:[%s] create selector failed, error:%v", info.FullName(), err)
		return nil, err
	}
	pods, err := oa.kubeCli.CoreV1().Pods(info.Namespace).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		glog.Errorf("cluster:[%s] query pods failed, error:%v", info.FullName(), err)
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
		slack.NotifyAndPanic(fmt.Errorf("failed to check failover"))
	}
}

func (oa *operatorActions) CheckRecover(cluster *TidbClusterConfig) (bool, error) {
	tc, err := oa.cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
	if err != nil {
		return false, nil
	}

	if tc.Status.PD.FailureMembers != nil && len(tc.Status.PD.FailureMembers) > 0 {
		glog.Infof("cluster: [%s]'s pd FailureMembers is not nil, continue to wait", cluster.FullName())
		return false, nil
	}

	if tc.Status.TiDB.FailureMembers != nil && len(tc.Status.TiDB.FailureMembers) > 0 {
		glog.Infof("cluster: [%s]'s tidb FailureMembers is not nil, continue to wait", cluster.FullName())
		return false, nil
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
		glog.Infof("tidbCluster:[%s/%s]'s store pod:[%s] have not become failuremember", tc.Namespace, tc.Name, pod.Name)
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

func (oa *operatorActions) CheckOneEtcdDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig, faultNode string) {
	glog.Infof("check k8s/operator/tidbCluster status when one etcd down")
	KeepOrDie(3*time.Second, 10*time.Minute, func() error {
		err := oa.CheckK8sAvailable(nil, nil)
		if err != nil {
			return err
		}
		glog.V(4).Infof("k8s cluster is available.")
		err = oa.CheckOperatorAvailable(operatorConfig)
		if err != nil {
			return err
		}
		glog.V(4).Infof("tidb operator is available.")
		err = oa.CheckTidbClustersAvailable(clusters)
		if err != nil {
			return err
		}
		glog.V(4).Infof("all clusters are available")
		return nil
	})
}

func (oa *operatorActions) CheckKubeProxyDownOrDie(clusters []*TidbClusterConfig) {
	glog.Infof("checking k8s/tidbCluster status when kube-proxy down")

	KeepOrDie(3*time.Second, 10*time.Minute, func() error {
		err := oa.CheckK8sAvailable(nil, nil)
		if err != nil {
			return err

		}
		glog.V(4).Infof("k8s cluster is available.")
		err = oa.CheckTidbClustersAvailable(clusters)
		if err != nil {
			return err
		}
		glog.V(4).Infof("all clusters are available.")
		return nil
	})
}

func (oa *operatorActions) CheckOneApiserverDownOrDie(operatorConfig *OperatorConfig, clusters []*TidbClusterConfig, faultNode string) {
	glog.Infof("check k8s/operator/tidbCluster status when one apiserver down")
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
		affectedPods[dnsPod.GetName()] = proxyPod
	}
	KeepOrDie(3*time.Second, 10*time.Minute, func() error {
		err := oa.CheckK8sAvailable(map[string]string{faultNode: faultNode}, affectedPods)
		if err != nil {
			return err
		}
		glog.V(4).Infof("k8s cluster is available.")
		err = oa.CheckOperatorAvailable(operatorConfig)
		if err != nil {
			return err
		}
		glog.V(4).Infof("tidb operator is available.")
		err = oa.CheckTidbClustersAvailable(clusters)
		if err != nil {
			return err
		}
		glog.V(4).Infof("all clusters is available")
		return nil
	})
}

func (oa *operatorActions) CheckOperatorDownOrDie(clusters []*TidbClusterConfig) {
	glog.Infof("checking k8s/tidbCluster status when operator down")

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
	return wait.Poll(3*time.Second, time.Minute, func() (bool, error) {
		nodes, err := oa.kubeCli.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			glog.Errorf("failed to list nodes,error:%v", err)
			return false, nil
		}
		for _, node := range nodes.Items {
			if _, exist := excludeNodes[node.GetName()]; exist {
				continue
			}
			for _, condition := range node.Status.Conditions {
				if condition.Type == corev1.NodeReady && condition.Status != corev1.ConditionTrue {
					return false, fmt.Errorf("node: [%s] is not in running", node.GetName())
				}
			}
		}
		systemPods, err := oa.kubeCli.CoreV1().Pods("kube-system").List(metav1.ListOptions{})
		if err != nil {
			glog.Errorf("failed to list kube-system pods,error:%v", err)
			return false, nil
		}
		for _, pod := range systemPods.Items {
			if _, exist := excludePods[pod.GetName()]; exist {
				continue
			}
			podState := GetPodStatus(&pod)
			if podState != string(corev1.PodRunning) {
				return false, fmt.Errorf("pod:[%s/%s] is unavailable,state is %s", pod.GetNamespace(), pod.GetName(), podState)
			}
		}
		return true, nil
	})
}

func (oa *operatorActions) CheckOperatorAvailable(operatorConfig *OperatorConfig) error {
	return wait.Poll(3*time.Second, 3*time.Minute, func() (bool, error) {
		controllerDeployment, err := oa.kubeCli.AppsV1().Deployments(operatorConfig.Namespace).Get(tidbControllerName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("failed to get deployment：%s failed,error:%v", tidbControllerName, err)
			return false, nil
		}
		if controllerDeployment.Status.AvailableReplicas != *controllerDeployment.Spec.Replicas {
			return false, fmt.Errorf("the %s is not available", tidbControllerName)
		}
		schedulerDeployment, err := oa.kubeCli.AppsV1().Deployments(operatorConfig.Namespace).Get(tidbSchedulerName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("failed to get deployment：%s failed,error:%v", tidbSchedulerName, err)
			return false, nil
		}
		if schedulerDeployment.Status.AvailableReplicas != *schedulerDeployment.Spec.Replicas {
			return false, fmt.Errorf("the %s is not available", tidbSchedulerName)
		}
		return true, nil
	})
}

func (oa *operatorActions) CheckTidbClustersAvailable(infos []*TidbClusterConfig) error {
	return wait.Poll(3*time.Second, 30*time.Second, func() (bool, error) {
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
	db, err := sql.Open("mysql", getDSN(info.Namespace, info.ClusterName, "test", info.Password))
	if err != nil {
		glog.Errorf("cluster:[%s] can't open connection to mysql: %v", info.FullName(), err)
		return false, nil
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE TABLE %s (name VARCHAR(64))", testTableName))
	if err != nil && !tableAlreadyExist(err) {
		glog.Errorf("cluster:[%s] can't create table to mysql: %v", info.FullName(), err)
		return false, nil
	}

	_, err = db.Exec(fmt.Sprintf("INSERT INTO %s VALUES (?)", testTableName), "testValue")
	if err != nil {
		glog.Errorf("cluster:[%s] can't insert data to mysql: %v", info.FullName(), err)
		return false, nil
	}

	return true, nil
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
