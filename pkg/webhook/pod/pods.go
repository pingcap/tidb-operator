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
	"encoding/json"
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	apps "k8s.io/api/apps/v1"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	memberUtils "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

type PodAdmissionControl struct {
	// kubernetes client interface
	kubeCli kubernetes.Interface
	// operator client interface
	operatorCli versioned.Interface
	// pd Control
	pdControl pdapi.PDControlInterface
	// the map of the service account from the request which should be checked by webhook
	serviceAccounts sets.String
}

const (
	stsControllerServiceAccounts = "system:serviceaccount:kube-system:statefulset-controller"
)

func NewPodAdmissionControl(kubeCli kubernetes.Interface, operatorCli versioned.Interface, PdControl pdapi.PDControlInterface, extraServiceAccounts []string, evictRegionLeaderTimeout time.Duration) *PodAdmissionControl {

	serviceAccounts := sets.NewString(stsControllerServiceAccounts)
	for _, sa := range extraServiceAccounts {
		serviceAccounts.Insert(sa)
	}
	EvictLeaderTimeout = evictRegionLeaderTimeout
	return &PodAdmissionControl{
		kubeCli:         kubeCli,
		operatorCli:     operatorCli,
		pdControl:       PdControl,
		serviceAccounts: serviceAccounts,
	}
}

// admitPayload used to simply the param to make each function more easier
type admitPayload struct {
	// the pod for admission request object
	pod *core.Pod
	// the ownerStatefulSet for target tidb component pod
	ownerStatefulSet *apps.StatefulSet
	// the owner tc for target tidb component pod
	tc *v1alpha1.TidbCluster
	// the pdClient for target tc
	pdClient pdapi.PDClient
}

func (pc *PodAdmissionControl) AdmitPods(ar *admission.AdmissionRequest) *admission.AdmissionResponse {

	name := ar.Name
	namespace := ar.Namespace
	operation := ar.Operation
	serviceAccount := ar.UserInfo.Username
	klog.Infof("receive %s pod[%s/%s] by sa[%s]", operation, namespace, name, serviceAccount)

	if !pc.serviceAccounts.Has(serviceAccount) {
		klog.Infof("Request was not sent by known controlled ServiceAccounts, admit to %s pod [%s/%s]", operation, namespace, name)
		return util.ARSuccess()
	}

	switch operation {
	case admission.Delete:
		return pc.admitDeletePods(name, namespace)
	case admission.Create:
		return pc.AdmitCreatePods(ar)
	default:
		klog.Infof("Admit to %s pod[%s/%s]", operation, namespace, name)
		return util.ARSuccess()
	}

}

// Webhook server receive request to delete pod
// if this pod wasn't member of tidbcluster, just let the request pass.
// Otherwise, we check tidbcluster and statefulset which own this pod whether they were existed.
// If either of them were deleted,we would let this request pass,
// otherwise we will check it decided by component type.
func (pc *PodAdmissionControl) admitDeletePods(name, namespace string) *admission.AdmissionResponse {

	klog.Infof("receive admission to %s pod[%s/%s]", "delete", namespace, name)

	// We would update pod annotations if they were deleted member by admission controller,
	// so we shall find this pod from apiServer considering getting latest pod info.
	pod, err := pc.kubeCli.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		klog.Infof("failed to find pod[%s/%s] during delete it,admit to delete", namespace, name)
		return util.ARSuccess()
	}

	l := label.Label(pod.Labels)

	if !l.IsManagedByTiDBOperator() {
		klog.Infof("pod[%s/%s] is not managed by TiDB-Operator,admit to create", namespace, name)
		return util.ARSuccess()
	}

	if !(l.IsPD() || l.IsTiKV() || l.IsTiDB()) {
		klog.Infof("pod[%s/%s] is not TiDB component,admit to delete", namespace, name)
		return util.ARSuccess()
	}

	tcName, exist := pod.Labels[label.InstanceLabelKey]
	if !exist {
		klog.Errorf("pod[%s/%s] has no label: %s", namespace, name, label.InstanceLabelKey)
		return util.ARSuccess()
	}

	tc, err := pc.operatorCli.PingcapV1alpha1().TidbClusters(namespace).Get(tcName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("tc[%s/%s] had been deleted,admit to delete pod[%s/%s]", namespace, tcName, namespace, name)
			return util.ARSuccess()
		}
		klog.Errorf("failed get tc[%s/%s],refuse to delete pod[%s/%s]", namespace, tcName, namespace, name)
		return util.ARFail(err)
	}

	ownerStatefulSet, err := getOwnerStatefulSetForTiDBComponent(pod, pc.kubeCli)
	if err != nil {
		if errors.IsNotFound(err) || err.Error() == fmt.Sprintf(failToFindTidbComponentOwnerStatefulset, namespace, name) {
			klog.Infof("owner statefulset for pod[%s/%s] is deleted,admit to delete pod", namespace, name)
			return util.ARSuccess()
		}
		klog.Infof("failed to get owner statefulset for pod[%s/%s],refuse to delete pod", namespace, name)
		return util.ARFail(fmt.Errorf("failed to get owner statefulset for pod[%s/%s],refuse to delete pod", namespace, name))
	}

	// Force Upgraded,Admit to Upgrade
	if memberUtils.NeedForceUpgrade(tc) {
		klog.Infof("tc[%s/%s] is force upgraded, admit to delete pod[%s/%s]", namespace, tcName, namespace, name)
		return util.ARSuccess()
	}

	ordinal, err := operatorUtils.GetOrdinalFromPodName(name)
	if err != nil {
		return util.ARFail(err)
	}

	// If there was only one replica for this statefulset,admit to delete it.
	if *ownerStatefulSet.Spec.Replicas == 1 && ordinal == 0 {
		klog.Infof("tc[%s/%s]'s pd only have one pod[%s/%s],admit to delete it.", namespace, tcName, namespace, name)
		return util.ARSuccess()
	}

	payload := &admitPayload{
		pod:              pod,
		tc:               tc,
		ownerStatefulSet: ownerStatefulSet,
		pdClient:         pc.pdControl.GetPDClient(pdapi.Namespace(namespace), tcName, tc.IsTLSClusterEnabled()),
	}

	if l.IsPD() {
		return pc.admitDeletePdPods(payload)
	} else if l.IsTiKV() {
		return pc.admitDeleteTiKVPods(payload)
	}

	klog.Infof("[%s/%s] is admit to be deleted", namespace, name)
	return util.ARSuccess()
}

// Webhook server receive request to create pod
// if this pod wasn't member of tidbcluster, just let the request pass.
// Currently we only check with tikv pod
func (pc *PodAdmissionControl) AdmitCreatePods(ar *admission.AdmissionRequest) *admission.AdmissionResponse {
	pod := &core.Pod{}
	if err := json.Unmarshal(ar.Object.Raw, pod); err != nil {
		klog.Errorf("Could not unmarshal raw object: %v", err)
		return util.ARFail(err)
	}

	name := pod.Name
	namespace := pod.Namespace

	klog.Infof("receive admission to %s pod[%s/%s]", "create", namespace, name)

	l := label.Label(pod.Labels)

	if !l.IsManagedByTiDBOperator() {
		klog.Infof("pod[%s/%s] is not managed by TiDB-Operator,admit to create", namespace, name)
		return util.ARSuccess()
	}

	if !(l.IsPD() || l.IsTiKV() || l.IsTiDB()) {
		klog.Infof("pod[%s/%s] is not TiDB component,admit to create", namespace, name)
		return util.ARSuccess()
	}

	tcName, exist := pod.Labels[label.InstanceLabelKey]
	if !exist {
		return util.ARSuccess()
	}

	tc, err := pc.operatorCli.PingcapV1alpha1().TidbClusters(namespace).Get(tcName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return util.ARSuccess()
		}
		klog.Errorf("failed get tc[%s/%s],refuse to create pod[%s/%s],%v", namespace, tcName, namespace, name, err)
		return util.ARFail(err)
	}

	if memberUtils.NeedForceUpgrade(tc) {
		klog.Infof("tc[%s/%s] is force upgraded, admit to create pod[%s/%s]", namespace, tcName, namespace, name)
		return util.ARSuccess()
	}

	if l.IsTiKV() {
		pdClient := pc.pdControl.GetPDClient(pdapi.Namespace(namespace), tcName, tc.IsTLSClusterEnabled())
		return pc.admitCreateTiKVPod(pod, tc, pdClient)
	}

	return util.ARSuccess()
}
