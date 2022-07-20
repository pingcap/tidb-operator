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

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/backup/util"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
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
	PrepareBackupMetadata(b *v1alpha1.Backup, tc *v1alpha1.TidbCluster, ns string) (string, error)

	// PrepareRestoreMetadata performs the preparations for creating
	// a volume from the snapshot that has been backed up.
	PrepareRestoreMetadata(r *v1alpha1.Restore) (string, error)

	// SetVolumeID sets the cloud provider specific identifier
	// for the PersistentVolume-PV.
	SetVolumeID(pv *corev1.PersistentVolume, volumeID string) error
}

type BaseSnapshotter struct {
	volRegexp *regexp.Regexp
	deps      *controller.Dependencies
	config    map[string]string
}

func (s *BaseSnapshotter) Init(deps *controller.Dependencies, conf map[string]string) error {
	s.deps = deps
	s.config = conf
	return nil
}

type CloudProviderFactory struct {
}

func NewCloudProviderFactory() *CloudProviderFactory {
	return &CloudProviderFactory{}
}

type CloudProviderInter interface {
	CreateSnapshotter(bt v1alpha1.BackupType) Snapshotter
}

// AWSElasticBlockStore/GCEPersistentDisk
func (cpf *CloudProviderFactory) CreateSnapshotter(bt v1alpha1.BackupType) (s Snapshotter) {
	switch bt {
	case v1alpha1.BackupTypeEBS:
		s = new(AWSSnapshotter)
	case v1alpha1.BackupTypeGCEPD:
		s = new(GCPSnapshotter)
	case v1alpha1.BackupTypeData:
		s = new(NoneSnapshotter)
	default:
		// do nothing and return nil directly
		return
	}
	return
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
	// dry-run mode for local pv test
	dryRun bool
}

func NewIntactStoresMixture(
	tc *v1alpha1.TidbCluster,
	pod *corev1.Pod,
	pvcs []*corev1.PersistentVolumeClaim,
	pvs []*corev1.PersistentVolume,
	volsMap map[string]string,
	mpTypeMap map[string]string,
	mpVolIDMap map[string]string,
	s Snapshotter) *StoresMixture {
	return &StoresMixture{
		tc:          tc,
		pod:         pod,
		pvcs:        pvcs,
		pvs:         pvs,
		volsMap:     volsMap,
		mpTypeMap:   mpTypeMap,
		mpVolIDMap:  mpVolIDMap,
		snapshotter: s,
	}
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

func NewRestoreStoresMixture(s Snapshotter, dryRun bool) *StoresMixture {
	return &StoresMixture{
		rsVolIDMap:  make(map[string]string),
		snapshotter: s,
		dryRun:      dryRun,
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

func (s *BaseSnapshotter) PrepareCSBK8SMeta(csb *CloudSnapBackup, ns string) ([]*corev1.Pod, string, error) {
	if s.deps == nil {
		return nil, "NotExistDependencies", fmt.Errorf("unexpected error for nil dependencies")
	}
	req, err := labels.NewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.TiKVLabelVal})
	if err != nil {
		return nil, fmt.Sprintf("unexpected error generating label selector: %v", err), err
	}
	sel := labels.NewSelector().Add(*req)
	pvcs, err := s.deps.PVCLister.PersistentVolumeClaims(ns).List(sel)

	if err != nil {
		return nil, fmt.Sprintf("failed to fetch pvcs %s:%s", label.ComponentLabelKey, label.TiKVLabelVal), err
	}
	csb.Kubernetes.PVCs = pvcs
	pvs, err := s.deps.PVLister.List(sel)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch pvs %s:%s", label.ComponentLabelKey, label.TiKVLabelVal), err
	}
	csb.Kubernetes.PVs = pvs
	pods, err := s.deps.PodLister.Pods(ns).List(sel)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch pods %s:%s", label.ComponentLabelKey, label.TiKVLabelVal), err
	}
	return pods, "", nil
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

func (s *BaseSnapshotter) prepareBackupMetadata(
	b *v1alpha1.Backup, tc *v1alpha1.TidbCluster, ns string, execr Snapshotter) (string, error) {
	csb := NewCloudSnapshotBackup(tc)
	pods, reason, err := s.PrepareCSBK8SMeta(csb, ns)
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

	b.Annotations[label.AnnBackupCloudSnapKey] = util.BytesToString(out)
	return "", nil
}

func (m *StoresMixture) ProcessCSBPVCsAndPVs(csb *CloudSnapBackup) (string, error) {
	pvcs := csb.Kubernetes.PVCs
	pvs := csb.Kubernetes.PVs

	m.generateRestoreVolumeIDMap(csb.TiKV.Stores)

	for i, pv := range pvs {
		klog.Infof("snapshotter-pv: %v", pv)

		// Reset the PV's binding status so that Kubernetes can properly
		// associate it with the restored PVC.
		resetVolumeBindingInfo(pvcs[i], pv)

		// Reset the PV's volumeID for restore from snapshot
		// Note: This function has to precede the following `resetMetadataAndStatus`
		if reason, err := m.resetRestoreVolumeID(pv); err != nil {
			return reason, err
		}

		// Clear out non-core metadata fields and status
		resetMetadataAndStatus(pvcs[i], pv)
	}

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

	m := NewRestoreStoresMixture(execr, r.Spec.DryRun)
	if reason, err := m.ProcessCSBPVCsAndPVs(csb); err != nil {
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

func commitPVsAndPVCsToK8S(
	deps *controller.Dependencies,
	r *v1alpha1.Restore,
	pvcs []*corev1.PersistentVolumeClaim,
	pvs []*corev1.PersistentVolume) (string, error) {
	if r.Spec.DryRun {
		return "", nil
	}

	for _, pv := range pvs {
		if err := deps.PVControl.CreatePV(r, pv); err != nil {
			return "CreatePVFailed", err
		}
	}

	for _, pvc := range pvcs {
		if err := deps.PVCControl.CreatePVC(r, pvc); err != nil {
			return "CreatePVCFailed", err
		}
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

	if m.dryRun {
		return "", nil
	}

	if err := m.snapshotter.SetVolumeID(pv, rsVolID); err != nil {
		return "ResetRestoreVolumeIDFailed", fmt.Errorf("failed to set pv-%s, %s", pv.Name, err.Error())
	}

	return "", nil
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

func resetMetadataAndStatus(pvc *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume) {
	// clear out metadata except core fields: "name", "namespace", "labels", "annotations"
	pvc.ObjectMeta = newMetaWithCoreFields(pvc.ObjectMeta)
	pv.ObjectMeta = newMetaWithCoreFields(pv.ObjectMeta)

	// clear out the annotation for store temporary volumeID
	delete(pv.Annotations, constants.AnnTemporaryVolumeID)

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

func NewDefaultSnapshotter(t v1alpha1.BackupType, d *controller.Dependencies) (Snapshotter, string, error) {
	f := NewCloudProviderFactory()
	s := f.CreateSnapshotter(t)
	if s == nil {
		return nil, "", nil
	}

	err := s.Init(d, nil)
	if err != nil {
		return s, "InitSnapshotterFailed", err
	}

	return s, "", nil
}
