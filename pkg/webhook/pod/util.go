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

package pod

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	pdutil "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func IsPodInPdMembers(tc *v1alpha1.TidbCluster, pod *core.Pod, pdClient pdapi.PDClient) (bool, error) {
	name := pod.Name
	namespace := pod.Namespace
	memberInfo, err := pdClient.GetMembers()
	if err != nil {
		return true, fmt.Errorf("tc[%s/%s] failed to get pd memberInfo during delete pod[%s/%s],%v", namespace, tc.Name, namespace, name, err)
	}
	for _, member := range memberInfo.Members {
		if member.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func IsStatefulSetUpgrading(set *v1.StatefulSet) bool {
	return !(set.Status.CurrentRevision == set.Status.UpdateRevision)
}

// each time we scale in pd replicas, we won't delete the pvc which belong to the
// pd pod who would be deleted by statefulset controller
// we add annotations to this pvc and delete it when we scale out the pd replicas
// for the new pd pod need new pvc
func addDeferDeletingToPVC(podAC *PodAdmissionControl, tc *v1alpha1.TidbCluster, setName, namespace string, ordinal int32) error {
	pvcName := operatorUtils.OrdinalPVCName(v1alpha1.PDMemberType, setName, ordinal)
	pvc, err := podAC.pvcControl.GetPVC(pvcName, namespace)
	if err != nil {
		return err
	}
	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}
	now := time.Now().Format(time.RFC3339)
	pvc.Annotations[label.AnnPVCDeferDeleting] = now
	_, err = podAC.pvcControl.UpdatePVC(tc, pvc)
	return err
}

// check whether the former upgraded pd pods were healthy in PD cluster during PD upgrading.
// If not,then return an error
func checkFormerPDPodStatus(kubeCli kubernetes.Interface, pdClient pdapi.PDClient, tc *v1alpha1.TidbCluster, namespace string, ordinal int32, replicas int32) error {
	healthInfo, err := pdClient.GetHealth()
	if err != nil {
		return err
	}
	membersHealthMap := map[string]bool{}
	for _, memberHealth := range healthInfo.Healths {
		membersHealthMap[memberHealth.Name] = memberHealth.Health
	}

	tcName := tc.Name
	for i := replicas - 1; i > ordinal; i-- {
		podName := pdutil.PdPodName(tcName, i)
		pod, err := kubeCli.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return fmt.Errorf("tidbcluster: [%s/%s]'s pd pod: [%s] has no label: %s", namespace, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		healthy, existed := membersHealthMap[podName]
		if revision != tc.Status.PD.StatefulSet.UpdateRevision || !existed || !healthy {
			return fmt.Errorf("tidbcluster: [%s/%s]'s pd upgraded pod: [%s] is not ready", namespace, tcName, podName)
		}
	}
	return nil
}

// check whether this pod have PD DeferDeleting Annotations
func IsPodWithPDDeferDeletingAnnotations(pod *core.Pod) bool {
	_, existed := pod.Annotations[label.AnnPDDeferDeleting]
	return existed
}

func addDeferDeletingToPDPod(podAC *PodAdmissionControl, pod *core.Pod) error {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	now := time.Now().Format(time.RFC3339)
	pod.Annotations[label.AnnPDDeferDeleting] = now
	_, err := podAC.kubeCli.CoreV1().Pods(pod.Namespace).Update(pod)
	return err
}

func isPDLeader(pdClient pdapi.PDClient, pod *core.Pod) (bool, error) {
	leader, err := pdClient.GetPDLeader()
	if err != nil {
		return false, err
	}
	return leader.Name == pod.Name, nil
}
