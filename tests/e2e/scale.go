package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/apimachinery/pkg/types"
)

const (
	pdScaleNum   = 2
	tikvScaleNum = 2
	tidbScaleNum = 1
)

var pdPodUIDsBeforeScale map[string]types.UID
var tikvPodUIDsBeforeScale map[string]types.UID
var tidbPodUIDsBeforeScale map[string]types.UID

func testScale() {
	By("When scale out TiDB cluster")
	err := wait.Poll(5*time.Second, 5*time.Minute, scaleOut)
	Expect(err).NotTo(HaveOccurred())

	By("Then all members should running")
	err = wait.Poll(5*time.Second, 5*time.Minute, allMembersRunning)
	Expect(err).NotTo(HaveOccurred())

	By("Then should be scaled out correctly")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaledCorrectly)
	Expect(err).NotTo(HaveOccurred())

	By("And the data is correct")
	err = wait.Poll(5*time.Second, 5*time.Minute, dataIsCorrect)
	Expect(err).NotTo(HaveOccurred())

	By("When scale in TiDB cluster")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaleIn)
	Expect(err).NotTo(HaveOccurred())

	By("then scale in TiDB cluster securely")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaleInSecurely)
	Expect(err).NotTo(HaveOccurred())

	By("Then all members should running")
	err = wait.Poll(5*time.Second, 5*time.Minute, allMembersRunning)
	Expect(err).NotTo(HaveOccurred())

	By("Then should be scaled in correctly")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaledCorrectly)
	Expect(err).NotTo(HaveOccurred())

	By("And the data is correct")
	err = wait.Poll(5*time.Second, 5*time.Minute, dataIsCorrect)
	Expect(err).NotTo(HaveOccurred())

}

func scaleOut() (bool, error) {
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tidbcluster when scale out tidbcluster, error: %v", err)
		return false, nil
	}
	pdPodUIDsBeforeScale, err = getPodsUID(v1alpha1.PDMemberType)
	if err != nil {
		return false, nil
	}
	tikvPodUIDsBeforeScale, err = getPodsUID(v1alpha1.TiKVMemberType)
	if err != nil {
		return false, nil
	}
	tidbPodUIDsBeforeScale, err = getPodsUID(v1alpha1.TiDBMemberType)
	if err != nil {
		return false, nil
	}

	tc.Spec.PD.Replicas = tc.Spec.PD.Replicas + pdScaleNum
	tc.Spec.TiKV.Replicas = tc.Spec.TiKV.Replicas + tikvScaleNum
	tc.Spec.TiDB.Replicas = tc.Spec.TiDB.Replicas + tidbScaleNum

	_, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(tc)
	if err != nil {
		logf("failed to update tidbcluster when scale out tidbcluster, error: %v", err)
		return false, nil
	}

	return true, nil
}

func scaleIn() (bool, error) {
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tidbcluster when scale in tidbcluster, error: %v", err)
		return false, nil
	}

	if tc.Spec.PD.Replicas <= pdScaleNum {
		return true, fmt.Errorf("the tidbcluster's pd replicas less then pdScaleNum: [%d]", pdScaleNum)
	}
	if tc.Spec.TiKV.Replicas <= tikvScaleNum {
		return true, fmt.Errorf("the tidbcluster's tikv replicas less then tikvScaleNum: [%d]", tikvScaleNum)
	}
	if tc.Spec.TiDB.Replicas <= tidbScaleNum {
		return true, fmt.Errorf("the tidbcluster's tidb replicas less then tikvScaleNum: [%d]", tidbScaleNum)
	}

	pdPodUIDsBeforeScale, err = getPodsUID(v1alpha1.PDMemberType)
	if err != nil {
		return false, nil
	}
	tikvPodUIDsBeforeScale, err = getPodsUID(v1alpha1.TiKVMemberType)
	if err != nil {
		return false, nil
	}
	tidbPodUIDsBeforeScale, err = getPodsUID(v1alpha1.TiDBMemberType)
	if err != nil {
		return false, nil
	}

	tc.Spec.PD.Replicas = tc.Spec.PD.Replicas - pdScaleNum
	tc.Spec.TiKV.Replicas = tc.Spec.TiKV.Replicas - tikvScaleNum
	tc.Spec.TiDB.Replicas = tc.Spec.TiDB.Replicas - tidbScaleNum

	_, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(tc)
	if err != nil {
		logf("failed to update tidbcluster when scale in tidbcluster, error: %v", err)
		return false, nil
	}

	return true, nil
}

func scaledCorrectly() (bool, error) {

	pdUIDs, err := getPodsUID(v1alpha1.PDMemberType)
	if err != nil {
		logf("failed to get pd pod uids")
		return false, nil
	}

	if len(pdPodUIDsBeforeScale) == len(pdUIDs) {
		return false, fmt.Errorf("")
	}
	if len(pdPodUIDsBeforeScale) > len(pdUIDs) {
		for podName, uid_now := range pdUIDs {
			if uid_before, ok := pdPodUIDsBeforeScale[podName]; ok && uid_before != uid_now {
				return false, fmt.Errorf("pod: [%s] have be recreate", podName)
			}
		}
	} else {
		for podName, uid_before := range pdPodUIDsBeforeScale {
			if uid_now, ok := pdUIDs[podName]; ok && uid_before != uid_now {
				return false, fmt.Errorf("pod: [%s] have be recreate", podName)
			}
		}
	}

	tikvUIDs, err := getPodsUID(v1alpha1.TiKVMemberType)
	if err != nil {
		logf("failed to get tikv pod uids")
		return false, nil
	}

	if len(tikvPodUIDsBeforeScale) == len(tikvUIDs) {
		return false, fmt.Errorf("")
	}
	if len(tikvPodUIDsBeforeScale) > len(tikvUIDs) {
		for podName, uid_now := range tikvUIDs {
			if uid_before, ok := tikvPodUIDsBeforeScale[podName]; ok && uid_before != uid_now {
				return false, fmt.Errorf("pod: [%s] have be recreate", podName)
			}
		}
	} else {
		for podName, uid_before := range tikvPodUIDsBeforeScale {
			if uid_now, ok := tikvUIDs[podName]; ok && uid_before != uid_now {
				return false, fmt.Errorf("pod: [%s] have be recreate", podName)
			}
		}
	}

	tidbUIDs, err := getPodsUID(v1alpha1.TiDBMemberType)
	if err != nil {
		logf("failed to get tidb pod uids")
		return false, nil
	}

	if len(tidbPodUIDsBeforeScale) == len(tidbUIDs) {
		return false, fmt.Errorf("the length of pods before scale equals pods after scale")
	}
	if len(tidbPodUIDsBeforeScale) > len(tidbUIDs) {
		for podName, uid_now := range tidbUIDs {
			if uid_before, ok := tidbPodUIDsBeforeScale[podName]; ok && uid_before != uid_now {
				return false, fmt.Errorf("pod: [%s] have be recreate", podName)
			}
		}
	} else {
		for podName, uid_before := range tidbPodUIDsBeforeScale {
			if uid_now, ok := tidbUIDs[podName]; ok && uid_before != uid_now {
				return false, fmt.Errorf("pod: [%s] have be recreate", podName)
			}
		}
	}

	return true, nil
}

// scaleInSecurity confirm member scale in securely
func scaleInSecurely() (bool, error) {
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tidbcluster when scale in tidbcluster, error: %v", err)
		return false, nil
	}

	pdSetName := controller.PDMemberName(clusterName)
	pdSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(pdSetName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get pd statefulset: [%s], error: %v", pdSetName, err)
		return false, nil
	}

	tikvSetName := controller.TiKVMemberName(clusterName)
	tikvSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(tikvSetName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tikvSet statefulset: [%s], error: %v", tikvSetName, err)
		return false, nil
	}

	tidbSetName := controller.TiDBMemberName(clusterName)
	tidbSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(tidbSetName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tikvSet statefulset: [%s], error: %v", tidbSetName, err)
		return false, nil
	}

	if len(tc.Status.TiKV.Stores.CurrentStores) > int(*tikvSet.Spec.Replicas) {
		return false, fmt.Errorf("the tc.Status.PD.Members length less then pdSet.Spec.Replicas")
	}

	if !(tc.Spec.PD.Replicas == pdSet.Status.ReadyReplicas && pdSet.Status.Replicas == pdSet.Status.ReadyReplicas) {
		return false, nil
	}

	if !(tc.Spec.TiKV.Replicas == tikvSet.Status.ReadyReplicas && tikvSet.Status.Replicas == tikvSet.Status.ReadyReplicas) {
		return false, nil
	}

	if !(tc.Spec.TiDB.Replicas == tidbSet.Status.ReadyReplicas && tidbSet.Status.Replicas == tidbSet.Status.ReadyReplicas) {
		return false, nil
	}

	return true, nil
}

func getPodsUID(memberType v1alpha1.MemberType) (map[string]types.UID, error) {
	var l label.Label
	switch memberType {
	case v1alpha1.PDMemberType:
		l = label.New().Cluster(clusterName).PD()
		break
	case v1alpha1.TiKVMemberType:
		l = label.New().Cluster(clusterName).TiKV()
		break
	case v1alpha1.TiDBMemberType:
		l = label.New().Cluster(clusterName).TiDB()
		break
	}

	selector, err := l.Selector()
	if err != nil {
		return nil, err
	}

	pods, err := kubeCli.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	result := map[string]types.UID{}
	for _, pod := range pods.Items {
		result[pod.GetName()] = pod.GetUID()
	}

	return result, nil
}
