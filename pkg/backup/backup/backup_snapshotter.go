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

type CloudSnapshotBackup struct {
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
	VolumesInfo []*VolumesInfoBackup `json:"volumes_info"`
}

type VolumesInfoBackup struct {
	StoreID string          `json:"store_id"`
	Volumes []*VolumeBackup `json:"volumes"`
}

type VolumeBackup struct {
	VolumeID  string `json:"volume_id"`
	Type      string `json:"type"`
	MountPath string `json:"mount_path"`
}

func NewCloudSnapshotBackup(tc *v1alpha1.TidbCluster) (csb *CloudSnapshotBackup) {
	if tc != nil && tc.Spec.TiKV != nil {
		csb = &CloudSnapshotBackup{
			TiKV: &TiKVBackup{
				Component: Component{
					Replicas: tc.Spec.TiKV.Replicas,
				},
				VolumesInfo: []*VolumesInfoBackup{},
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

type VolumesMixture struct {
	tc          *v1alpha1.TidbCluster
	pod         *corev1.Pod
	pvcs        []*corev1.PersistentVolumeClaim
	pvs         []*corev1.PersistentVolume
	volsMap     map[string]string
	mpTypeMap   map[string]string
	mpVolIDMap  map[string]string
	snapshotter Snapshotter
}

func NewIntactVolumesMixture(
	tc *v1alpha1.TidbCluster,
	pod *corev1.Pod,
	pvcs []*corev1.PersistentVolumeClaim,
	pvs []*corev1.PersistentVolume,
	volsMap map[string]string,
	mpTypeMap map[string]string,
	mpVolIDMap map[string]string,
	s Snapshotter) *VolumesMixture {
	return &VolumesMixture{
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

func NewVolumesMixture(
	tc *v1alpha1.TidbCluster,
	pvcs []*corev1.PersistentVolumeClaim,
	pvs []*corev1.PersistentVolume,
	s Snapshotter) *VolumesMixture {
	return &VolumesMixture{
		tc:          tc,
		pvcs:        pvcs,
		pvs:         pvs,
		volsMap:     make(map[string]string),
		mpTypeMap:   make(map[string]string),
		mpVolIDMap:  make(map[string]string),
		snapshotter: s,
	}
}

func (m *VolumesMixture) SetPod(pod *corev1.Pod) {
	m.pod = pod
}

func (m *VolumesMixture) CollectVolumesInfo() {
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
	m.mpTypeMap[constants.TiKVDataVolumeMountPath] = constants.TiKVDataVolumeType
}

func (m *VolumesMixture) ExtractVolumeIDs() (string, error) {
	mpVolMap := make(map[string]corev1.Volume)
	for _, vol := range m.pod.Spec.Volumes {
		if mp, ok := m.volsMap[vol.Name]; ok {
			mpVolMap[mp] = vol
		}
	}

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

func (s *BaseSnapshotter) prepareBackupMetadata(
	b *v1alpha1.Backup, tc *v1alpha1.TidbCluster, ns string, impl Snapshotter) (string, error) {
	csb := NewCloudSnapshotBackup(tc)

	req, err := labels.NewRequirement(label.ComponentLabelKey, selection.Equals, []string{label.TiKVLabelVal})
	if err != nil {
		return fmt.Sprintf("unexpected error generating label selector: %v", err), err
	}
	sel := labels.NewSelector().Add(*req)
	pvcs, err := s.backupMgr.deps.PVCLister.PersistentVolumeClaims(ns).List(sel)

	if err != nil {
		return fmt.Sprintf("failed to fetch pvcs %s:%s", label.ComponentLabelKey, label.TiKVLabelVal), err
	}
	csb.Kubernetes.PVCs = pvcs
	pvs, err := s.backupMgr.deps.PVLister.List(sel)
	if err != nil {
		return fmt.Sprintf("failed to fetch pvs %s:%s", label.ComponentLabelKey, label.TiKVLabelVal), err
	}
	csb.Kubernetes.PVs = pvs
	pods, err := s.backupMgr.deps.PodLister.Pods(ns).List(sel)
	if err != nil {
		return fmt.Sprintf("failed to fetch pods %s:%s", label.ComponentLabelKey, label.TiKVLabelVal), err
	}

	volMix := NewVolumesMixture(tc, pvcs, pvs, impl)
	volMix.CollectVolumesInfo()

	for _, pod := range pods {
		volMix.SetPod(pod)
		reason, err := volMix.ExtractVolumeIDs()
		if err != nil {
			return reason, err
		}

		volsInfo := &VolumesInfoBackup{
			StoreID: pod.Labels[label.StoreIDLabelKey],
			Volumes: []*VolumeBackup{},
		}
		for mp, volID := range volMix.mpVolIDMap {
			vol := &VolumeBackup{
				VolumeID:  volID,
				Type:      volMix.mpTypeMap[mp],
				MountPath: mp,
			}
			volsInfo.Volumes = append(volsInfo.Volumes, vol)
		}
		csb.TiKV.VolumesInfo = append(csb.TiKV.VolumesInfo, volsInfo)
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