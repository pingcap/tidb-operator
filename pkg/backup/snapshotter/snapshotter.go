// Copyright 2022 PingCAP, Inc.
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

package snapshotter

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/backup/util"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog/v2"
)

type Snapshotter interface {
	// Init prepares the Snapshotter for usage, it would add
	// specific parameters as config map[string]string in the future.
	Init(deps *controller.Dependencies, conf map[string]string) error

	// GetVolumeID returns the cloud provider specific identifier
	// for the PersistentVolume-PV.
	GetVolumeID(pv *corev1.PersistentVolume) (string, error)

	// PrepareBackupMetadata performs the preparations for creating
	// a snapshot from the used volume for PV/PVC.
	PrepareBackupMetadata(b *v1alpha1.Backup, tc *v1alpha1.TidbCluster) (string, error)

	// PrepareRestoreMetadata performs the preparations for creating
	// a volume from the snapshot that has been backed up.
	PrepareRestoreMetadata(r *v1alpha1.Restore) (string, error)

	// SetVolumeID sets the cloud provider specific identifier
	// for the PersistentVolume-PV.
	SetVolumeID(pv *corev1.PersistentVolume, volumeID string) error
}

type BaseSnapshotter struct {
	//nolint:structcheck // false positive
	volRegexp *regexp.Regexp
	deps      *controller.Dependencies
	config    map[string]string
}

func (s *BaseSnapshotter) Init(deps *controller.Dependencies, conf map[string]string) error {
	s.deps = deps
	s.config = conf
	return nil
}

func NewSnapshotterForBackup(m v1alpha1.BackupMode, d *controller.Dependencies) (Snapshotter, string, error) {
	var s Snapshotter
	switch m {
	case v1alpha1.BackupModeVolumeSnapshot:
		// Currently, we only support aws volume snapshot. If gcp volume snapshot is supported
		// in the future, we can infer the provider from the storage class.
		s = &AWSSnapshotter{}
	default:
		s = &NoneSnapshotter{}
	}
	err := s.Init(d, nil)
	if err != nil {
		return s, "InitSnapshotterFailed", err
	}

	return s, "", nil
}

func NewSnapshotterForRestore(m v1alpha1.RestoreMode, d *controller.Dependencies) (Snapshotter, string, error) {
	var s Snapshotter
	switch m {
	case v1alpha1.RestoreModeVolumeSnapshot:
		// Currently, we only support aws volume snapshot. If gcp volume snapshot is supported
		// in the future, we can infer the provider from the storage class.
		s = &AWSSnapshotter{}
	default:
		s = &NoneSnapshotter{}
	}
	err := s.Init(d, nil)
	if err != nil {
		return s, "InitSnapshotterFailed", err
	}

	return s, "", nil
}

func (s *BaseSnapshotter) PrepareCSBK8SMeta(csb *CloudSnapBackup, tc *v1alpha1.TidbCluster) ([]*corev1.Pod, string, error) {
	if s.deps == nil {
		return nil, "NotExistDependencies", fmt.Errorf("unexpected error for nil dependencies")
	}

	sel, err := label.New().Instance(tc.Name).TiKV().Selector()
	if err != nil {
		return nil, fmt.Sprintf("unexpected error generating pvc label selector: %v", err), err
	}

	pvcs, err := s.deps.PVCLister.PersistentVolumeClaims(tc.Namespace).List(sel)

	if err != nil {
		return nil, fmt.Sprintf("failed to fetch pvcs %s:%s", label.ComponentLabelKey, label.TiKVLabelVal), err
	}
	csb.Kubernetes.PVCs = pvcs

	pvSels, err := label.New().Instance(tc.Name).Namespace(tc.Namespace).TiKV().Selector()
	if err != nil {
		return nil, fmt.Sprintf("unexpected error generating pv label selector: %v", err), err
	}
	pvs, err := s.deps.PVLister.List(pvSels)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch pvs %s:%s", label.ComponentLabelKey, label.TiKVLabelVal), err
	}
	csb.Kubernetes.PVs = pvs
	pods, err := s.deps.PodLister.Pods(tc.Namespace).List(sel)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch pods %s:%s", label.ComponentLabelKey, label.TiKVLabelVal), err
	}
	return pods, "", nil
}

func (s *BaseSnapshotter) prepareBackupMetadata(
	b *v1alpha1.Backup, tc *v1alpha1.TidbCluster, execr Snapshotter) (string, error) {
	csb := NewCloudSnapshotBackup(tc)
	pods, reason, err := s.PrepareCSBK8SMeta(csb, tc)
	if err != nil {
		return reason, err
	}

	m := NewBackupStoresMixture(tc, csb.Kubernetes.PVCs, csb.Kubernetes.PVs, execr)
	if reason, err := m.PrepareCSBStoresMeta(csb, pods); err != nil {
		return reason, err
	}

	if reason, err := placeCloudSnapBackup(b, csb); err != nil {
		return reason, err
	}

	return "", nil
}

func placeCloudSnapBackup(b *v1alpha1.Backup, csb *CloudSnapBackup) (string, error) {
	out, err := json.Marshal(csb)
	if err != nil {
		return "ParseCloudSnapshotBackupFailed", err
	}

	if b.Annotations == nil {
		b.Annotations = make(map[string]string)
	}
	b.Annotations[label.AnnBackupCloudSnapKey] = util.BytesToString(out)
	return "", nil
}

func (s *BaseSnapshotter) prepareRestoreMetadata(r *v1alpha1.Restore, execr Snapshotter) (string, error) {
	csb, reason, err := extractCloudSnapBackup(r)
	if err != nil {
		return reason, err
	}

	if reason, err := checkCloudSnapBackup(csb); err != nil {
		return reason, err
	}

	m := NewRestoreStoresMixture(execr)
	if reason, err := m.ProcessCSBPVCsAndPVs(r, csb); err != nil {
		return reason, err
	}

	// commit new PVs and PVCs to kubernetes api-server
	if reason, err := commitPVsAndPVCsToK8S(s.deps, r, csb.Kubernetes.PVCs, csb.Kubernetes.PVs); err != nil {
		return reason, err
	}

	return "", nil
}

func extractCloudSnapBackup(r *v1alpha1.Restore) (*CloudSnapBackup, string, error) {
	str, ok := r.Annotations[label.AnnBackupCloudSnapKey]
	if !ok {
		return nil, "GetCloudSnapBackupFailed", errors.New("restore.annotation for CloudSnapBackup not found")
	}

	csb := &CloudSnapBackup{}
	err := json.Unmarshal(util.StringToBytes(str), csb)
	if err != nil {
		return nil, "ParseCloudSnapBackupFailed", err
	}
	return csb, "", nil
}

func checkCloudSnapBackup(b *CloudSnapBackup) (string, error) {
	if b == nil {
		return "GetCloudSnapBackupFailed", errors.New("restore for CloudSnapBackup not found")
	}
	if b.Kubernetes == nil {
		return "GetKubernetesFailed", errors.New(".kubernetes for CloudSnapBackup not found")
	}

	pvsLen := len(b.Kubernetes.PVs)
	pvcsLen := len(b.Kubernetes.PVCs)
	if len(b.Kubernetes.PVs) == 0 || len(b.Kubernetes.PVCs) == 0 {
		return "GetKubernetesResourceFailed", errors.New(".kubernetes.pvcs or pvs for CloudSnapBackup not found")
	}

	if pvsLen != pvcsLen {
		return "InvalidKubernetesResource", errors.New(".kubernetes.pvcs and pvs for CloudSnapBackup not matched")
	}

	if b.TiKV == nil || len(b.TiKV.Stores) == 0 {
		return "GetTiKVStoresFailed", errors.New(".tikv.stores for CloudSnapBackup not found")
	}

	return "", nil
}

func commitPVsAndPVCsToK8S(
	deps *controller.Dependencies,
	r *v1alpha1.Restore,
	pvcs []*corev1.PersistentVolumeClaim,
	pvs []*corev1.PersistentVolume) (string, error) {
	sel, err := label.New().Instance(r.Spec.BR.Cluster).TiKV().Selector()
	if err != nil {
		return "BuildTiKVSelectorFailed", err
	}
	existingPVs, err := deps.PVLister.List(sel)
	if err != nil {
		return "ListPVsFailed", err
	}
	refPVCMap := make(map[string]struct{})
	for _, pv := range existingPVs {
		if pv.Spec.ClaimRef != nil {
			refPVCMap[pv.Spec.ClaimRef.Name] = struct{}{}
		}
	}

	// Since we may generate random PV names, to avoid creating duplicate PVs,
	// we need to check if the PV already exists before creating it.
	for _, pv := range pvs {
		if pv.Spec.ClaimRef != nil {
			if _, ok := refPVCMap[pv.Spec.ClaimRef.Name]; ok {
				continue
			}
		}
		if err := deps.PVControl.CreatePV(r, pv); err != nil {
			return "CreatePVFailed", err
		}
	}

	for _, pvc := range pvcs {
		if err := deps.PVCControl.CreatePVC(r, pvc); err != nil {
			if apierrors.IsAlreadyExists(err) {
				continue
			}
			return "CreatePVCFailed", err
		}
	}

	return "", nil
}

type CloudSnapBackup struct {
	TiKV       *TiKVBackup            `json:"tikv"`
	PD         Component              `json:"pd"`
	TiDB       Component              `json:"tidb"`
	Kubernetes *KubernetesBackup      `json:"kubernetes"`
	Options    map[string]interface{} `json:"options"`
}

type KubernetesBackup struct {
	PVCs         []*corev1.PersistentVolumeClaim `json:"pvcs"`
	PVs          []*corev1.PersistentVolume      `json:"pvs"`
	TiDBCluster  *v1alpha1.TidbCluster           `json:"crd_tidb_cluster"`
	Unstructured *unstructured.Unstructured      `json:"options"`
}

type Component struct {
	Replicas int32 `json:"replicas"`
}

type TiKVBackup struct {
	Component
	Stores []*StoresBackup `json:"stores"`
}

type StoresBackup struct {
	StoreID uint64          `json:"store_id"`
	Volumes []*VolumeBackup `json:"volumes"`
}

type VolumeBackup struct {
	VolumeID        string `json:"volume_id"`
	Type            string `json:"type"`
	MountPath       string `json:"mount_path"`
	SnapshotID      string `json:"snapshot_id"`
	RestoreVolumeID string `json:"restore_volume_id"`
}

func NewCloudSnapshotBackup(tc *v1alpha1.TidbCluster) (csb *CloudSnapBackup) {
	if tc != nil && tc.Spec.TiKV != nil {
		csb = &CloudSnapBackup{
			TiKV: &TiKVBackup{
				Component: Component{
					Replicas: tc.Spec.TiKV.Replicas,
				},
				Stores: []*StoresBackup{},
			},
			PD: Component{
				Replicas: tc.Spec.PD.Replicas,
			},
			TiDB: Component{
				Replicas: tc.Spec.TiDB.Replicas,
			},
			Kubernetes: &KubernetesBackup{
				PVs:          []*corev1.PersistentVolume{},
				PVCs:         []*corev1.PersistentVolumeClaim{},
				TiDBCluster:  tc,
				Unstructured: nil,
			},
			Options: nil,
		}
	}
	return
}

type StoresMixture struct {
	// TidbCluster as CRD
	tc *v1alpha1.TidbCluster
	// Pod as resource native to Kubernetes
	pod *corev1.Pod
	// PersistentVolumeClaim as resource native to Kubernetes
	pvcs []*corev1.PersistentVolumeClaim
	// PersistentVolume as resource native to Kubernetes
	pvs []*corev1.PersistentVolume
	// key: volumeName, value: mountPath
	volsMap map[string]string
	// key: mountPath, value: dirConfigType
	mpTypeMap map[string]string
	// key: mountPath, value: volumeID
	mpVolIDMap map[string]string
	// key: volumeID, value: restoreVolumeID, for restore
	rsVolIDMap map[string]string
	// support snapshot for the cloudprovider
	snapshotter Snapshotter
}

func NewBackupStoresMixture(
	tc *v1alpha1.TidbCluster,
	pvcs []*corev1.PersistentVolumeClaim,
	pvs []*corev1.PersistentVolume,
	s Snapshotter) *StoresMixture {
	return &StoresMixture{
		tc:          tc,
		pvcs:        pvcs,
		pvs:         pvs,
		volsMap:     make(map[string]string),
		mpTypeMap:   make(map[string]string),
		mpVolIDMap:  make(map[string]string),
		snapshotter: s,
	}
}

func NewRestoreStoresMixture(s Snapshotter) *StoresMixture {
	return &StoresMixture{
		rsVolIDMap:  make(map[string]string),
		snapshotter: s,
	}
}

func (m *StoresMixture) SetPod(pod *corev1.Pod) {
	m.pod = pod
}

func (m *StoresMixture) collectVolumesInfo() {
	if m.tc != nil && m.tc.Spec.TiKV != nil {
		for _, sv := range m.tc.Spec.TiKV.StorageVolumes {
			for k, v := range m.tc.Spec.TiKV.Config.Inner() {
				if sv.MountPath == v {
					m.mpTypeMap[sv.MountPath] = k
				}
			}
			volName := fmt.Sprintf("%s-%s", v1alpha1.TiKVMemberType.String(), sv.Name)
			m.volsMap[volName] = sv.MountPath
		}
	}

	m.volsMap[v1alpha1.TiKVMemberType.String()] = constants.TiKVDataVolumeMountPath
	m.mpTypeMap[constants.TiKVDataVolumeMountPath] = constants.TiKVDataVolumeConfType
}

func (m *StoresMixture) extractVolumeIDs() (string, error) {
	// key: volumePVCClaimName, value: mountPath
	claimMpMap := make(map[string]string)
	for _, vol := range m.pod.Spec.Volumes {
		if mp, ok := m.volsMap[vol.Name]; ok {
			claimMpMap[vol.VolumeSource.PersistentVolumeClaim.ClaimName] = mp
		}
	}

	// key: pvcName, value: mountPath
	pvcMpMap := make(map[string]string)
	for _, pvc := range m.pvcs {
		if mp, ok := claimMpMap[pvc.Name]; ok {
			pvcMpMap[pvc.Spec.VolumeName] = mp
		}
	}

	// key: mountPath, value: PV
	mpPVMap := make(map[string]*corev1.PersistentVolume)
	for _, pv := range m.pvs {
		if mp, ok := pvcMpMap[pv.Name]; ok {
			mpPVMap[mp] = pv
		}
	}

	// key: mountPath, value: volumeID
	mpVolIDMap := make(map[string]string)
	for mp, pv := range mpPVMap {
		volID, err := m.snapshotter.GetVolumeID(pv)
		if err != nil {
			return "GetVolumeIDFailed", err
		}
		mpVolIDMap[mp] = volID

		// set volume-id into pv-annotation temporarily for
		// restore to locate pv mapping volume-id quickly
		if pv.Annotations == nil {
			pv.Annotations = make(map[string]string)
		}
		pv.Annotations[constants.AnnTemporaryVolumeID] = volID
	}

	m.mpVolIDMap = mpVolIDMap
	return "", nil
}

func (m *StoresMixture) PrepareCSBStoresMeta(csb *CloudSnapBackup, pods []*corev1.Pod) (string, error) {
	if csb.TiKV.Stores == nil {
		csb.TiKV.Stores = []*StoresBackup{}
	}

	m.collectVolumesInfo()

	for _, pod := range pods {
		m.SetPod(pod)
		reason, err := m.extractVolumeIDs()
		if err != nil {
			return reason, err
		}

		storeID, _ := strconv.ParseUint(pod.Labels[label.StoreIDLabelKey], 10, 64)
		stores := &StoresBackup{
			StoreID: storeID,
			Volumes: []*VolumeBackup{},
		}
		for mp, volID := range m.mpVolIDMap {
			vol := &VolumeBackup{
				VolumeID:  volID,
				Type:      m.mpTypeMap[mp],
				MountPath: mp,
			}
			stores.Volumes = append(stores.Volumes, vol)
		}
		csb.TiKV.Stores = append(csb.TiKV.Stores, stores)
	}

	return "", nil
}

func (m *StoresMixture) ProcessCSBPVCsAndPVs(r *v1alpha1.Restore, csb *CloudSnapBackup) (string, error) {
	pvcs := csb.Kubernetes.PVCs
	pvs := csb.Kubernetes.PVs

	m.generateRestoreVolumeIDMap(csb.TiKV.Stores)

	backupClusterName := csb.Kubernetes.TiDBCluster.Name

	pvcMap := make(map[string]*corev1.PersistentVolumeClaim)
	for _, pvc := range pvcs {
		pvcMap[pvc.Name] = pvc
	}

	for _, pv := range pvs {
		klog.Infof("snapshotter-pv: %v", pv)

		if pv.Spec.ClaimRef == nil {
			return "PVClaimRefNil", fmt.Errorf("pv %s claimRef is nil", pv.Name)
		}
		pvc, ok := pvcMap[pv.Spec.ClaimRef.Name]
		if !ok {
			return "PVCNotFound", fmt.Errorf("pvc %s/%s not found", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
		}

		// Reset the PV's binding status so that Kubernetes can properly
		// associate it with the restored PVC.
		resetVolumeBindingInfo(pvc, pv)

		// Reset the PV's volumeID for restore from snapshot
		// Note: This function has to precede the following `resetMetadataAndStatus`
		if reason, err := m.resetRestoreVolumeID(pv); err != nil {
			return reason, err
		}

		// Clear out non-core metadata fields and status
		resetMetadataAndStatus(r, backupClusterName, pvc, pv)
	}

	return "", nil
}

func (m *StoresMixture) generateRestoreVolumeIDMap(stores []*StoresBackup) {
	vols := []*VolumeBackup{}
	for _, store := range stores {
		vols = append(vols, store.Volumes...)
	}
	for _, vol := range vols {
		m.rsVolIDMap[vol.VolumeID] = vol.RestoreVolumeID
	}
}

func (m *StoresMixture) resetRestoreVolumeID(pv *corev1.PersistentVolume) (string, error) {
	// the reason for not using `snapshotter.GetVolumeID` directly here is to consider using
	// a different driver or provisioner for backup and restore to extend conveniently later
	volID, ok := pv.Annotations[constants.AnnTemporaryVolumeID]
	if !ok {
		return "GetVolumeIDFailed", fmt.Errorf("%s pv.annotations[%s] not found", pv.GetName(), constants.AnnTemporaryVolumeID)
	}

	var rsVolID string
	if rsVolID, ok = m.rsVolIDMap[volID]; !ok {
		return "GetRestoreVolumeIDFailed", fmt.Errorf("%s volumeID-%s mapping restoreVolumeID not found", pv.GetName(), volID)
	}

	if err := m.snapshotter.SetVolumeID(pv, rsVolID); err != nil {
		return "ResetRestoreVolumeIDFailed", fmt.Errorf("failed to set pv-%s, %s", pv.Name, err.Error())
	}

	return "", nil
}

// resetVolumeBindingInfo clears any necessary metadata out of a PersistentVolume
// or PersistentVolumeClaim that would make it ineligible to be re-bound.
func resetVolumeBindingInfo(pvc *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume) {
	// Upon restore, this new PV will look like a statically provisioned, manually-
	// bound volume rather than one bound by the controller, so remove the annotation
	// that signals that a controller bound it.
	delete(pvc.Annotations, constants.KubeAnnBindCompleted)
	// Remove the annotation that signals that the PV is already bound; we want
	// the PV(C) controller to take the two objects and bind them again.
	delete(pvc.Annotations, constants.KubeAnnBoundByController)

	// Clean out ClaimRef UID and resourceVersion, since this information is
	// highly unique.
	if pv.Spec.ClaimRef != nil {
		pv.Spec.ClaimRef.ResourceVersion = ""
		pv.Spec.ClaimRef.UID = ""
	}

	// Remove the provisioned-by annotation which signals that the persistent
	// volume was dynamically provisioned; it is now statically provisioned.
	delete(pv.Annotations, constants.KubeAnnDynamicallyProvisioned)
}

func resetMetadataAndStatus(
	r *v1alpha1.Restore,
	backupClusterName string,
	pvc *corev1.PersistentVolumeClaim,
	pv *corev1.PersistentVolume,
) {
	// clear out metadata except core fields: "name", "namespace", "labels", "annotations"
	pvc.ObjectMeta = newMetaWithCoreFields(pvc.ObjectMeta)
	pv.ObjectMeta = newMetaWithCoreFields(pv.ObjectMeta)

	// clear out the annotation for store temporary volumeID
	delete(pv.Annotations, constants.AnnTemporaryVolumeID)

	restoreClusterNamespace := r.Spec.BR.ClusterNamespace
	restoreClusterName := r.Spec.BR.Cluster
	// The restore cluster is not the same as the backup cluster, we need to reset the
	// pvc/pv namespace and name to the restore cluster's namespace and name.
	if restoreClusterNamespace != pvc.Namespace || restoreClusterName != backupClusterName {
		// The PVC name format is tikv[-${additionalVolumeName}]-${statefulSetName}-${ordinal}.
		// We need to replace the statefulSetName with the restore cluster's statefulSetName.
		backupStatefulSetName := controller.TiKVMemberName(backupClusterName)
		restoreStatefulSetName := controller.TiKVMemberName(restoreClusterName)
		newPVCName := regexp.MustCompile(fmt.Sprintf("%s-([0-9]+)$", backupStatefulSetName)).
			ReplaceAllString(pvc.Name, fmt.Sprintf("%s-$1", restoreStatefulSetName))
		klog.Infof("reset PVC %s/%s to %s/%s", pvc.Namespace, pvc.Name, restoreClusterNamespace, newPVCName)
		pvc.Namespace = restoreClusterNamespace
		pvc.Name = newPVCName

		newPVName := "pvc-" + string(uuid.NewUUID())
		klog.Infof("reset PV %s to %s", pv.Name, newPVName)
		pv.Name = newPVName

		if pv.Spec.ClaimRef != nil {
			pv.Spec.ClaimRef.Name = newPVCName
			pv.Spec.ClaimRef.Namespace = restoreClusterNamespace
		}
		pvc.Spec.VolumeName = pv.Name

		if pvc.Labels != nil {
			pvc.Labels[label.InstanceLabelKey] = restoreClusterName
			if podName, ok := pvc.Labels[label.AnnPodNameKey]; ok {
				pvc.Labels[label.AnnPodNameKey] = strings.ReplaceAll(podName, backupStatefulSetName, restoreStatefulSetName)
			}
		}
		if pv.Labels != nil {
			pv.Labels[label.InstanceLabelKey] = restoreClusterName
			pv.Labels[label.NamespaceLabelKey] = restoreClusterNamespace
		}
	}

	// Never restore status
	pvc.Status = corev1.PersistentVolumeClaimStatus{}
	pv.Status = corev1.PersistentVolumeStatus{}
}

func newMetaWithCoreFields(meta metav1.ObjectMeta) metav1.ObjectMeta {
	newMeta := metav1.ObjectMeta{
		Name:        meta.GetName(),
		Namespace:   meta.GetNamespace(),
		Labels:      meta.GetLabels(),
		Annotations: meta.GetAnnotations(),
	}
	return newMeta
}
