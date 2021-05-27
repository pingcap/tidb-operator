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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/pkg/util/toml"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
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

func MemberPodName(controllerName, controllerKind string, ordinal int32, memberType v1alpha1.MemberType) (string, error) {
	switch controllerKind {
	case v1alpha1.TiDBClusterKind:
		return fmt.Sprintf("%s-%s-%d", controllerName, memberType.String(), ordinal), nil
	default:
		return "", fmt.Errorf("unknown controller kind[%s]", controllerKind)
	}
}

func TiFlashPodName(tcName string, ordinal int32) string {
	return fmt.Sprintf("%s-%d", controller.TiFlashMemberName(tcName), ordinal)
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

func ticdcPodName(tcName string, ordinal int32) string {
	return fmt.Sprintf("%s-%d", controller.TiCDCMemberName(tcName), ordinal)
}

func DMMasterPodName(dcName string, ordinal int32) string {
	return fmt.Sprintf("%s-%d", controller.DMMasterMemberName(dcName), ordinal)
}

func PdName(tcName string, ordinal int32, namespace string, clusterDomain string) string {
	if len(clusterDomain) > 0 {
		return fmt.Sprintf("%s.%s-pd-peer.%s.svc.%s", PdPodName(tcName, ordinal), tcName, namespace, clusterDomain)
	}
	return PdPodName(tcName, ordinal)
}

// NeedForceUpgrade check if force upgrade is necessary
func NeedForceUpgrade(ann map[string]string) bool {
	// Check if annotation 'pingcap.com/force-upgrade: "true"' is set
	if ann != nil {
		forceVal, ok := ann[label.AnnForceUpgradeKey]
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
	return toml.Marshal(v)
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
func getStsAnnotations(tcAnns map[string]string, component string) map[string]string {
	anns := map[string]string{}
	if tcAnns == nil {
		return anns
	}

	// ensure the delete-slots annotation
	var key string
	switch component {
	case label.PDLabelVal:
		key = label.AnnPDDeleteSlots
	case label.TiDBLabelVal:
		key = label.AnnTiDBDeleteSlots
	case label.TiKVLabelVal:
		key = label.AnnTiKVDeleteSlots
	case label.TiFlashLabelVal:
		key = label.AnnTiFlashDeleteSlots
	case label.DMMasterLabelVal:
		key = label.AnnDMMasterDeleteSlots
	case label.DMWorkerLabelVal:
		key = label.AnnDMWorkerDeleteSlots
	default:
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

// UpdateStatefulSet is a template function to update the statefulset of components
func UpdateStatefulSet(setCtl controller.StatefulSetControlInterface, object runtime.Object, newSet, oldSet *apps.StatefulSet) error {
	isOrphan := metav1.GetControllerOf(oldSet) == nil
	if newSet.Annotations == nil {
		newSet.Annotations = map[string]string{}
	}
	if oldSet.Annotations == nil {
		oldSet.Annotations = map[string]string{}
	}

	// Check if an upgrade is needed.
	// If not, early return.
	if util.StatefulSetEqual(*newSet, *oldSet) && !isOrphan {
		return nil
	}

	set := *oldSet

	// update specs for sts
	*set.Spec.Replicas = *newSet.Spec.Replicas
	set.Spec.UpdateStrategy = newSet.Spec.UpdateStrategy
	set.Labels = newSet.Labels
	set.Annotations = newSet.Annotations
	set.Spec.Template = newSet.Spec.Template
	if isOrphan {
		set.OwnerReferences = newSet.OwnerReferences
	}

	var podConfig string
	var hasPodConfig bool
	if oldSet.Spec.Template.Annotations != nil {
		podConfig, hasPodConfig = oldSet.Spec.Template.Annotations[LastAppliedConfigAnnotation]
	}
	if hasPodConfig {
		if set.Spec.Template.Annotations == nil {
			set.Spec.Template.Annotations = map[string]string{}
		}
		set.Spec.Template.Annotations[LastAppliedConfigAnnotation] = podConfig
	}
	v, ok := oldSet.Annotations[label.AnnStsLastSyncTimestamp]
	if ok {
		set.Annotations[label.AnnStsLastSyncTimestamp] = v
	}

	err := SetStatefulSetLastAppliedConfigAnnotation(&set)
	if err != nil {
		return err
	}

	// commit to k8s
	_, err = setCtl.UpdateStatefulSet(object, &set)
	return err
}

// findContainerByName finds targetContainer by containerName, If not find, then return nil
func findContainerByName(sts *apps.StatefulSet, containerName string) *corev1.Container {
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == containerName {
			return &c
		}
	}
	return nil
}

func getTikVConfigMapForTiKVSpec(tikvSpec *v1alpha1.TiKVSpec, tc *v1alpha1.TidbCluster, scriptModel *TiKVStartScriptModel) (*corev1.ConfigMap, error) {
	config := tikvSpec.Config
	if tc.IsTLSClusterEnabled() {
		config.Set("security.ca-path", path.Join(tikvClusterCertPath, tlsSecretRootCAKey))
		config.Set("security.cert-path", path.Join(tikvClusterCertPath, corev1.TLSCertKey))
		config.Set("security.key-path", path.Join(tikvClusterCertPath, corev1.TLSPrivateKeyKey))
	}
	confText, err := config.MarshalTOML()
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

// shouldRecover checks whether we should perform recovery operation.
func shouldRecoverDM(dc *v1alpha1.DMCluster, component string, podLister corelisters.PodLister) bool {
	var members map[string]v1alpha1.WorkerMember
	var failureMembers map[string]v1alpha1.WorkerFailureMember
	var ordinals sets.Int32
	var podPrefix string

	switch component {
	case label.DMWorkerLabelVal:
		members = dc.Status.Worker.Members
		failureMembers = dc.Status.Worker.FailureMembers
		ordinals = dc.WorkerStsDesiredOrdinals(true)
		podPrefix = controller.DMWorkerMemberName(dc.Name)
	default:
		klog.Warningf("Unexpected component %s for %s/%s in shouldRecover", component, dc.Namespace, dc.Name)
		return false
	}
	if failureMembers == nil {
		return false
	}
	// If all desired replicas (excluding failover pods) of dm cluster are
	// healthy, we can perform our failover recovery operation.
	// Note that failover pods may fail (e.g. lack of resources) and we don't care
	// about them because we're going to delete them.
	for ordinal := range ordinals {
		name := fmt.Sprintf("%s-%d", podPrefix, ordinal)
		pod, err := podLister.Pods(dc.Namespace).Get(name)
		if err != nil {
			klog.Errorf("pod %s/%s does not exist: %v", dc.Namespace, name, err)
			return false
		}
		if !podutil.IsPodReady(pod) {
			return false
		}
		var exist bool
		for _, v := range members {
			if v.Name == pod.Name {
				exist = true
				if v.Stage == v1alpha1.DMWorkerStateOffline {
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

func CreateOrUpdateService(serviceLister corelisters.ServiceLister, serviceControl controller.ServiceControlInterface, newSvc *corev1.Service, obj runtime.Object) error {
	oldSvcTmp, err := serviceLister.Services(newSvc.Namespace).Get(newSvc.Name)
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return serviceControl.CreateService(obj, newSvc)
	}
	if err != nil {
		return fmt.Errorf("createOrUpdateService: fail to get svc %s for obj %v, error: %s", newSvc.Name, obj, err)
	}

	oldSvc := oldSvcTmp.DeepCopy()
	util.RetainManagedFields(newSvc, oldSvc)

	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	annoEqual := util.IsSubMapOf(newSvc.Annotations, oldSvc.Annotations)
	isOrphan := metav1.GetControllerOf(oldSvc) == nil

	if !equal || !annoEqual || isOrphan {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		err = controller.SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
		// apply change of annotations if any
		for k, v := range newSvc.Annotations {
			svc.Annotations[k] = v
		}
		// also override labels when adopt orphan
		if isOrphan {
			svc.OwnerReferences = newSvc.OwnerReferences
			svc.Labels = newSvc.Labels
		}
		_, err = serviceControl.UpdateService(obj, &svc)
		return err
	}
	return nil
}

// addDeferDeletingAnnoToPVC set the label
func addDeferDeletingAnnoToPVC(tc *v1alpha1.TidbCluster, pvc *corev1.PersistentVolumeClaim, pvcControl controller.PVCControlInterface) error {
	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}
	now := time.Now().Format(time.RFC3339)
	pvc.Annotations[label.AnnPVCDeferDeleting] = now
	if _, err := pvcControl.UpdatePVC(tc, pvc); err != nil {
		klog.Errorf("failed to set PVC %s/%s annotation %q to %q", tc.Namespace, pvc.Name, label.AnnPVCDeferDeleting, now)
		return err
	}
	klog.Infof("set PVC %s/%s annotation %q to %q successfully", tc.Namespace, pvc.Name, label.AnnPVCDeferDeleting, now)
	return nil
}

// GetPVCSelectorForPod compose a PVC selector from a tc/dm-cluster member pod at ordinal position
func GetPVCSelectorForPod(controller runtime.Object, memberType v1alpha1.MemberType, ordinal int32) (labels.Selector, error) {
	meta := controller.(metav1.Object)
	var podName string
	var l label.Label
	switch controller.(type) {
	case *v1alpha1.TidbCluster:
		podName = ordinalPodName(memberType, meta.GetName(), ordinal)
		l = label.New().Instance(meta.GetName())
		l[label.AnnPodNameKey] = podName
	case *v1alpha1.DMCluster:
		// podName = ordinalPodName(memberType, meta.GetName(), ordinal)
		l = label.NewDM().Instance(meta.GetName())
		// just delete all defer Deleting pvc for convenience. Or dm have to support sync meta info labels for pod/pvc which seems unnecessary
		// l[label.AnnPodNameKey] = podName
	default:
		kind := controller.GetObjectKind().GroupVersionKind().Kind
		return nil, fmt.Errorf("object %s/%s of kind %s has unknown controller", meta.GetNamespace(), meta.GetName(), kind)
	}
	return l.Selector()
}
