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
	"strconv"
	"strings"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/backup/testutils"
	"github.com/r3labs/diff/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetVolumeID(t *testing.T) {
	type getPVWantedVolumeID func(pv *corev1.PersistentVolume) (string, bool)

	cases := []struct {
		name string
		s    Snapshotter
		pv   *corev1.PersistentVolume
		f1   func(pv *corev1.PersistentVolume)
		f2   func(pv *corev1.PersistentVolume) string
		f3   getPVWantedVolumeID
		f4   getPVWantedVolumeID
	}{
		{
			name: "aws",
			s:    &AWSSnapshotter{},
			pv: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-1",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{},
				},
			},
			f1: func(pv *corev1.PersistentVolume) {
				pv.Spec.AWSElasticBlockStore = &corev1.AWSElasticBlockStoreVolumeSource{}
			},
			f2: func(pv *corev1.PersistentVolume) (want string) {
				pv.Spec.AWSElasticBlockStore.VolumeID = "foo"
				want = ""
				return
			},
			f3: func(pv *corev1.PersistentVolume) (want string, wantErr bool) {
				pv.Spec.AWSElasticBlockStore.VolumeID = "aws://us-east-1c/vol-abc123"
				want = "vol-abc123"
				wantErr = false
				return
			},
			f4: func(pv *corev1.PersistentVolume) (want string, wantErr bool) {
				pv.Spec.AWSElasticBlockStore.VolumeID = "vol-abc123"
				want = "vol-abc123"
				wantErr = false
				return
			},
		},
		{
			name: "gcp",
			s:    &GCPSnapshotter{},
			pv: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-2",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{},
				},
			},
			f1: func(pv *corev1.PersistentVolume) {
				pv.Spec.GCEPersistentDisk = &corev1.GCEPersistentDiskVolumeSource{}
			},
			f2: func(pv *corev1.PersistentVolume) (want string) {
				pv.Spec.GCEPersistentDisk.PDName = "abc123"
				want = "abc123"
				return
			},
			f3: func(pv *corev1.PersistentVolume) (want string, wantErr bool) {
				pv.Spec.GCEPersistentDisk.PDName = ""
				want = ""
				wantErr = true
				return
			},
			f4: func(pv *corev1.PersistentVolume) (want string, wantErr bool) {
				pv.Spec.GCEPersistentDisk.PDName = ""
				want = ""
				wantErr = true
				return
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.Init(nil, nil)

			// missing spec.awsElasticBlockStore/gcePersistentDisk -> no error
			volumeID, err := tt.s.GetVolumeID(tt.pv)
			require.NoError(t, err)
			assert.Equal(t, "", volumeID)

			// missing spec.awsElasticBlockStore.volumeID/gcePersistentDisk.pdName -> error
			tt.f1(tt.pv)
			volumeID, err = tt.s.GetVolumeID(tt.pv)
			assert.Error(t, err)
			assert.Equal(t, "", volumeID)

			// aws regex miss but gcp normal
			want := tt.f2(tt.pv)
			volumeID, err = tt.s.GetVolumeID(tt.pv)
			assert.NoError(t, err)
			assert.Equal(t, want, volumeID)

			// aws regex match 1 but gcp do nothing
			want, wantErr := tt.f3(tt.pv)
			volumeID, err = tt.s.GetVolumeID(tt.pv)
			if wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, want, volumeID)

			// aws regex match 2 but gcp do nothing
			want, wantErr = tt.f4(tt.pv)
			volumeID, err = tt.s.GetVolumeID(tt.pv)
			if wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, want, volumeID)
		})
	}
}

func TestGetVolumeIDForCSI(t *testing.T) {
	sAWS := &AWSSnapshotter{}
	sAWS.Init(nil, nil)
	sGCP := &GCPSnapshotter{}
	sGCP.Init(nil, nil)

	cases := []struct {
		name    string
		s       Snapshotter
		csiPV   *corev1.PersistentVolume
		want    string
		wantErr bool
	}{
		{
			name: "aws csi driver",
			s:    sAWS,
			csiPV: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-1",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							Driver:       "ebs.csi.aws.com",
							VolumeHandle: "vol-0866e1c99bd130a2c",
							FSType:       "ext4",
						},
					},
				},
			},
			want:    "vol-0866e1c99bd130a2c",
			wantErr: false,
		},
		{
			name: "unknown csi driver",
			s:    sAWS,
			csiPV: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-2",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							Driver:       "unknown.drv.com",
							VolumeHandle: "vol-0866e1c99bd130a2c",
							FSType:       "ext4",
						},
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "gke csi driver",
			s:    sGCP,
			csiPV: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-3",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							Driver: "pd.csi.storage.gke.io",
							VolumeAttributes: map[string]string{
								"storage.kubernetes.io/csiProvisionerIdentity": "1637243273131-8081-pd.csi.storage.gke.io",
							},
							VolumeHandle: "projects/test-gcp/zones/us-central1-f/disks/pvc-a970184f-6cc1-4769-85ad-61dcaf8bf51d",
							FSType:       "ext4",
						},
					},
				},
			},
			want:    "pvc-a970184f-6cc1-4769-85ad-61dcaf8bf51d",
			wantErr: false,
		},
		{
			name: "gke csi driver with invalid handle name",
			s:    sGCP,
			csiPV: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-4",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							Driver:       "pd.csi.storage.gke.io",
							VolumeHandle: "pvc-a970184f-6cc1-4769-85ad-61dcaf8bf51d",
							FSType:       "ext4",
						},
					},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "unknown driver",
			s:    sGCP,
			csiPV: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-5",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							Driver:       "xxx.csi.storage.gke.io",
							VolumeHandle: "pvc-a970184f-6cc1-4769-85ad-61dcaf8bf51d",
							FSType:       "ext4",
						},
					},
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			volumeID, err := tt.s.GetVolumeID(tt.csiPV)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, volumeID)
		})
	}
}

func TestPrepareCSBK8SMeta(t *testing.T) {
	b := &BaseSnapshotter{}
	csb := &CloudSnapBackup{
		Kubernetes: &KubernetesBackup{
			PVs:          []*corev1.PersistentVolume{},
			PVCs:         []*corev1.PersistentVolumeClaim{},
			TiDBCluster:  &v1alpha1.TidbCluster{},
			Unstructured: nil,
		},
	}
	helper := newHelper(t)
	defer helper.Close()
	b.deps = helper.Deps
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
		},
	}
	_, _, err := b.PrepareCSBK8SMeta(csb, tc)
	assert.NoError(t, err)
}

func TestPrepareCSBStoresMeta(t *testing.T) {
	sAWS := &AWSSnapshotter{}
	sAWS.Init(nil, nil)

	tc, pods, pvcs, pvs := constructTidbClusterWithSpecTiKV()

	b := NewBackupStoresMixture(tc, pvcs, pvs, sAWS)
	b.collectVolumesInfo()

	volsMapWanted := map[string]string{
		"tikv":         "/var/lib/tikv",
		"tikv-add-vol": "/test/raft-engine",
	}
	require.Equal(t, volsMapWanted, b.volsMap)

	mpTypeMapWanted := map[string]string{
		"/var/lib/tikv":     "storage.data-dir",
		"/test/raft-engine": "raft-engine.dir",
	}
	require.Equal(t, mpTypeMapWanted, b.mpTypeMap)

	csb := NewCloudSnapshotBackup(tc)
	_, err := b.PrepareCSBStoresMeta(csb, pods)
	require.NoError(t, err)

	storesWanted := []*StoresBackup{
		{
			StoreID: 1,
			Volumes: []*VolumeBackup{
				{
					VolumeID:  "vol-0e444aca5b73faaa0",
					Type:      "storage.data-dir",
					MountPath: "/var/lib/tikv",
				},
				{
					VolumeID:  "vol-0e444aca5b73fbbb0",
					Type:      "raft-engine.dir",
					MountPath: "/test/raft-engine",
				},
			},
		},
		{
			StoreID: 2,
			Volumes: []*VolumeBackup{
				{
					VolumeID:  "vol-0e444aca5b73faaa1",
					Type:      "storage.data-dir",
					MountPath: "/var/lib/tikv",
				},
				{
					VolumeID:  "vol-0e444aca5b73fbbb1",
					Type:      "raft-engine.dir",
					MountPath: "/test/raft-engine",
				},
			},
		},
		{
			StoreID: 3,
			Volumes: []*VolumeBackup{
				{
					VolumeID:  "vol-0e444aca5b73faaa2",
					Type:      "storage.data-dir",
					MountPath: "/var/lib/tikv",
				},
				{
					VolumeID:  "vol-0e444aca5b73fbbb2",
					Type:      "raft-engine.dir",
					MountPath: "/test/raft-engine",
				},
			},
		},
	}
	changelog, err := diff.Diff(storesWanted, csb.TiKV.Stores)
	assert.NoError(t, err)
	assert.Len(t, changelog, 0)

	vols := []*VolumeBackup{}
	for _, store := range storesWanted {
		vols = append(vols, store.Volumes...)
	}
	volIDs := []string{}
	for _, vol := range vols {
		volIDs = append(volIDs, vol.VolumeID)
	}
	csb.Kubernetes.PVs = pvs
	for _, pv := range csb.Kubernetes.PVs {
		require.NotNil(t, pv.Annotations[constants.AnnTemporaryVolumeID])
		assert.Contains(t, volIDs, pv.Annotations[constants.AnnTemporaryVolumeID])
	}
}

func TestGenerateBackupMetadata(t *testing.T) {
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps

	tc := &v1alpha1.TidbCluster{
		Spec: v1alpha1.TidbClusterSpec{
			TiKV: &v1alpha1.TiKVSpec{
				Replicas: 3,
			},
			PD: &v1alpha1.PDSpec{
				Replicas: 3,
			},
			TiDB: &v1alpha1.TiDBSpec{
				Replicas: 2,
			},
		},
	}

	cases := []struct {
		name      string
		backup    *v1alpha1.Backup
		wantSSNil bool
		wantErr   bool
	}{
		{
			name: "test-aws",
			backup: &v1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: make(map[string]string),
				},
				Spec: v1alpha1.BackupSpec{
					Type: v1alpha1.BackupTypeFull,
					Mode: v1alpha1.BackupModeVolumeSnapshot,
				},
			},
			wantSSNil: false,
			wantErr:   false,
		},
		{
			name: "test-noncloud-or-origin",
			backup: &v1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: make(map[string]string),
				},
				Spec: v1alpha1.BackupSpec{
					Type: v1alpha1.BackupTypeFull,
					Mode: v1alpha1.BackupModeVolumeSnapshot,
				},
			},
			wantSSNil: true,
			wantErr:   false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, _, err := NewSnapshotterForBackup(tt.backup.Spec.Mode, deps)
			require.NoError(t, err)
			_, _, err = s.GenerateBackupMetadata(tt.backup, tc)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.NotNil(t, tt.backup.Annotations[label.AnnBackupCloudSnapKey])
		})
	}
}

func constructTidbClusterWithSpecTiKV() (
	tc *v1alpha1.TidbCluster,
	pods []*corev1.Pod,
	pvcs []*corev1.PersistentVolumeClaim,
	pvs []*corev1.PersistentVolume) {
	tc = &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-db",
		},
		Spec: v1alpha1.TidbClusterSpec{
			TiKV: &v1alpha1.TiKVSpec{
				Replicas: 3,
				StorageVolumes: []v1alpha1.StorageVolume{
					{
						Name:      "add-vol",
						MountPath: "/test/raft-engine",
					},
				},
				Config: &v1alpha1.TiKVConfigWraper{
					GenericConfig: &config.GenericConfig{
						MP: map[string]interface{}{
							"raft-engine.dir": "/test/raft-engine",
						},
					},
				},
			},
			PD: &v1alpha1.PDSpec{
				Replicas: 3,
			},
			TiDB: &v1alpha1.TiDBSpec{
				Replicas: 2,
			},
		},
	}

	for i := 0; i < 3; i++ {
		pods = append(pods, constructPods(i)...)
		pvcs = append(pvcs, constructPVCs(i)...)
		pvs = append(pvs, constructPVs(i)...)
	}

	return
}

func constructPods(ordinal int) (pods []*corev1.Pod) {
	pods = append(pods, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-db-tikv-" + strconv.Itoa(ordinal),
			Labels: map[string]string{
				label.ComponentLabelKey: label.TiKVLabelVal,
				label.StoreIDLabelKey:   strconv.Itoa(ordinal + 1),
			},
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: label.TiKVLabelVal,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "tikv-test-db-tikv-" + strconv.Itoa(ordinal),
						},
					},
				},
				{
					Name: label.TiKVLabelVal + "-add-vol",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "tikv-add-vol-test-db-tikv-" + strconv.Itoa(ordinal),
						},
					},
				},
			},
		},
	})
	return
}

func constructPVs(ordinal int) (pvs []*corev1.PersistentVolume) {
	pvs = append(pvs, &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pv-test-aaa" + strconv.Itoa(ordinal),
			Labels: map[string]string{
				label.ComponentLabelKey: label.TiKVLabelVal,
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver:       "ebs.csi.aws.com",
					VolumeHandle: "vol-0e444aca5b73faaa" + strconv.Itoa(ordinal),
				},
			},
		},
	})
	pvs = append(pvs, &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pv-test-bbb" + strconv.Itoa(ordinal),
			Labels: map[string]string{
				label.ComponentLabelKey: label.TiKVLabelVal,
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver:       "ebs.csi.aws.com",
					VolumeHandle: "vol-0e444aca5b73fbbb" + strconv.Itoa(ordinal),
				},
			},
		},
	})
	return
}

func constructPVCs(ordinal int) (pvcs []*corev1.PersistentVolumeClaim) {
	pvcs = append(pvcs, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tikv-test-db-tikv-" + strconv.Itoa(ordinal),
			Labels: map[string]string{
				label.ComponentLabelKey: label.TiKVLabelVal,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "pv-test-aaa" + strconv.Itoa(ordinal),
		},
	})
	pvcs = append(pvcs, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tikv-add-vol-test-db-tikv-" + strconv.Itoa(ordinal),
			Labels: map[string]string{
				label.ComponentLabelKey: label.TiKVLabelVal,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "pv-test-bbb" + strconv.Itoa(ordinal),
		},
	})
	return
}

type helper struct {
	testutils.Helper
}

func newHelper(t *testing.T) *helper {
	h := testutils.NewHelper(t)
	return &helper{*h}
}

func TestSetVolumeID(t *testing.T) {
	type setPVWantedFunc func(pv *corev1.PersistentVolume) (want string)

	cases := []struct {
		name               string
		s                  Snapshotter
		volName            string
		pv                 *corev1.PersistentVolume
		funcs              []setPVWantedFunc
		testPVVolumeWanted func(t *testing.T, pv *corev1.PersistentVolume, wanted string)
	}{
		{
			name:    "aws",
			s:       &AWSSnapshotter{},
			volName: "vol-updated",
			pv: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-1",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{},
				},
			},
			funcs: []setPVWantedFunc{
				func(pv *corev1.PersistentVolume) (want string) {
					pv.Spec.AWSElasticBlockStore = &corev1.AWSElasticBlockStoreVolumeSource{}
					want = "vol-updated"
					return
				},
				func(pv *corev1.PersistentVolume) (want string) {
					pv.Labels = map[string]string{
						"failure-domain.beta.kubernetes.io/zone": "us-east-1a",
					}
					want = "aws://us-east-1a/vol-updated"
					return
				},
			},
			testPVVolumeWanted: func(t *testing.T, pv *corev1.PersistentVolume, wanted string) {
				require.NotNil(t, pv.Spec.AWSElasticBlockStore)
				assert.Equal(t, wanted, pv.Spec.AWSElasticBlockStore.VolumeID)
			},
		},
		{
			name:    "gcp",
			s:       &GCPSnapshotter{},
			volName: "abc123",
			pv: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-2",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{},
				},
			},
			funcs: []setPVWantedFunc{
				func(pv *corev1.PersistentVolume) (want string) {
					pv.Spec.GCEPersistentDisk = &corev1.GCEPersistentDiskVolumeSource{
						PDName: "abc123",
					}
					want = "abc123"
					return
				},
			},
			testPVVolumeWanted: func(t *testing.T, pv *corev1.PersistentVolume, wanted string) {
				require.NotNil(t, pv.Spec.GCEPersistentDisk)
				assert.Equal(t, wanted, pv.Spec.GCEPersistentDisk.PDName)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.Init(nil, nil)

			// missing spec.awsElasticBlockStore/gcePersistentDisk -> error
			err := tt.s.SetVolumeID(tt.pv, tt.volName)
			require.Error(t, err)

			// happy path
			for _, fSetPVWanted := range tt.funcs {
				wanted := fSetPVWanted(tt.pv)
				err := tt.s.SetVolumeID(tt.pv, tt.volName)
				require.NoError(t, err)
				tt.testPVVolumeWanted(t, tt.pv, wanted)
			}
		})
	}
}

func TestSetVolumeIDForCSI(t *testing.T) {
	sAWS := &AWSSnapshotter{}
	sAWS.Init(nil, nil)
	sGCP := &GCPSnapshotter{}
	sGCP.Init(nil, nil)

	cases := []struct {
		name     string
		s        Snapshotter
		csiPV    *corev1.PersistentVolume
		volumeID string
		wantErr  bool
	}{
		{
			name: "set ID to CSI with aws EBS CSI driver",
			s:    sAWS,
			csiPV: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-1",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							Driver:       "ebs.csi.aws.com",
							VolumeHandle: "vol-0866e1c99bd130a2c",
							FSType:       "ext4",
						},
					},
				},
			},
			volumeID: "vol-abcd",
			wantErr:  false,
		},
		{
			name: "set ID to CSI with EFS CSI driver",
			s:    sAWS,
			csiPV: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-2",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							Driver: "efs.csi.aws.com",
							FSType: "ext4",
						},
					},
				},
			},
			volumeID: "vol-abcd",
			wantErr:  true,
		},
		{
			name: "set ID to CSI with GKE pd CSI driver",
			s:    sGCP,
			csiPV: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-3",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							Driver:       "pd.csi.storage.gke.io",
							VolumeHandle: "projects/test-gcp/zones/us-central1-f/disks/pvc-a970184f-6cc1-4769-85ad-61dcaf8bf51d",
							FSType:       "ext4",
						},
					},
				},
			},
			volumeID: "restore-fd9729b5-868b-4544-9568-1c5d9121dabc",
			wantErr:  false,
		},
		{
			name: "set ID to CSI with GKE pd CSI driver, but the volumeHandle is invalid",
			s:    sGCP,
			csiPV: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-4",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							Driver:       "pd.csi.storage.gke.io",
							VolumeHandle: "pvc-a970184f-6cc1-4769-85ad-61dcaf8bf51d",
							FSType:       "ext4",
						},
					},
				},
			},
			volumeID: "restore-fd9729b5-868b-4544-9568-1c5d9121dabc",
			wantErr:  true,
		},
		{
			name: "set ID to CSI with unknown driver",
			s:    sGCP,
			csiPV: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv-5",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							Driver:       "xxx.csi.storage.gke.io",
							VolumeHandle: "projects/test-gcp/zones/us-central1-f/disks/pvc-a970184f-6cc1-4769-85ad-61dcaf8bf51d",
							FSType:       "ext4",
						},
					},
				},
			},
			volumeID: "restore-fd9729b5-868b-4544-9568-1c5d9121dabc",
			wantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.SetVolumeID(tt.csiPV, tt.volumeID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			// happy path
			require.NoError(t, err)
			newPV := tt.csiPV.DeepCopy()
			if _, ok := tt.s.(*GCPSnapshotter); ok {
				orilVolHandle := tt.csiPV.Spec.CSI.VolumeHandle
				ind := strings.LastIndex(newPV.Spec.CSI.VolumeHandle, "/")
				assert.Equal(t, tt.volumeID, newPV.Spec.CSI.VolumeHandle[ind+1:])
				assert.Equal(t, orilVolHandle[:ind], newPV.Spec.CSI.VolumeHandle[:ind])
			} else {
				assert.Equal(t, tt.volumeID, newPV.Spec.CSI.VolumeHandle)
			}
		})
	}
}

func TestPrepareRestoreMetadata(t *testing.T) {
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps

	restore := &v1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: v1alpha1.RestoreSpec{
			Type: v1alpha1.BackupTypeFull,
			Mode: v1alpha1.RestoreModeVolumeSnapshot,
			BR: &v1alpha1.BRConfig{
				Cluster:          "test",
				ClusterNamespace: "test",
			},
		},
	}

	s, _, err := NewSnapshotterForRestore(restore.Spec.Mode, deps)
	require.NoError(t, err)

	// missing .annotation["tidb.pingcap.com/backup-cloud-snapshot"] as metadata
	reason, err := s.PrepareRestoreMetadata(restore, &CloudSnapBackup{})
	require.NotEmpty(t, reason)
	require.Error(t, err)

	meta := testutils.ConstructRestoreMetaStr()
	csb := &CloudSnapBackup{}
	_ = json.Unmarshal([]byte(meta), csb)
	// happy path
	reason, err = s.PrepareRestoreMetadata(restore, csb)
	require.Empty(t, reason)
	require.NoError(t, err)
}

func TestProcessCSBPVCsAndPVs(t *testing.T) {
	sAWS := &AWSSnapshotter{}
	err := sAWS.Init(nil, nil)
	require.NoError(t, err)

	csb := &CloudSnapBackup{
		TiKV: &TiKVBackup{
			Stores: []*StoresBackup{
				{
					StoreID: 1,
					Volumes: []*VolumeBackup{
						{
							VolumeID:        "vol-0e65f40961a9f6244",
							SnapshotID:      "snap-1234567890abcdef0",
							RestoreVolumeID: "vol-0e65f40961a9f0001",
						},
					},
				},
				{
					StoreID: 2,
					Volumes: []*VolumeBackup{
						{
							VolumeID:        "vol-0e65f40961a9f6245",
							SnapshotID:      "snap-1234567890abcdef1",
							RestoreVolumeID: "vol-0e65f40961a9f0002",
						},
					},
				},
				{
					StoreID: 3,
					Volumes: []*VolumeBackup{
						{
							VolumeID:        "vol-0e65f40961a9f6246",
							SnapshotID:      "snap-1234567890abcdef2",
							RestoreVolumeID: "vol-0e65f40961a9f0003",
						},
					},
				},
			},
		},
		Kubernetes: &KubernetesBackup{
			PVs: []*corev1.PersistentVolume{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pv-1",
						Labels: map[string]string{
							"test/label": "retained",
						},
						Annotations: map[string]string{
							constants.KubeAnnDynamicallyProvisioned: "ebs.csi.aws.com",
							constants.AnnTemporaryVolumeID:          "vol-0e65f40961a9f6244",
							"test/annotation":                       "retained",
						},
						UID:             "301b0e8b-3538-4f61-a0fd-a25abd9a3122",
						ResourceVersion: "1958",
						Finalizers: []string{
							"kubernetes.io/pv-protection",
						},
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeSource: corev1.PersistentVolumeSource{
							CSI: &corev1.CSIPersistentVolumeSource{
								Driver:       "ebs.csi.aws.com",
								VolumeHandle: "vol-0e65f40961a9f6244",
								FSType:       "ext4",
							},
						},
						ClaimRef: &corev1.ObjectReference{
							Name:            "pvc-1",
							UID:             "301b0e8b-3538-4f61-a0fd-a25abd9a3121",
							ResourceVersion: "1957",
						},
					},
					Status: corev1.PersistentVolumeStatus{
						Phase: corev1.VolumeBound,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pv-2",
						Labels: map[string]string{
							"test/label": "retained",
						},
						Annotations: map[string]string{
							constants.KubeAnnDynamicallyProvisioned: "ebs.csi.aws.com",
							constants.AnnTemporaryVolumeID:          "vol-0e65f40961a9f6245",
							"test/annotation":                       "retained",
						},
						UID:             "301b0e8b-3538-4f61-a0fd-a25abd9a3124",
						ResourceVersion: "1960",
						Finalizers: []string{
							"kubernetes.io/pv-protection",
						},
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeSource: corev1.PersistentVolumeSource{
							CSI: &corev1.CSIPersistentVolumeSource{
								Driver:       "ebs.csi.aws.com",
								VolumeHandle: "vol-0e65f40961a9f6245",
								FSType:       "ext4",
							},
						},
						ClaimRef: &corev1.ObjectReference{
							Name:            "pvc-2",
							UID:             "301b0e8b-3538-4f61-a0fd-a25abd9a3123",
							ResourceVersion: "1959",
						},
					},
					Status: corev1.PersistentVolumeStatus{
						Phase: corev1.VolumeBound,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pv-3",
						Labels: map[string]string{
							"test/label": "retained",
						},
						Annotations: map[string]string{
							constants.KubeAnnDynamicallyProvisioned: "ebs.csi.aws.com",
							constants.AnnTemporaryVolumeID:          "vol-0e65f40961a9f6246",
							"test/annotation":                       "retained",
						},
						UID:             "301b0e8b-3538-4f61-a0fd-a25abd9a3126",
						ResourceVersion: "1962",
						Finalizers: []string{
							"kubernetes.io/pv-protection",
						},
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeSource: corev1.PersistentVolumeSource{
							CSI: &corev1.CSIPersistentVolumeSource{
								Driver:       "ebs.csi.aws.com",
								VolumeHandle: "vol-0e65f40961a9f6246",
								FSType:       "ext4",
							},
						},
						ClaimRef: &corev1.ObjectReference{
							Name:            "pvc-3",
							UID:             "301b0e8b-3538-4f61-a0fd-a25abd9a3125",
							ResourceVersion: "1961",
						},
					},
					Status: corev1.PersistentVolumeStatus{
						Phase: corev1.VolumeBound,
					},
				},
			},
			PVCs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pvc-1",
						Namespace: "default",
						Labels: map[string]string{
							"test/label": "retained",
						},
						Annotations: map[string]string{
							constants.KubeAnnBindCompleted:     "yes",
							constants.KubeAnnBoundByController: "yes",
							"test/annotation":                  "retained",
						},
						UID:             "301b0e8b-3538-4f61-a0fd-a25abd9a3121",
						ResourceVersion: "1957",
						Finalizers: []string{
							"kubernetes.io/pvc-protection",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pv-1",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pvc-2",
						Namespace: "default",
						Labels: map[string]string{
							"test/label": "retained",
						},
						Annotations: map[string]string{
							constants.KubeAnnBindCompleted:     "yes",
							constants.KubeAnnBoundByController: "yes",
							"test/annotation":                  "retained",
						},
						UID:             "301b0e8b-3538-4f61-a0fd-a25abd9a3123",
						ResourceVersion: "1959",
						Finalizers: []string{
							"kubernetes.io/pvc-protection",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pv-2",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pvc-3",
						Namespace: "default",
						Labels: map[string]string{
							"test/label": "retained",
						},
						Annotations: map[string]string{
							constants.KubeAnnBindCompleted:     "yes",
							constants.KubeAnnBoundByController: "yes",
							"test/annotation":                  "retained",
						},
						UID:             "301b0e8b-3538-4f61-a0fd-a25abd9a3125",
						ResourceVersion: "1961",
						Finalizers: []string{
							"kubernetes.io/pvc-protection",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pv-3",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			TiDBCluster: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			},
			Unstructured: nil,
		},
	}

	restore := &v1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "restore-1",
			Namespace: "default",
		},
		Spec: v1alpha1.RestoreSpec{
			Type: v1alpha1.BackupTypeFull,
			Mode: v1alpha1.RestoreModeVolumeSnapshot,
			BR: &v1alpha1.BRConfig{
				Cluster:          "test",
				ClusterNamespace: "default",
			},
		},
	}

	m := NewRestoreStoresMixture(sAWS)
	reason, err := m.ProcessCSBPVCsAndPVs(restore, csb)
	require.Empty(t, reason)
	require.NoError(t, err)

	// happy path for backup-volumeID mapping restore-volumeID
	volIDMapWanted := map[string]string{
		"vol-0e65f40961a9f6244": "vol-0e65f40961a9f0001",
		"vol-0e65f40961a9f6245": "vol-0e65f40961a9f0002",
		"vol-0e65f40961a9f6246": "vol-0e65f40961a9f0003",
	}
	require.Equal(t, volIDMapWanted, m.rsVolIDMap)

	// happy path for reformed PVs as the reborn resource
	pvsWanted := []*corev1.PersistentVolume{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv-1",
				Labels: map[string]string{
					"test/label": "retained",
				},
				Annotations: map[string]string{
					"test/annotation": "retained",
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					CSI: &corev1.CSIPersistentVolumeSource{
						Driver:       "ebs.csi.aws.com",
						VolumeHandle: "vol-0e65f40961a9f0001",
						FSType:       "ext4",
					},
				},
				ClaimRef: &corev1.ObjectReference{
					Name: "pvc-1",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv-2",
				Labels: map[string]string{
					"test/label": "retained",
				},
				Annotations: map[string]string{
					"test/annotation": "retained",
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					CSI: &corev1.CSIPersistentVolumeSource{
						Driver:       "ebs.csi.aws.com",
						VolumeHandle: "vol-0e65f40961a9f0002",
						FSType:       "ext4",
					},
				},
				ClaimRef: &corev1.ObjectReference{
					Name: "pvc-2",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv-3",
				Labels: map[string]string{
					"test/label": "retained",
				},
				Annotations: map[string]string{
					"test/annotation": "retained",
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					CSI: &corev1.CSIPersistentVolumeSource{
						Driver:       "ebs.csi.aws.com",
						VolumeHandle: "vol-0e65f40961a9f0003",
						FSType:       "ext4",
					},
				},
				ClaimRef: &corev1.ObjectReference{
					Name: "pvc-3",
				},
			},
		},
	}
	assert.Equal(t, pvsWanted, csb.Kubernetes.PVs)

	// happy path for reformed PVCs as the reborn resource
	pvcsWanted := []*corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pvc-1",
				Namespace: "default",
				Labels: map[string]string{
					"test/label": "retained",
				},
				Annotations: map[string]string{
					"test/annotation": "retained",
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "pv-1",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pvc-2",
				Namespace: "default",
				Labels: map[string]string{
					"test/label": "retained",
				},
				Annotations: map[string]string{
					"test/annotation": "retained",
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "pv-2",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pvc-3",
				Namespace: "default",
				Labels: map[string]string{
					"test/label": "retained",
				},
				Annotations: map[string]string{
					"test/annotation": "retained",
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "pv-3",
			},
		},
	}
	assert.Equal(t, pvcsWanted, csb.Kubernetes.PVCs)
}
