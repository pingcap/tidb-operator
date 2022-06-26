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

package backup

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

type Snapshotter interface {
	// Init prepares the Snapshotter for usage, it would add specific parameters as config map[string]string in the future.
	Init(bm *backupManager, conf map[string]string) error

	// GetVolumeID returns the cloud provider specific identifier for the PersistentVolume.
	GetVolumeID(pv *corev1.PersistentVolume) (string, error)

	PrepareBackupMetadata(b *v1alpha1.Backup, tc *v1alpha1.TidbCluster, ns string) (string, error)

	// SetVolumeID sets the cloud provider specific identifier for the PersistentVolume.
	// SetVolumeID(pv *corev1.PersistentVolume, volumeID string) (*corev1.PersistentVolume, error)
}

type BaseSnapshotter struct {
	volRegexp *regexp.Regexp
	backupMgr *backupManager
	config    map[string]string
}

func (s *BaseSnapshotter) Init(bm *backupManager, conf map[string]string) error {
	s.backupMgr = bm
	s.config = conf
	return nil
}

type AWSSnapshotter struct {
	BaseSnapshotter
}

func (s *AWSSnapshotter) Init(bm *backupManager, conf map[string]string) error {
	s.BaseSnapshotter.Init(bm, conf)
	s.volRegexp = regexp.MustCompile("vol-.*")
	return nil
}

func (s *AWSSnapshotter) GetVolumeID(pv *corev1.PersistentVolume) (string, error) {
	if pv == nil {
		return "", nil
	}

	if pv.Spec.CSI != nil {
		driver := pv.Spec.CSI.Driver
		if driver == constants.EbsCSIDriver {
			return s.volRegexp.FindString(pv.Spec.CSI.VolumeHandle), nil
		}
		return "", fmt.Errorf("unable to handle CSI driver: %s", driver)
	}
	if pv.Spec.AWSElasticBlockStore != nil {
		if pv.Spec.AWSElasticBlockStore.VolumeID == "" {
			return "", fmt.Errorf("spec.awsElasticBlockStore.volumeID not found")
		}
		return s.volRegexp.FindString(pv.Spec.AWSElasticBlockStore.VolumeID), nil
	}

	return "", nil
}

type GCPSnapshotter struct {
	BaseSnapshotter
}

func (s *GCPSnapshotter) Init(bm *backupManager, conf map[string]string) error {
	s.BaseSnapshotter.Init(bm, conf)
	s.volRegexp = regexp.MustCompile(`^projects\/[^\/]+\/(zones|regions)\/[^\/]+\/disks\/[^\/]+$`)
	return nil
}

func (s *GCPSnapshotter) GetVolumeID(pv *corev1.PersistentVolume) (string, error) {
	if pv == nil {
		return "", nil
	}

	if pv.Spec.CSI != nil {
		driver := pv.Spec.CSI.Driver
		if driver == constants.PdCSIDriver {
			handle := pv.Spec.CSI.VolumeHandle
			if !s.volRegexp.MatchString(handle) {
				return "", fmt.Errorf("invalid volumeHandle for CSI driver:%s, expected projects/{project}/zones/{zone}/disks/{name}, got %s",
					constants.PdCSIDriver, handle)
			}
			l := strings.Split(handle, "/")
			return l[len(l)-1], nil
		}
		return "", fmt.Errorf("unable to handle CSI driver: %s", driver)
	}

	if pv.Spec.GCEPersistentDisk != nil {
		if pv.Spec.GCEPersistentDisk.PDName == "" {
			return "", fmt.Errorf("spec.gcePersistentDisk.pdName not found")
		}
		return pv.Spec.GCEPersistentDisk.PDName, nil
	}

	return "", nil
}

type CloudProviderFactory struct {
}

type CloudProviderInter interface {
	CreateSnapshotter(bt v1alpha1.BackupType) Snapshotter
}

// AWSElasticBlockStore/GCEPersistentDisk
func (cpf *CloudProviderFactory) CreateSnapshotter(bt v1alpha1.BackupType) (s Snapshotter) {
	switch bt {
	case "ebs":
		s = new(AWSSnapshotter)
	case "gcepd":
		s = new(GCPSnapshotter)
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
	VolumeID  string `json:"volume_id"`
	Type      string `json:"type"`
	MountPath string `json:"mount_path"`
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
	// support snapshot for the cloudprovider
	snapshotter Snapshotter
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

func NewStoresMixture(
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
	// key: mountPath, value: Volume
	mpVolMap := make(map[string]corev1.Volume)
	for _, vol := range m.pod.Spec.Volumes {
		if mp, ok := m.volsMap[vol.Name]; ok {
			mpVolMap[mp] = vol
		}
	}

	// key: mountPath, value: PV
	// TODO: optimize the loop as repeated comparisons
	mpPVMap := make(map[string]*corev1.PersistentVolume)
	for mp, vol := range mpVolMap {
		for _, pvc := range m.pvcs {
			if pvc.Name == vol.VolumeSource.PersistentVolumeClaim.ClaimName {
				for _, pv := range m.pvs {
					if pvc.Spec.VolumeName == pv.Name {
						mpPVMap[mp] = pv
						break
					}
				}
				break
			}
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
	}

	m.mpVolIDMap = mpVolIDMap
	return "", nil
}

func (s *BaseSnapshotter) PrepareCSBK8SMeta(csb *CloudSnapBackup, ns string) ([]*corev1.Pod, string, error) {
	if s.backupMgr == nil {
		return nil, "NotExistBackupManager", fmt.Errorf("unexpected error for backup-manager is nil")
	}
	req, err := labels.NewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.TiKVLabelVal})
	if err != nil {
		return nil, fmt.Sprintf("unexpected error generating label selector: %v", err), err
	}
	sel := labels.NewSelector().Add(*req)
	pvcs, err := s.backupMgr.deps.PVCLister.PersistentVolumeClaims(ns).List(sel)

	if err != nil {
		return nil, fmt.Sprintf("failed to fetch pvcs %s:%s", label.ComponentLabelKey, label.TiKVLabelVal), err
	}
	csb.Kubernetes.PVCs = pvcs
	pvs, err := s.backupMgr.deps.PVLister.List(sel)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch pvs %s:%s", label.ComponentLabelKey, label.TiKVLabelVal), err
	}
	csb.Kubernetes.PVs = pvs
	pods, err := s.backupMgr.deps.PodLister.Pods(ns).List(sel)
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

	storesMix := NewStoresMixture(tc, csb.Kubernetes.PVCs, csb.Kubernetes.PVs, execr)
	reason, err = storesMix.PrepareCSBStoresMeta(csb, pods)
	if err != nil {
		return reason, err
	}

	out, err := json.Marshal(csb)
	if err != nil {
		return "ParseCloudSnapshotBackupFailed", err
	}
	b.Annotations[label.AnnBackupCloudSnapKey] = string(out)
	return "", nil
}

func (s *AWSSnapshotter) PrepareBackupMetadata(b *v1alpha1.Backup, tc *v1alpha1.TidbCluster, ns string) (string, error) {
	return s.BaseSnapshotter.prepareBackupMetadata(b, tc, ns, s)
}

func (s *GCPSnapshotter) PrepareBackupMetadata(b *v1alpha1.Backup, tc *v1alpha1.TidbCluster, ns string) (string, error) {
	return s.BaseSnapshotter.prepareBackupMetadata(b, tc, ns, s)
}
