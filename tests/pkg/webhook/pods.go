package webhook

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/tests/slack"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	"k8s.io/api/admission/v1beta1"
	glog "k8s.io/klog"
)

// only allow pods to be delete when it is not ddlowner of tidb, not leader of pd and not
// master of tikv.
func admitPods(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	glog.V(4).Infof("admitting pods")

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

	glog.V(4).Infof("delete %s pod [%s]", pod.Labels[label.ComponentLabelKey], pod.GetName())

	tc, err := versionCli.PingcapV1alpha1().TidbClusters(namespace).Get(pod.Labels[label.InstanceLabelKey], metav1.GetOptions{})
	if err != nil {
		glog.Infof("fail to fetch tidbcluster info namespace %s clustername(instance) %s err %v", namespace, pod.Labels[label.InstanceLabelKey], err)
		return &reviewResponse
	}

	pdClient := pdapi.NewDefaultPDControl(kubeCli).GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.Spec.EnableTLSCluster)
	tidbController := controller.NewDefaultTiDBControl()

	// if pod is already deleting, return Allowed
	if pod.DeletionTimestamp != nil {
		glog.V(4).Infof("pod:[%s/%s] status is timestamp %s", namespace, name, pod.DeletionTimestamp)
		reviewResponse.Allowed = true
		return &reviewResponse
	}

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
			err := fmt.Errorf("tidb is ddl owner, can't be deleted namespace %s name %s", namespace, name)
			glog.Error(err)
			sendErr := slack.SendErrMsg(err.Error())
			if sendErr != nil {
				glog.Error(sendErr)
			}
			// TODO use context instead
			os.Exit(3)
		}
		glog.Infof("savely delete pod namespace %s name %s isowner %t", namespace, name, info.IsOwner)

	} else if pod.Labels[label.ComponentLabelKey] == "pd" {

		leader, err := pdClient.GetPDLeader()
		if err != nil {
			glog.Errorf("fail to get pd leader %v", err)
			return &reviewResponse
		}

		if leader.Name == name && tc.Status.PD.StatefulSet.Replicas > 1 {
			time.Sleep(10 * time.Second)
			err := fmt.Errorf("pd is leader, can't be deleted namespace %s name %s", namespace, name)
			glog.Error(err)
			sendErr := slack.SendErrMsg(err.Error())
			if sendErr != nil {
				glog.Error(sendErr)
			}
			// TODO use context instead
			os.Exit(3)
		}
		glog.Infof("savely delete pod namespace %s name %s leader name %s", namespace, name, leader.Name)
	}
	reviewResponse.Allowed = true
	return &reviewResponse
}
