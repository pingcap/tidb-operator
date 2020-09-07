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

	"github.com/BurntSushi/toml"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// mutatePod mutates the pod by setting hotRegion label if the pod is created by AutoScaling
func (pc *PodAdmissionControl) mutatePod(ar *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	if !features.DefaultFeatureGate.Enabled(features.AutoScaling) {
		return util.ARSuccess()
	}

	pod := &corev1.Pod{}
	if err := json.Unmarshal(ar.Object.Raw, pod); err != nil {
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

	if err := pc.tikvHotRegionSchedule(tc, pod); err != nil {
		return util.ARFail(err)
	}

	patch, err := util.CreateJsonPatch(original, pod)
	if err != nil {
		return util.ARFail(err)
	}
	return util.ARPatch(patch)
}

func (pc *PodAdmissionControl) tikvHotRegionSchedule(tc *v1alpha1.TidbCluster, pod *corev1.Pod) error {
	podName := pod.Name
	ordinal, err := operatorUtils.GetOrdinalFromPodName(podName)
	if err != nil {
		return err
	}
	sets := operatorUtils.GetAutoScalingOutSlots(tc, v1alpha1.TiKVMemberType)
	if !sets.Has(ordinal) {
		return nil
	}

	cm, err := pc.getTikvConfigMap(tc, pod)
	if err != nil {
		klog.Infof("tc[%s/%s]'s tikv %s configmap not found, error: %v", tc.Namespace, tc.Name, pod.Name, err)
		return err
	}
	v, ok := cm.Data["config-file"]
	if !ok {
		return fmt.Errorf("tc[%s/%s]'s tikv config[config-file] is missing", tc.Namespace, tc.Name)
	}
	config := &v1alpha1.TiKVConfig{}
	err = toml.Unmarshal([]byte(v), config)
	if err != nil {
		return err
	}
	if config.Server == nil {
		config.Server = &v1alpha1.TiKVServerConfig{}
	}
	if config.Server.Labels == nil {
		config.Server.Labels = map[string]string{}
	}
	// TODO: add document to explain the hot region label
	config.Server.Labels["specialUse"] = "hotRegion"
	for id, c := range pod.Spec.Containers {
		if c.Name == "tikv" {
			appendExtraLabelsENVForTiKV(config.Server.Labels, &c)
			pod.Spec.Containers[id] = c
			break
		}
	}
	return nil
}

// Get tikv original configmap from the pod spec template volume
func (pc *PodAdmissionControl) getTikvConfigMap(tc *v1alpha1.TidbCluster, pod *corev1.Pod) (*corev1.ConfigMap, error) {
	cnName := ""
	for _, v := range pod.Spec.Volumes {
		if (v.Name == "config" || v.Name == "startup-script") && v.ConfigMap != nil {
			cnName = v.ConfigMap.Name
			break
		}
	}
	if cnName == "" {
		return nil, fmt.Errorf("tc[%s/%s] 's tikv configmap can't find", tc.Namespace, tc.Name)
	}
	return pc.kubeCli.CoreV1().ConfigMaps(tc.Namespace).Get(cnName, metav1.GetOptions{})
}
