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

package webhook

import (
	"fmt"
	"os"
	"time"

	"github.com/pingcap/tidb-operator/tests/slack"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/tests/pkg/client"

	"k8s.io/api/admission/v1beta1"
	"k8s.io/klog"
)

// only allow pods to be delete when it is not ddlowner of tidb, not leader of pd and not
// leader of tikv.
func (wh *webhook) admitPods(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	klog.V(4).Infof("admitting pods")

	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if ar.Request.Resource != podResource {
		err := fmt.Errorf("expect resource to be %s", podResource)
		klog.Errorf("%v", err)
		return toAdmissionResponse(err)
	}

	versionCli, kubeCli, _, _, _ := client.NewCliOrDie()

	name := ar.Request.Name
	namespace := ar.Request.Namespace

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = false

	if !wh.namespaces.Has(namespace) {
		klog.V(4).Infof("%q is not in our namespaces %v, skip", namespace, wh.namespaces.List())
		reviewResponse.Allowed = true
		return &reviewResponse
	}

	pod, err := kubeCli.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		klog.Infof("api server send wrong pod info namespace %s name %s err %v", namespace, name, err)
		return &reviewResponse
	}

	klog.V(4).Infof("delete %s pod [%s]", pod.Labels[label.ComponentLabelKey], pod.GetName())

	tc, err := versionCli.PingcapV1alpha1().TidbClusters(namespace).Get(pod.Labels[label.InstanceLabelKey], metav1.GetOptions{})
	if err != nil {
		klog.Infof("fail to fetch tidbcluster info namespace %s clustername(instance) %s err %v", namespace, pod.Labels[label.InstanceLabelKey], err)
		return &reviewResponse
	}

	pdClient := pdapi.NewDefaultPDControl(kubeCli).GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled())

	// if pod is already deleting, return Allowed
	if pod.DeletionTimestamp != nil {
		klog.V(4).Infof("pod:[%s/%s] status is timestamp %s", namespace, name, pod.DeletionTimestamp)
		reviewResponse.Allowed = true
		return &reviewResponse
	}

	if pod.Labels[label.ComponentLabelKey] == "pd" {

		leader, err := pdClient.GetPDLeader()
		if err != nil {
			klog.Errorf("fail to get pd leader %v", err)
			return &reviewResponse
		}

		if leader.Name == name && tc.Status.PD.StatefulSet.Replicas > 1 {
			time.Sleep(10 * time.Second)
			err := fmt.Errorf("pd is leader, can't be deleted namespace %s name %s", namespace, name)
			klog.Error(err)
			sendErr := slack.SendErrMsg(err.Error())
			if sendErr != nil {
				klog.Error(sendErr)
			}
			// TODO use context instead
			os.Exit(3)
		}
		klog.Infof("savely delete pod namespace %s name %s leader name %s", namespace, name, leader.Name)
	}
	reviewResponse.Allowed = true
	return &reviewResponse
}
