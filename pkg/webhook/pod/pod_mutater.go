// Copyright 2020 PingCAP, Inc.
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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

func (pc *PodAdmissionControl) mutatePod(ar *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	pod := &corev1.Pod{}
	if err := json.Unmarshal(ar.Object.Raw, pod); err != nil {
		klog.Errorf("admission validating failed: cannot unmarshal %s to %T", ar.Kind, pod)
		return util.ARFail(err)
	}
	original := pod.DeepCopy()

	l := label.Label(pod.Labels)
	if !l.IsManagedByTiDBOperator() {
		return util.ARSuccess()
	}
	if !l.IsTiKV() {
		return util.ARSuccess()
	}
	klog.Infof("receive mutation core mutate tikv pod")

	tcName, exist := pod.Labels[label.InstanceLabelKey]
	if !exist {
		return util.ARSuccess()
	}
	namespace := ar.Namespace

	tc, err := pc.operatorCli.PingcapV1alpha1().TidbClusters(namespace).Get(tcName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return util.ARSuccess()
		}
		return util.ARFail(err)
	}
	klog.Infof("annotations =%v", tc.Annotations)
	podName := pod.Name
	ordinal, err := operatorUtils.GetOrdinalFromPodName(podName)
	if err != nil {
		return util.ARFail(err)
	}
	sets := operatorUtils.GetAutoScalingOutSlots(tc, v1alpha1.TiKVMemberType)
	klog.Infof("sets = %v", sets.List())

	if !sets.Has(ordinal) {
		klog.Infof("receive mutation core mutate tikv normal pod")
		return util.ARSuccess()
	}

	klog.Infof("receive mutation core mutate tikv auto-scaling pod")
	cmName := fmt.Sprintf("%s-autoscaling", controller.TiKVMemberName(tcName))
	klog.Infof("origin pod %v", original)
	for _, v := range pod.Spec.Volumes {
		if v.Name == "config" && v.ConfigMap != nil {
			klog.Info("here! find it !!!!!!!!!!!!1")
			v.ConfigMap.LocalObjectReference = corev1.LocalObjectReference{
				Name: cmName,
			}
			break
		}
	}
	klog.Infof("update pod %v", pod)
	patch, err := util.CreateJsonPatch(original, pod)
	if err != nil {
		return util.ARFail(err)
	}
	return util.ARPatch(patch)
}
