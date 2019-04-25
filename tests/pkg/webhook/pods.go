package webhook

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	"k8s.io/api/admission/v1beta1"
)

var (
	// Pod name may the same in different namespaces
	kvLeaderMap map[string]map[string]int
)

func GetAllKVLeaders(versionCli versioned.Interface, namespace string, clusterName string) error {

	if kvLeaderMap == nil {
		kvLeaderMap = make(map[string]map[string]int)
	}

	if kvLeaderMap[namespace] == nil {
		kvLeaderMap[namespace] = make(map[string]int)
	}

	tc, err := versionCli.PingcapV1alpha1().TidbClusters(namespace).Get(clusterName, metav1.GetOptions{})

	if err != nil {
		glog.Infof("fail to get tc clustername %s namesapce %s %v", clusterName, namespace, err)
		return err
	}

	pdClient := controller.NewDefaultPDControl().GetPDClient(tc)

	for _, store := range tc.Status.TiKV.Stores {
		storeID, err := strconv.ParseUint(store.ID, 10, 64)
		if err != nil {
			glog.Errorf("fail to convert string to int while deleting TIKV err %v", err)
			return err
		}
		storeInfo, err := pdClient.GetStore(storeID)
		if err != nil {
			glog.Errorf("fail to read response %v", err)
			return err
		}
		kvLeaderMap[namespace][store.PodName] = storeInfo.Status.LeaderCount
	}

	return nil
}

// only allow pods to be delete when it is not ddlowner of tidb, not leader of pd and not
// master of tikv.
func admitPods(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	glog.Infof("admitting pods")

	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if ar.Request.Resource != podResource {
		err := fmt.Errorf("expect resource to be %s", podResource)
		glog.Errorf("%v", err)
		return toAdmissionResponse(err)
	}

	versionCli, kubeCli := client.NewCliOrDie()

	name := ar.Request.Name
	namespace := ar.Request.Namespace

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = false

	pod, err := kubeCli.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		glog.Infof("api server send wrong pod info namespace %s name %s err %v", namespace, name, err)
		return &reviewResponse
	}

	glog.Infof("delete pod %s", pod.Labels[label.ComponentLabelKey])

	tc, err := versionCli.PingcapV1alpha1().TidbClusters(namespace).Get(pod.Labels[label.InstanceLabelKey], metav1.GetOptions{})
	if err != nil {
		glog.Infof("fail to fetch tidbcluster info namespace %s clustername(instance) %s err %v", namespace, pod.Labels[label.InstanceLabelKey], err)
		return &reviewResponse
	}

	pdClient := controller.NewDefaultPDControl().GetPDClient(tc)
	tidbController := controller.NewDefaultTiDBControl()

	if pod.Labels[label.ComponentLabelKey] == "tidb" {

		ordinal, err := strconv.ParseInt(strings.Split(name, "-")[len(strings.Split(name, "-"))-1], 10, 32)
		if err != nil {
			glog.Errorf("fail to convert string to int while deleting TiDB err %v", err)
			return &reviewResponse
		}

		info, err := tidbController.GetInfo(tc, int32(ordinal))
		if err != nil {
			glog.Errorf("fail to get tidb info error:%v", err)
			return &reviewResponse
		}

		if info.IsOwner && tc.Status.TiDB.StatefulSet.Replicas > 1 {
			time.Sleep(10 * time.Second)
			glog.Errorf("tidb is ddl owner, can't be deleted namespace %s name %s", namespace, name)
			os.Exit(3)
		} else {
			glog.Infof("savely delete pod namespace %s name %s isowner %t", namespace, name, info.IsOwner)
		}

	} else if pod.Labels[label.ComponentLabelKey] == "pd" {

		leader, err := pdClient.GetPDLeader()
		if err != nil {
			glog.Errorf("fail to get pd leader %v", err)
			return &reviewResponse
		}

		if leader.Name == name && tc.Status.TiDB.StatefulSet.Replicas > 1 {
			time.Sleep(10 * time.Second)
			glog.Errorf("pd is leader, can't be deleted namespace %s name %s", namespace, name)
			os.Exit(3)
		} else {
			glog.Infof("savely delete pod namespace %s name %s leader name %s", namespace, name, leader.Name)
		}

	} else if pod.Labels[label.ComponentLabelKey] == "tikv" {

		var storeID uint64
		storeID = 0
		for _, store := range tc.Status.TiKV.Stores {
			if store.PodName == name {
				storeID, err = strconv.ParseUint(store.ID, 10, 64)
				if err != nil {
					glog.Errorf("fail to convert string to int while deleting PD err %v", err)
					return &reviewResponse
				}
				break
			}
		}

		// Fail to get store in stores
		if storeID == 0 {
			glog.Errorf("fail to find store in TIKV.Stores podname %s", name)
			return &reviewResponse
		}

		storeInfo, err := pdClient.GetStore(storeID)
		if err != nil {
			glog.Errorf("fail to read storeID %d response %v", storeID, err)
			return &reviewResponse
		}

		beforeCount := kvLeaderMap[namespace][name]
		afterCount := storeInfo.Status.LeaderCount

		if beforeCount != 0 && beforeCount <= afterCount && tc.Status.TiKV.StatefulSet.Replicas > 1 {
			time.Sleep(10 * time.Second)
			glog.Errorf("kv leader is not zero, can't be deleted namespace %s name %s leaderCount %d", namespace, name, storeInfo.Status.LeaderCount)
			os.Exit(3)
		} else {
			glog.Infof("savely delete pod namespace %s name %s before count %d after count %d", namespace, name, beforeCount, afterCount)
		}
	}
	reviewResponse.Allowed = true
	return &reviewResponse
}
