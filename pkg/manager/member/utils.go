// Copyright 2018 PingCAP, Inc.
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

package member

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
<<<<<<< HEAD
	"k8s.io/klog"
=======
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/utils/pointer"
>>>>>>> 11a4647... support recover from failover for tiflash and tikv (#3189)
)

const (
	// LastAppliedConfigAnnotation is annotation key of last applied configuration
	LastAppliedConfigAnnotation = "pingcap.com/last-applied-configuration"
	// ImagePullBackOff is the pod state of image pull failed
	ImagePullBackOff = "ImagePullBackOff"
	// ErrImagePull is the pod state of image pull failed
	ErrImagePull = "ErrImagePull"
)

func annotationsMountVolume() (corev1.VolumeMount, corev1.Volume) {
	m := corev1.VolumeMount{Name: "annotations", ReadOnly: true, MountPath: "/etc/podinfo"}
	v := corev1.Volume{
		Name: "annotations",
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path:     "annotations",
						FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations"},
					},
				},
			},
		},
	}
	return m, v
}

// statefulSetIsUpgrading confirms whether the statefulSet is upgrading phase
func statefulSetIsUpgrading(set *apps.StatefulSet) bool {
	if set.Status.CurrentRevision != set.Status.UpdateRevision {
		return true
	}
	if set.Generation > set.Status.ObservedGeneration && *set.Spec.Replicas == set.Status.Replicas {
		return true
	}
	return false
}

// SetStatefulSetLastAppliedConfigAnnotation set last applied config to Statefulset's annotation
func SetStatefulSetLastAppliedConfigAnnotation(set *apps.StatefulSet) error {
	setApply, err := util.Encode(set.Spec)
	if err != nil {
		return err
	}
	if set.Annotations == nil {
		set.Annotations = map[string]string{}
	}
	set.Annotations[LastAppliedConfigAnnotation] = setApply
	return nil
}

// GetLastAppliedConfig get last applied config info from Statefulset's annotation and the podTemplate's annotation
func GetLastAppliedConfig(set *apps.StatefulSet) (*apps.StatefulSetSpec, *corev1.PodSpec, error) {
	specAppliedConfig, ok := set.Annotations[LastAppliedConfigAnnotation]
	if !ok {
		return nil, nil, fmt.Errorf("statefulset:[%s/%s] not found spec's apply config", set.GetNamespace(), set.GetName())
	}
	spec := &apps.StatefulSetSpec{}
	err := json.Unmarshal([]byte(specAppliedConfig), spec)
	if err != nil {
		return nil, nil, err
	}

	return spec, &spec.Template.Spec, nil
}

// statefulSetEqual compares the new Statefulset's spec with old Statefulset's last applied config
func statefulSetEqual(new apps.StatefulSet, old apps.StatefulSet) bool {
	// The annotations in old sts may include LastAppliedConfigAnnotation
	tmpAnno := map[string]string{}
	for k, v := range old.Annotations {
		if k != LastAppliedConfigAnnotation {
			tmpAnno[k] = v
		}
	}
	if !apiequality.Semantic.DeepEqual(new.Annotations, tmpAnno) {
		return false
	}
	oldConfig := apps.StatefulSetSpec{}
	if lastAppliedConfig, ok := old.Annotations[LastAppliedConfigAnnotation]; ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldConfig)
		if err != nil {
			klog.Errorf("unmarshal Statefulset: [%s/%s]'s applied config failed,error: %v", old.GetNamespace(), old.GetName(), err)
			return false
		}
		// oldConfig.Template.Annotations may include LastAppliedConfigAnnotation to keep backward compatiability
		// Please check detail in https://github.com/pingcap/tidb-operator/pull/1489
		tmpTemplate := oldConfig.Template.DeepCopy()
		delete(tmpTemplate.Annotations, LastAppliedConfigAnnotation)
		return apiequality.Semantic.DeepEqual(oldConfig.Replicas, new.Spec.Replicas) &&
			apiequality.Semantic.DeepEqual(*tmpTemplate, new.Spec.Template) &&
			apiequality.Semantic.DeepEqual(oldConfig.UpdateStrategy, new.Spec.UpdateStrategy)
	}
	return false
}

// templateEqual compares the new podTemplateSpec's spec with old podTemplateSpec's last applied config
func templateEqual(new *apps.StatefulSet, old *apps.StatefulSet) bool {
	oldStsSpec := apps.StatefulSetSpec{}
	lastAppliedConfig, ok := old.Annotations[LastAppliedConfigAnnotation]
	if ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldStsSpec)
		if err != nil {
			klog.Errorf("unmarshal PodTemplate: [%s/%s]'s applied config failed,error: %v", old.GetNamespace(), old.GetName(), err)
			return false
		}
		return apiequality.Semantic.DeepEqual(oldStsSpec.Template.Spec, new.Spec.Template.Spec)
	}
	return false
}

// setUpgradePartition set statefulSet's rolling update partition
func setUpgradePartition(set *apps.StatefulSet, upgradeOrdinal int32) {
	set.Spec.UpdateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{Partition: &upgradeOrdinal}
	klog.Infof("set %s/%s partition to %d", set.GetNamespace(), set.GetName(), upgradeOrdinal)
}

func MemberPodName(tcName string, ordinal int32, memberType v1alpha1.MemberType) string {
	return fmt.Sprintf("%s-%s-%d", tcName, memberType.String(), ordinal)
}

func TikvPodName(tcName string, ordinal int32) string {
	return fmt.Sprintf("%s-%d", controller.TiKVMemberName(tcName), ordinal)
}

func PdPodName(tcName string, ordinal int32) string {
	return fmt.Sprintf("%s-%d", controller.PDMemberName(tcName), ordinal)
}

func tidbPodName(tcName string, ordinal int32) string {
	return fmt.Sprintf("%s-%d", controller.TiDBMemberName(tcName), ordinal)
}

// CombineAnnotations merges two annotations maps
func CombineAnnotations(a, b map[string]string) map[string]string {
	if a == nil {
		a = make(map[string]string)
	}
	for k, v := range b {
		a[k] = v
	}
	return a
}

// NeedForceUpgrade check if force upgrade is necessary
func NeedForceUpgrade(tc *v1alpha1.TidbCluster) bool {
	// Check if annotation 'pingcap.com/force-upgrade: "true"' is set
	if tc.Annotations != nil {
		forceVal, ok := tc.Annotations[label.AnnForceUpgradeKey]
		if ok && (forceVal == label.AnnForceUpgradeVal) {
			return true
		}
	}
	return false
}

// FindConfigMapVolume returns the configmap which's name matches the predicate in a PodSpec, empty indicates not found
func FindConfigMapVolume(podSpec *corev1.PodSpec, pred func(string) bool) string {
	for _, vol := range podSpec.Volumes {
		if vol.ConfigMap != nil && pred(vol.ConfigMap.LocalObjectReference.Name) {
			return vol.ConfigMap.LocalObjectReference.Name
		}
	}
	return ""
}

// MarshalTOML is a template function that try to marshal a go value to toml
func MarshalTOML(v interface{}) ([]byte, error) {
	buff := new(bytes.Buffer)
	encoder := toml.NewEncoder(buff)
	err := encoder.Encode(v)
	if err != nil {
		return nil, err
	}
	data := buff.Bytes()
	return data, nil
}

func UnmarshalTOML(b []byte, obj interface{}) error {
	return toml.Unmarshal(b, obj)
}

func Sha256Sum(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}

func AddConfigMapDigestSuffix(cm *corev1.ConfigMap) error {
	sum, err := Sha256Sum(cm.Data)
	if err != nil {
		return err
	}
	suffix := fmt.Sprintf("%x", sum)[0:7]
	cm.Name = fmt.Sprintf("%s-%s", cm.Name, suffix)
	return nil
}

// getStsAnnotations gets annotations for statefulset of given component.
func getStsAnnotations(tc *v1alpha1.TidbCluster, component string) map[string]string {
	anns := map[string]string{}
	tcAnns := tc.Annotations
	if tcAnns == nil {
		return anns
	}
	// delete slots
	var key string
	if component == label.PDLabelVal {
		key = label.AnnPDDeleteSlots
	} else if component == label.TiDBLabelVal {
		key = label.AnnTiDBDeleteSlots
	} else if component == label.TiKVLabelVal {
		key = label.AnnTiKVDeleteSlots
	} else if component == label.TiFlashLabelVal {
		key = label.AnnTiFlashDeleteSlots
	} else {
		return anns
	}
	if val, ok := tcAnns[key]; ok {
		anns[helper.DeleteSlotsAnn] = val
	}
	return anns
}

// MapContainers index containers of Pod by container name in favor of looking up
func MapContainers(podSpec *corev1.PodSpec) map[string]corev1.Container {
	m := map[string]corev1.Container{}
	for _, c := range podSpec.Containers {
		m[c.Name] = c
	}
	return m
}

// updateStatefulSet is a template function to update the statefulset of components
func updateStatefulSet(setCtl controller.StatefulSetControlInterface, tc *v1alpha1.TidbCluster, newSet, oldSet *apps.StatefulSet) error {
	isOrphan := metav1.GetControllerOf(oldSet) == nil
	if newSet.Annotations == nil {
		newSet.Annotations = map[string]string{}
	}
	if oldSet.Annotations == nil {
		oldSet.Annotations = map[string]string{}
	}
	if !statefulSetEqual(*newSet, *oldSet) || isOrphan {
		set := *oldSet
		// Retain the deprecated last applied pod template annotation for backward compatibility
		var podConfig string
		var hasPodConfig bool
		if oldSet.Spec.Template.Annotations != nil {
			podConfig, hasPodConfig = oldSet.Spec.Template.Annotations[LastAppliedConfigAnnotation]
		}
		set.Spec.Template = newSet.Spec.Template
		if hasPodConfig {
			set.Spec.Template.Annotations[LastAppliedConfigAnnotation] = podConfig
		}
		set.Annotations = newSet.Annotations
		v, ok := oldSet.Annotations[label.AnnStsLastSyncTimestamp]
		if ok {
			set.Annotations[label.AnnStsLastSyncTimestamp] = v
		}
		*set.Spec.Replicas = *newSet.Spec.Replicas
		set.Spec.UpdateStrategy = newSet.Spec.UpdateStrategy
		if isOrphan {
			set.OwnerReferences = newSet.OwnerReferences
			set.Labels = newSet.Labels
		}
		err := SetStatefulSetLastAppliedConfigAnnotation(&set)
		if err != nil {
			return err
		}
		_, err = setCtl.UpdateStatefulSet(tc, &set)
		return err
	}

	return nil
}

// filter targetContainer by  containerName, If not find, then return nil
func filterContainer(sts *apps.StatefulSet, containerName string) *corev1.Container {
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == containerName {
			return &c
		}
	}
	return nil
}

func copyAnnotations(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := map[string]string{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
<<<<<<< HEAD
=======

func getTikVConfigMapForTiKVSpec(tikvSpec *v1alpha1.TiKVSpec, tc *v1alpha1.TidbCluster, scriptModel *TiKVStartScriptModel) (*corev1.ConfigMap, error) {
	config := tikvSpec.Config
	if tc.IsTLSClusterEnabled() {
		if config.Security == nil {
			config.Security = &v1alpha1.TiKVSecurityConfig{}
		}
		config.Security.CAPath = pointer.StringPtr(path.Join(tikvClusterCertPath, tlsSecretRootCAKey))
		config.Security.CertPath = pointer.StringPtr(path.Join(tikvClusterCertPath, corev1.TLSCertKey))
		config.Security.KeyPath = pointer.StringPtr(path.Join(tikvClusterCertPath, corev1.TLSPrivateKeyKey))
	}
	confText, err := MarshalTOML(config)
	if err != nil {
		return nil, err
	}
	startScript, err := RenderTiKVStartScript(scriptModel)
	if err != nil {
		return nil, err
	}
	cm := &corev1.ConfigMap{
		Data: map[string]string{
			"config-file":    transformTiKVConfigMap(string(confText), tc),
			"startup-script": startScript,
		},
	}
	return cm, nil
}

// shouldRecover checks whether we should perform recovery operation.
func shouldRecover(tc *v1alpha1.TidbCluster, component string, podLister corelisters.PodLister) bool {
	var stores map[string]v1alpha1.TiKVStore
	var failureStores map[string]v1alpha1.TiKVFailureStore
	var ordinals sets.Int32
	var podPrefix string

	switch component {
	case label.TiKVLabelVal:
		stores = tc.Status.TiKV.Stores
		failureStores = tc.Status.TiKV.FailureStores
		ordinals = tc.TiKVStsDesiredOrdinals(true)
		podPrefix = controller.TiKVMemberName(tc.Name)
	case label.TiFlashLabelVal:
		stores = tc.Status.TiFlash.Stores
		failureStores = tc.Status.TiFlash.FailureStores
		ordinals = tc.TiFlashStsDesiredOrdinals(true)
		podPrefix = controller.TiFlashMemberName(tc.Name)
	default:
		klog.Warningf("Unexpected component %s for %s/%s in shouldRecover", component, tc.Namespace, tc.Name)
		return false
	}
	if failureStores == nil {
		return false
	}
	// If all desired replicas (excluding failover pods) of tidb cluster are
	// healthy, we can perform our failover recovery operation.
	// Note that failover pods may fail (e.g. lack of resources) and we don't care
	// about them because we're going to delete them.
	for ordinal := range ordinals {
		name := fmt.Sprintf("%s-%d", podPrefix, ordinal)
		pod, err := podLister.Pods(tc.Namespace).Get(name)
		if err != nil {
			klog.Errorf("pod %s/%s does not exist: %v", tc.Namespace, name, err)
			return false
		}
		if !podutil.IsPodReady(pod) {
			return false
		}
		var exist bool
		for _, v := range stores {
			if v.PodName == pod.Name {
				exist = true
				if v.State != v1alpha1.TiKVStateUp {
					return false
				}
			}
		}
		if !exist {
			return false
		}
	}
	return true
}
>>>>>>> 11a4647... support recover from failover for tiflash and tikv (#3189)
