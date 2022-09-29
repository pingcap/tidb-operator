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
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/toml"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/member/startscript"
	"github.com/pingcap/tidb-operator/pkg/util"

	"github.com/Masterminds/semver"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
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

var (
	// The first version that moves the rocksdb info and raft info log to store and rotate as the TiKV log is v5.0.0
	// https://github.com/tikv/tikv/pull/7358
	tikvLessThanV500, _ = semver.NewConstraint("<v5.0.0-0")
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

// PdName should match the start arg `--name` of pd-server
// See the start script of PD in pkg/manager/member/startscript/v1.pdStartScriptTpl
// and pkg/manager/member/startscript/v2.RenderPDStartScript
func PdName(tcName string, ordinal int32, namespace string, clusterDomain string, acrossK8s bool) string {
	if len(clusterDomain) > 0 {
		return fmt.Sprintf("%s.%s-pd-peer.%s.svc.%s", PdPodName(tcName, ordinal), tcName, namespace, clusterDomain)
	}

	// clusterDomain is not set
	if acrossK8s {
		return fmt.Sprintf("%s.%s-pd-peer.%s.svc", PdPodName(tcName, ordinal), tcName, namespace)
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

// MarshalTOML is a template function that try to marshal a go value to toml
func MarshalTOML(v interface{}) ([]byte, error) {
	return toml.Marshal(v)
}

func UnmarshalTOML(b []byte, obj interface{}) error {
	return toml.Unmarshal(b, obj)
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

// MapInitContainers index init containers of Pod by container name in favor of looking up
func MapInitContainers(podSpec *corev1.PodSpec) map[string]corev1.Container {
	m := map[string]corev1.Container{}
	for _, c := range podSpec.InitContainers {
		m[c.Name] = c
	}
	return m
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

func getTikVConfigMapForTiKVSpec(tikvSpec *v1alpha1.TiKVSpec, tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	config := tikvSpec.Config.DeepCopy()
	if tc.IsTLSClusterEnabled() {
		config.Set("security.ca-path", path.Join(tikvClusterCertPath, tlsSecretRootCAKey))
		config.Set("security.cert-path", path.Join(tikvClusterCertPath, corev1.TLSCertKey))
		config.Set("security.key-path", path.Join(tikvClusterCertPath, corev1.TLSPrivateKeyKey))
	}
	confText, err := config.MarshalTOML()
	if err != nil {
		return nil, err
	}

	startScript, err := startscript.RenderTiKVStartScript(tc)
	if err != nil {
		return nil, fmt.Errorf("render start-script for tc %s/%s failed: %v", tc.Namespace, tc.Name, err)
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
func addDeferDeletingAnnoToPVC(tc *v1alpha1.TidbCluster, pvc *corev1.PersistentVolumeClaim, pvcControl controller.PVCControlInterface, scaleInTime ...string) error {
	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}
	now := time.Now().Format(time.RFC3339)
	pvc.Annotations[label.AnnPVCDeferDeleting] = now
	// scaleInTime indicates the time when call scale in, for test only since pvc defer deleting time may be different in same scale in call.
	if len(scaleInTime) > 0 {
		pvc.Annotations[label.AnnPVCScaleInTime] = scaleInTime[0]
	}
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

// TiKVLessThanV50 checks whether the `image` is less than v5.0.0
func TiKVLessThanV50(image string) bool {
	_, version := parseImage(image)
	v, err := semver.NewVersion(version)
	if err != nil {
		klog.Errorf("Parse version %s failure, error: %v", version, err)
		return false
	}
	if tikvLessThanV500.Check(v) {
		return true
	}
	return false
}

// parseImage returns the image name and the tag from the input image string
func parseImage(image string) (string, string) {
	var name, tag string
	colonIdx := strings.LastIndexByte(image, ':')
	if colonIdx >= 0 {
		name = image[:colonIdx]
		tag = image[colonIdx+1:]
	} else {
		name = image
	}
	return name, tag
}

var ErrNotFoundStoreID = fmt.Errorf("not found")

func TiKVStoreIDFromStatus(tc *v1alpha1.TidbCluster, podName string) (uint64, error) {
	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == podName {
			storeID, err := strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				return 0, err
			}

			return storeID, nil
		}
	}
	return 0, ErrNotFoundStoreID
}

// MergePatchContainers adds patches to base using a strategic merge patch and
// iterating by container name, failing on the first error
func MergePatchContainers(base, patches []corev1.Container) ([]corev1.Container, error) {
	var out []corev1.Container

	containersByName := make(map[string]corev1.Container)
	for _, c := range patches {
		containersByName[c.Name] = c
	}

	// Patch the containers that exist in base.
	for _, container := range base {
		patchContainer, ok := containersByName[container.Name]
		if !ok {
			// This container didn't need to be patched.
			out = append(out, container)
			continue
		}

		containerBytes, err := json.Marshal(container)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON for container %s, error: %v", container.Name, err)
		}

		patchBytes, err := json.Marshal(patchContainer)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON for patch container %s, error: %v", container.Name, err)
		}

		// Calculate the patch result.
		jsonResult, err := strategicpatch.StrategicMergePatch(containerBytes, patchBytes, corev1.Container{})
		if err != nil {
			return nil, fmt.Errorf("failed to generate merge patch for container %s, error: %v", container.Name, err)
		}

		var patchResult corev1.Container
		if err := json.Unmarshal(jsonResult, &patchResult); err != nil {
			return nil, fmt.Errorf("failed to unmarshal merged container %s, error: %v", container.Name, err)
		}

		// Add the patch result and remove the corresponding key from the to do list.
		out = append(out, patchResult)
		delete(containersByName, container.Name)
	}

	// Append containers that are left in containersByName.
	// Iterate over patches to preserve the order.
	for _, c := range patches {
		if container, found := containersByName[c.Name]; found {
			out = append(out, container)
		}
	}

	return out, nil
}
