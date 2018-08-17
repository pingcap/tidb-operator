package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	pdScaleOutTo   = 5
	tikvScaleOutTo = 5
	tidbScaleOutTo = 3
	pdScaleInTo    = 3
	tikvScaleInTo  = 3
	tidbScaleInTo  = 2
)

var podUIDsBeforeScale map[string]types.UID

func testScale() {
	By(fmt.Sprintf("When scale out TiDB cluster: pd ==> [%d], tikv ==> [%d], tidb ==> [%d]", pdScaleOutTo, tikvScaleOutTo, tidbScaleOutTo))
	err := wait.Poll(5*time.Second, 5*time.Minute, scaleOut)
	Expect(err).NotTo(HaveOccurred())

	By("Then TiDB cluster should scale out successfully")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaled)
	Expect(err).NotTo(HaveOccurred())

	By("And should scaled out correctly")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaledCorrectly)
	Expect(err).NotTo(HaveOccurred())

	By("And the data is correct")
	err = wait.Poll(5*time.Second, 5*time.Minute, dataIsCorrect)
	Expect(err).NotTo(HaveOccurred())

	By(fmt.Sprintf("When scale in TiDB cluster: pd ==> [%d], tikv ==> [%d], tidb ==> [%d]", pdScaleInTo, tikvScaleInTo, tidbScaleInTo))
	err = wait.Poll(5*time.Second, 5*time.Minute, scaleIn)
	Expect(err).NotTo(HaveOccurred())

	By("Then TiDB cluster scale in securely")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaleInSecurely)
	Expect(err).NotTo(HaveOccurred())

	By("And should scale in successfully")
	err = wait.Poll(5*time.Second, 5*time.Minute, scaled)
	Expect(err).NotTo(HaveOccurred())

	By("And should be scaled in correctly")
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
	podUIDsBeforeScale, err = getPodsUID()
	if err != nil {
		return false, nil
	}

	tc.Spec.PD.Replicas = pdScaleOutTo
	tc.Spec.TiKV.Replicas = tikvScaleOutTo
	tc.Spec.TiDB.Replicas = tidbScaleOutTo

	_, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(tc)
	if err != nil {
		logf("failed to update tidbcluster when scale out tidbcluster, error: %v", err)
		return false, nil
	}

	return true, nil
}

func scaled() (bool, error) {
	return allMembersRunning()
}

func scaleIn() (bool, error) {
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tidbcluster when scale in tidbcluster, error: %v", err)
		return false, nil
	}

	if tc.Spec.PD.Replicas <= pdScaleInTo {
		return true, fmt.Errorf("the tidbcluster's pd replicas less then pdScaleInTo: [%d]", pdScaleInTo)
	}
	if tc.Spec.TiKV.Replicas <= tikvScaleInTo {
		return true, fmt.Errorf("the tidbcluster's tikv replicas less then tikvScaleInTo: [%d]", tikvScaleInTo)
	}
	if tc.Spec.TiDB.Replicas <= tidbScaleInTo {
		return true, fmt.Errorf("the tidbcluster's tidb replicas less then tidbScaleInTo: [%d]", tidbScaleInTo)
	}

	podUIDsBeforeScale, err = getPodsUID()
	if err != nil {
		return false, nil
	}
	if err != nil {
		return false, nil
	}

	tc.Spec.PD.Replicas = pdScaleInTo
	tc.Spec.TiKV.Replicas = tikvScaleInTo
	tc.Spec.TiDB.Replicas = tidbScaleInTo

	_, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(tc)
	if err != nil {
		logf("failed to update tidbcluster when scale in tidbcluster, error: %v", err)
		return false, nil
	}

	return true, nil
}

func scaledCorrectly() (bool, error) {
	podUIDs, err := getPodsUID()
	if err != nil {
		logf("failed to get pd pods's uid, error: %v", err)
		return false, nil
	}

	if len(podUIDsBeforeScale) == len(podUIDs) {
		return false, fmt.Errorf("the length of pods before scale equals the length of pods after scale")
	}
	if len(podUIDsBeforeScale) > len(podUIDs) {
		for podName, uidAfter := range podUIDs {
			if uidBefore, ok := podUIDsBeforeScale[podName]; ok && uidBefore != uidAfter {
				return false, fmt.Errorf("pod: [%s] have be recreated", podName)
			}
		}
	} else {
		for podName, uidBefore := range podUIDsBeforeScale {
			if uidAfter, ok := podUIDs[podName]; ok && uidBefore != uidAfter {
				return false, fmt.Errorf("pod: [%s] have be recreated", podName)
			}
		}
	}

	return true, nil
}

// scaleInSecurity confirms member scale in securely
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
		return false, fmt.Errorf("the pdSet.Spec.Replicas may reduce before tikv complete offline")
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

func getPodsUID() (map[string]types.UID, error) {
	result := map[string]types.UID{}

	selector, err := label.New().Cluster(clusterName).PD().Selector()
	if err != nil {
		return nil, err
	}
	pdPods, err := kubeCli.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	for _, pod := range pdPods.Items {
		result[pod.GetName()] = pod.GetUID()
	}

	selector, err = label.New().Cluster(clusterName).TiKV().Selector()
	if err != nil {
		return nil, err
	}
	tikvPods, err := kubeCli.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	for _, pod := range tikvPods.Items {
		result[pod.GetName()] = pod.GetUID()
	}

	selector, err = label.New().Cluster(clusterName).TiDB().Selector()
	if err != nil {
		return nil, err
	}
	tidbPods, err := kubeCli.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	for _, pod := range tidbPods.Items {
		result[pod.GetName()] = pod.GetUID()
	}

	return result, nil
}
