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

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	memberUtils "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

type PodAdmissionControl struct {
	// kubernetes client interface
	kubeCli kubernetes.Interface
	// operator client interface
	operatorCli versioned.Interface
	// pvc control
	pvcControl controller.PVCControlInterface
	// pd Control
	pdControl pdapi.PDControlInterface
	// pod Lister
	podLister corelisters.PodLister
}

func NewPodAdmissionControl(kubeCli kubernetes.Interface, operatorCli versioned.Interface, PdControl pdapi.PDControlInterface, kubeInformerFactory kubeinformers.SharedInformerFactory, recorder record.EventRecorder) *PodAdmissionControl {

	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	PVCControl := controller.NewRealPVCControl(kubeCli, recorder, pvcInformer.Lister())

	podInformer := kubeInformerFactory.Core().V1().Pods()

	return &PodAdmissionControl{
		kubeCli:     kubeCli,
		operatorCli: operatorCli,
		pvcControl:  PVCControl,
		pdControl:   PdControl,
		podLister:   podInformer.Lister(),
	}
}

func (pc *PodAdmissionControl) AdmitPods(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

	name := ar.Request.Name
	namespace := ar.Request.Namespace
	operation := ar.Request.Operation
	klog.Infof("receive admission to %s pod[%s/%s]", operation, namespace, name)

	switch operation {
	case v1beta1.Delete:
		return pc.admitDeletePods(name, namespace)
	default:
		klog.Infof("Admit to %s pod[%s/%s]", operation, namespace, name)
		return util.ARSuccess()
	}
}

// Webhook server receive request to delete pod
// if this pod wasn't member of tidbcluster, just let the request pass.
// Otherwise, we check tidbcluster and statefulset which own this pod whether they were existed.
// If either of them were deleted,we would let this request pass,
//// otherwise we will check it decided by component type.
func (pc *PodAdmissionControl) admitDeletePods(name, namespace string) *v1beta1.AdmissionResponse {

	pod, err := pc.podLister.Pods(namespace).Get(name)
	if err != nil {
		klog.Infof("failed to find pod[%s/%s] during delete it,admit to delete", namespace, name)
		return util.ARSuccess()
	}

	l := label.Label(pod.Labels)
	if !(l.IsPD() || l.IsTiKV() || l.IsTiDB()) {
		klog.Infof("pod[%s/%s] is not TiDB component,admit to delete", namespace, name)
		return util.ARSuccess()
	}

	tcName, exist := pod.Labels[label.InstanceLabelKey]
	if !exist {
		klog.Errorf("pod[%s/%s] has no label: %s", namespace, name, label.InstanceLabelKey)
		return util.ARFail(fmt.Errorf("pod[%s/%s] has no label: %s", namespace, name, label.InstanceLabelKey))
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

	if len(pod.OwnerReferences) == 0 {
		return util.ARSuccess()
	}

	var ownerStatefulSetName string
	for _, ownerReference := range pod.OwnerReferences {
		if ownerReference.Kind == "StatefulSet" {
			ownerStatefulSetName = ownerReference.Name
			break
		}
	}

	if len(ownerStatefulSetName) == 0 {
		klog.Infof("pod[%s/%s] is not owned by StatefulSet,admit to delete it", namespace, name)
		return util.ARSuccess()
	}

	ownerStatefulSet, err := pc.kubeCli.AppsV1().StatefulSets(namespace).Get(ownerStatefulSetName, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("statefulset[%s/%s] had been deleted,admit to delete pod[%s/%s]", namespace, ownerStatefulSetName, namespace, name)
			return util.ARSuccess()
		}
		klog.Errorf("failed to get statefulset[%s/%s],refuse to delete pod[%s/%s]", namespace, ownerStatefulSetName, namespace, name)
		return util.ARFail(fmt.Errorf("failed to get statefulset[%s/%s],refuse to delete pod[%s/%s]", namespace, ownerStatefulSetName, namespace, name))
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

	if l.IsPD() {
		pdClient := pc.pdControl.GetPDClient(pdapi.Namespace(namespace), tcName, tc.Spec.EnableTLSCluster)
		return pc.admitDeletePdPods(pod, ownerStatefulSet, tc, pdClient)
	}

	klog.Infof("[%s/%s] is admit to be deleted", namespace, name)
	return util.ARSuccess()
}
