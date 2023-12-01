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
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
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

	// GenerateBackupMetadata performs the preparations for creating
	// a snapshot from the used volume for PV/PVC.
	GenerateBackupMetadata(b *v1alpha1.Backup, tc *v1alpha1.TidbCluster) (*CloudSnapBackup, string, error)

	// PrepareRestoreMetadata performs the preparations for creating
	// a volume from the snapshot that has been backed up.
	PrepareRestoreMetadata(r *v1alpha1.Restore, csb *CloudSnapBackup) (string, error)

	// SetVolumeID sets the cloud provider specific identifier
	// for the PersistentVolume-PV.
	SetVolumeID(pv *corev1.PersistentVolume, volumeID string) error

	// ResetPvAvailableZone resets az of pv if the volumes restore to another az
	ResetPvAvailableZone(r *v1alpha1.Restore, pv *corev1.PersistentVolume)

	// AddVolumeTags add operator related tags to volumes
	AddVolumeTags(pvs []*corev1.PersistentVolume) error
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

func (s *BaseSnapshotter) generateBackupMetadata(
	b *v1alpha1.Backup, tc *v1alpha1.TidbCluster, execr Snapshotter) (*CloudSnapBackup, string, error) {
	csb := NewCloudSnapshotBackup(tc)
	pods, reason, err := s.PrepareCSBK8SMeta(csb, tc)
	if err != nil {
		return nil, reason, err
	}

	m := NewBackupStoresMixture(tc, csb.Kubernetes.PVCs, csb.Kubernetes.PVs, execr)
	if reason, err := m.PrepareCSBStoresMeta(csb, pods); err != nil {
		return nil, reason, err
	}

	return csb, "", nil
}

func (s *BaseSnapshotter) prepareRestoreMetadata(r *v1alpha1.Restore, csb *CloudSnapBackup, execr Snapshotter) (string, error) {
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
	clusterNamespace := r.Spec.BR.ClusterNamespace
	if clusterNamespace == "" {
		clusterNamespace = r.Namespace
	}
	pvSel, err := label.New().Instance(r.Spec.BR.Cluster).TiKV().Namespace(clusterNamespace).Selector()
	if err != nil {
		return "BuildTiKVPvSelectorFailed", err
	}
	existingPVs, err := deps.PVLister.List(pvSel)
	if err != nil {
		return "ListPVsFailed", err
	}

	refPVC2PVMap := make(map[string]*corev1.PersistentVolume)
	for _, pv := range existingPVs {
		if pv.Spec.ClaimRef != nil {
			namespacedName := buildNamespacedName(pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
			refPVC2PVMap[namespacedName] = pv
		}
	}
	pvcMap := make(map[string]*corev1.PersistentVolumeClaim)
	for _, pvc := range pvcs {
		pvcMap[pvc.Name] = pvc
	}

	// Since we may generate random PV names, to avoid creating duplicate PVs,
	// we need to check if the PV already exists before creating it.
	for _, pv := range pvs {
		if pv.Spec.ClaimRef != nil {
			namespacedName := buildNamespacedName(pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
			if existingPV, ok := refPVC2PVMap[namespacedName]; ok {
				// check the existing pv with same pvc ref if created by this restore
				if existingPV.Annotations[constants.AnnRestoredVolumeID] == pv.Annotations[constants.AnnRestoredVolumeID] {
					klog.Infof("Restore %s/%s existing pv %s has same volume id %s, skip creating pv %s",
						r.Namespace, r.Name, existingPV.Name, existingPV.Annotations[constants.AnnRestoredVolumeID], pv.Name)

					// restored pv has already created, bind corresponding pvc with existing pv
					pvc := pvcMap[pv.Spec.ClaimRef.Name]
					pvc.Spec.VolumeName = existingPV.Name
					klog.Infof("Restore %s/%s bind pvc %s with existing pv %s",
						r.Namespace, r.Name, pvc.Name, existingPV.Name)
					continue
				} else {
					klog.Warningf("Restore %s/%s finds existing pv %s has same pvc ref with the pv %s we will create. Maybe there is volume leak, please take a look",
						r.Namespace, r.Name, existingPV.Name, pv.Name)
				}
			}
		}
		if err := deps.PVControl.CreatePV(r, pv); err != nil {
			return "CreatePVFailed", err
		}
	}

	pvcSel, err := label.New().Instance(r.Spec.BR.Cluster).TiKV().Selector()
	if err != nil {
		return "BuildTiKVPvcSelectorFailed", err
	}
	existingPVCs, err := deps.PVCLister.PersistentVolumeClaims(clusterNamespace).List(pvcSel)
	if err != nil {
		return "ListPVCsFailed", err
	}
	existingPVCMap := make(map[string]*corev1.PersistentVolumeClaim, len(existingPVCs))
	for _, pvc := range existingPVCs {
		existingPVCMap[pvc.Name] = pvc
	}

	for _, pvc := range pvcs {
		if existingPVC, ok := existingPVCMap[pvc.Name]; ok {
			// check if the existing pvc is created by this restore
			if existingPVC.Spec.VolumeName == pvc.Spec.VolumeName {
				klog.Infof("Restore %s/%s the pvc %s is already existing, skip it", r.Namespace, r.Name, pvc.Name)
				continue
			} else {
				return "ExistingPVCConflict", fmt.Errorf(
					"pvc %s/%s already exists, and has different volume. please remove it carefully to continue volume restore process",
					existingPVC.Namespace, existingPVC.Name)
			}
		}

		if err := deps.PVCControl.CreatePVC(r, pvc); err != nil {
			if apierrors.IsAlreadyExists(err) {
				continue
			}
			return "CreatePVCFailed", err
		}
	}

	return "", nil
}

func buildNamespacedName(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
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
					Replicas: int32(len(tc.Status.TiKV.Stores) + len(tc.Status.TiKV.PeerStores)),
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
	m.generateRestoreVolumeIDMap(csb.TiKV.Stores)

	backupClusterName := csb.Kubernetes.TiDBCluster.Name
	pvcMap := make(map[string]*corev1.PersistentVolumeClaim)
	for _, pvc := range csb.Kubernetes.PVCs {
		pvcMap[pvc.Name] = pvc
	}
	volID2PV := make(map[string]*corev1.PersistentVolume)
	for _, pv := range csb.Kubernetes.PVs {
		volID, ok := pv.Annotations[constants.AnnTemporaryVolumeID]
		if ok {
			volID2PV[volID] = pv
		}
	}

	pvs, pvcs := make([]*corev1.PersistentVolume, 0, len(m.rsVolIDMap)), make([]*corev1.PersistentVolumeClaim, 0, len(m.rsVolIDMap))
	for backupVolID, restoreVolID := range m.rsVolIDMap {
		pv, ok := volID2PV[backupVolID]
		if !ok {
			return "GetPVFailed", fmt.Errorf("pv with volume id %s not found", backupVolID)
		}

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
		if err := m.snapshotter.SetVolumeID(pv, restoreVolID); err != nil {
			return "ResetRestoreVolumeIDFailed", fmt.Errorf("failed to set pv-%s, %s", pv.Name, err.Error())
		}
		// Clear out non-core metadata fields and status
		resetMetadataAndStatus(r, backupClusterName, pvc, pv)
		m.snapshotter.ResetPvAvailableZone(r, pv)
		pv.Annotations[constants.AnnRestoredVolumeID] = restoreVolID

		pvs = append(pvs, pv)
		pvcs = append(pvcs, pvc)
	}

	restoreSTSName := controller.TiKVMemberName(r.Spec.BR.Cluster)
	sequentialPVCs, sequentialPVs, err := resetPVCSequence(restoreSTSName, pvcs, pvs)
	if err != nil {
		klog.Errorf("reset pvcs to sequential error: %s", err.Error())
		return "InvalidPVCName", err
	}

	csb.Kubernetes.PVCs = sequentialPVCs
	csb.Kubernetes.PVs = sequentialPVs
	return "", nil
}

// If a backup cluster has scaled in and the tidb-operator enabled advanced-statefulset(ref: https://docs.pingcap.com/zh/tidb-in-kubernetes/stable/advanced-statefulset)
// the name of pvc can be not sequential, and every tikv pod can have multiple pvc, eg:
// [tikv-db-tikv-0, tikv-db-tikv-2, tikv-db-tikv-3, tikv-raft-db-tikv-0, tikv-raft-db-tikv-2, tikv-raft-db-tikv-3]
// the format of tikv pvc name is {volume_name}-{tc_name}-tikv-{index}
// When create cluster, we should ensure pvc name with same volume sequential, so we should reset it to
// [tikv-db-tikv-0, tikv-db-tikv-1, tikv-db-tikv-2, tikv-raft-db-tikv-0, tikv-raft-db-tikv-1, tikv-raft-db-tikv-2]
func resetPVCSequence(stsName string, pvcs []*corev1.PersistentVolumeClaim, pvs []*corev1.PersistentVolume) (
	[]*corev1.PersistentVolumeClaim, []*corev1.PersistentVolume, error) {
	type indexedPVC struct {
		index int
		pvc   *corev1.PersistentVolumeClaim
	}
	indexedPVCsGroups := make(map[string][]*indexedPVC, 8)
	reStr := fmt.Sprintf(`^(.+)-%s-(\d+)$`, stsName)
	re := regexp.MustCompile(reStr)
	// because one tikv may have multiple volumes, corresponding multiple pvc, and every pvc has serial number.
	// we should split pvc to multiple groups by volume name, and make the pvc sequential in every group
	for _, pvc := range pvcs {
		subMatches := re.FindStringSubmatch(pvc.Name)
		// subMatches contains full text that matches regex and the matches in brackets.
		// so if the pvc matches regex, it should contain 3 items.
		if len(subMatches) != 3 {
			return nil, nil, fmt.Errorf("pvc name %s doesn't match regex %s", pvc.Name, reStr)
		}
		volumeName := subMatches[1]
		pvcNumberStr := subMatches[2]
		// get the number of pvc, for example, there is a pvc "tikv-db-tikv-0", try to get the number "0"
		index, err := strconv.Atoi(pvcNumberStr)
		if err != nil {
			return nil, nil, fmt.Errorf("parse index %s of pvc %s to int: %s", pvcNumberStr, pvc.Name, err.Error())
		}
		// get the volume name of pod from pvc, for example, there is a pvc "tikv-db-tikv-0", get "tikv"
		// there can be multiple volumes in a pod, so we should group pvc by volume name
		indexedPVCs, ok := indexedPVCsGroups[volumeName]
		if !ok {
			indexedPVCs = make([]*indexedPVC, 0, len(pvcs))
		}
		indexedPVCsGroups[volumeName] = append(indexedPVCs, &indexedPVC{
			index: index,
			pvc:   pvc,
		})
	}

	for _, indexedPVCs := range indexedPVCsGroups {
		sort.Slice(indexedPVCs, func(i, j int) bool {
			return indexedPVCs[i].index < indexedPVCs[j].index
		})
	}
	pvc2pv := make(map[string]*corev1.PersistentVolume, len(pvs))
	for _, pv := range pvs {
		pvc2pv[pv.Spec.ClaimRef.Name] = pv
	}

	sequentialPVCs := make([]*corev1.PersistentVolumeClaim, 0, len(pvcs))
	sequentialPVs := make([]*corev1.PersistentVolume, 0, len(pvs))
	// for every pvc group, make the pvc list in the group sequential
	for _, indexedPVCs := range indexedPVCsGroups {
		for i, iPVC := range indexedPVCs {
			pv, ok := pvc2pv[iPVC.pvc.Name]
			if !ok {
				return nil, nil, fmt.Errorf("pv with claim %s not found", iPVC.pvc.Name)
			}
			// when create cluster, tidb-operator will create tikv instances from 0 to n-1, and pvcs are also [0, n-1]
			// for backup cluster, advanced statefulset allows scaling in at arbitrary position
			// so pvcs of backup might not be [0, n-1], we should reset it to [0, n-1]
			if i != iPVC.index {
				names := strings.Split(iPVC.pvc.Name, "-")
				restorePVCName := fmt.Sprintf("%s-%d", strings.Join(names[:len(names)-1], "-"), i)

				klog.Infof("reset pvc name %s to %s", iPVC.pvc.Name, restorePVCName)
				iPVC.pvc.Name = restorePVCName
				pv.Spec.ClaimRef.Name = restorePVCName
			}
			sequentialPVCs = append(sequentialPVCs, iPVC.pvc)
			sequentialPVs = append(sequentialPVs, pv)
		}
	}
	return sequentialPVCs, sequentialPVs, nil
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
