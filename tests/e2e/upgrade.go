package e2e

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func testUpgrade() {
	By("Then start upgrade to new version")
	err := wait.Poll(5*time.Second, 5*time.Minute, upgrade)
	Expect(err).NotTo(HaveOccurred())

	By("Then members should upgrade by order: pd ==> tikv ==> tidb")
	err = wait.Poll(5*time.Second, 5*time.Minute, memberUpgraded)
	Expect(err).NotTo(HaveOccurred())

	By("Then all members should running")
	err = wait.Poll(5*time.Second, 5*time.Minute, allMembersRunning)
	Expect(err).NotTo(HaveOccurred())

	By("Then the data is correct")
	err = wait.Poll(5*time.Second, 5*time.Minute, dataIsCorrect)
	Expect(err).NotTo(HaveOccurred())
}

func upgrade() (bool, error) {
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		logf("failed to upgrade tidbcluster, error: v%", err)
		return false, err
	}

	tc.Spec.PD.Image = strings.Replace(tc.Spec.PD.Image, getImageTag(tc.Spec.PD.Image), "v2.0.0", -1)
	tc.Spec.TiKV.Image = strings.Replace(tc.Spec.TiKV.Image, getImageTag(tc.Spec.TiKV.Image), "v2.0.0", -1)
	tc.Spec.TiDB.Image = strings.Replace(tc.Spec.TiDB.Image, getImageTag(tc.Spec.TiDB.Image), "v2.0.0", -1)

	_, err = cli.PingcapV1alpha1().TidbClusters(ns).Update(tc)
	if err != nil {
		logf("failed to upgrade tidbcluster, error: v%", err)
		return false, err
	}

	return true, nil
}

func memberUpgraded() (bool, error) {
	tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tidbcluster: [%s], error: %v", clusterName, err)
		return false, err
	}
	pdSetName := controller.PDMemberName(tc.GetName())
	pdSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(pdSetName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get pd statefulset: [%s], error: %v", pdSetName, err)
		return false, err
	}

	tikvSetName := controller.TiKVMemberName(tc.GetName())
	tikvSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(tikvSetName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tikvSet statefulset: [%s], error: %v", pdSetName, err)
		return false, err
	}

	tidbSetName := controller.TiDBMemberName(tc.GetName())
	tidbSet, err := kubeCli.AppsV1beta1().StatefulSets(ns).Get(tidbSetName, metav1.GetOptions{})
	if err != nil {
		logf("failed to get tikvSet statefulset: [%s], error: %v", pdSetName, err)
		return false, err
	}

	if !imageUpgraded(tc, v1alpha1.PDMemberType, pdSet) {
		return false, nil
	}

	if tc.Status.PD.Phase == v1alpha1.Upgrade {
		Expect(tc.Status.TiKV.Phase).NotTo(Equal(v1alpha1.Upgrade))
		Expect(tc.Status.TiDB.Phase).NotTo(Equal(v1alpha1.Upgrade))
		Expect(imageUpgraded(tc, v1alpha1.PDMemberType, pdSet)).To(BeTrue())
		Expect(imageUpgraded(tc, v1alpha1.TiKVMemberType, tikvSet)).To(BeFalse())
		Expect(imageUpgraded(tc, v1alpha1.TiDBMemberType, tidbSet)).To(BeFalse())
		return false, nil
	} else if tc.Status.TiKV.Phase == v1alpha1.Upgrade {
		Expect(tc.Status.TiDB.Phase).NotTo(Equal(v1alpha1.Upgrade))
		Expect(imageUpgraded(tc, v1alpha1.TiKVMemberType, tikvSet)).To(BeTrue())
		Expect(imageUpgraded(tc, v1alpha1.TiDBMemberType, tidbSet)).To(BeFalse())
		upgraded, err := podsUpgraded(tc, v1alpha1.PDMemberType, pdSet)
		if err != nil {
			logf("check pd's pods upgraded failed: %v", err)
			return false, nil
		}
		Expect(upgraded).To(BeTrue())
	} else if tc.Status.TiDB.Phase == v1alpha1.Upgrade {
		Expect(imageUpgraded(tc, v1alpha1.TiDBMemberType, tidbSet)).To(BeTrue())
		upgraded, err := podsUpgraded(tc, v1alpha1.PDMemberType, pdSet)
		if err != nil {
			logf("check pd's pods upgraded failed: %v", err)
			return false, nil
		}
		Expect(upgraded).To(BeTrue())
		upgraded, err = podsUpgraded(tc, v1alpha1.TiKVMemberType, tikvSet)
		if err != nil {
			logf("check tikv's pods upgraded failed: %v", err)
			return false, nil
		}
		Expect(upgraded).To(BeTrue())
		return false, nil
	} else {
		upgraded, err := podsUpgraded(tc, v1alpha1.PDMemberType, pdSet)
		if err != nil {
			logf("check pd's pods upgraded failed: %v", err)
			return false, nil
		}
		Expect(upgraded).To(BeTrue())
		upgraded, err = podsUpgraded(tc, v1alpha1.TiKVMemberType, tikvSet)
		if err != nil {
			logf("check tikv's pods upgraded failed: %v", err)
			return false, nil
		}
		Expect(upgraded).To(BeTrue())
		upgraded, err = podsUpgraded(tc, v1alpha1.TiDBMemberType, tidbSet)
		if err != nil {
			logf("check tikv's pods upgraded failed: %v", err)
			return false, nil
		}
		Expect(upgraded).To(BeTrue())
		return true, nil
	}
	return false, nil
}

func imageUpgraded(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, set *apps.StatefulSet) bool {
	for _, container := range set.Spec.Template.Spec.Containers {
		if container.Name == memberType.String() {
			if container.Image == getImage(tc, memberType) {
				return true
			}
		}
	}
	return false
}

func podsUpgraded(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, set *apps.StatefulSet) (bool, error) {
	selector, err := label.New().Cluster(tc.GetName()).PD().Selector()
	if err != nil {
		return false, err
	}
	pods, err := kubeCli.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return false, err
	}

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if container.Image == memberType.String() {
				if container.Image != getImage(tc, memberType) {
					return false, nil
				}
			}
		}
	}

	return true, nil
}

func getImage(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) string {
	switch memberType {
	case v1alpha1.PDMemberType:
		return tc.Spec.PD.Image
	case v1alpha1.TiKVMemberType:
		return tc.Spec.TiKV.Image
	case v1alpha1.TiDBMemberType:
		return tc.Spec.TiDB.Image
	default:
		return ""
	}
}

func getImageTag(image string) string {
	strs := strings.Split(image, ":")
	return strs[len(strs)-1]
}
