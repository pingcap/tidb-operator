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
	"strconv"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"github.com/pingcap/tidb-operator/pkg/backup/testutils"
	"github.com/r3labs/diff/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetVolumeID(t *testing.T) {
	cases := []struct {
		name string
		s    Snapshotter
		pv   *corev1.PersistentVolume
		f1   func(pv *corev1.PersistentVolume)
		f2   func(pv *corev1.PersistentVolume) string
		f3   func(pv *corev1.PersistentVolume) (string, bool)
		f4   func(pv *corev1.PersistentVolume) (string, bool)
	}{
		{
			name: "AWS",
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
			name: "GCP",
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
							VolumeHandle: "projects/velero-gcp/zones/us-central1-f/disks/pvc-a970184f-6cc1-4769-85ad-61dcaf8bf51d",
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
	_, _, err := b.PrepareCSBK8SMeta(csb, "test-ns")
	assert.NoError(t, err)
}

func TestPrepareCSBStoresMeta(t *testing.T) {
	sAWS := &AWSSnapshotter{}
	sAWS.Init(nil, nil)

	tc, pods, pvcs, pvs := constructTidbClusterWithSpecTiKV()

	b := NewStoresMixture(tc, pvcs, pvs, sAWS)
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
}

func TestPrepareBackupMetadata(t *testing.T) {
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
					Type: "ebs",
				},
			},
			wantSSNil: false,
			wantErr:   false,
		},
		{
			name: "test-gcp",
			backup: &v1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: make(map[string]string),
				},
				Spec: v1alpha1.BackupSpec{
					Type: "gcepd",
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
					Type: "full",
				},
			},
			wantSSNil: true,
			wantErr:   false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cpFactory := &CloudProviderFactory{}
			s := cpFactory.CreateSnapshotter(tt.backup.Spec.Type)
			if tt.wantSSNil {
				assert.Nil(t, s)
				return
			} else {
				assert.NotNil(t, s)
			}
			s.Init(deps, nil)
			_, err := s.PrepareBackupMetadata(tt.backup, tc, "test-ns")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NotNil(t, tt.backup.Annotations[label.AnnBackupCloudSnapKey])
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
		pods = append(pods, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-db-tikv-" + strconv.Itoa(i),
				Labels: map[string]string{
					label.ComponentLabelKey: label.TiKVLabelVal,
					label.StoreIDLabelKey:   strconv.Itoa(i + 1),
				},
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: label.TiKVLabelVal,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "tikv-test-db-tikv-" + strconv.Itoa(i),
							},
						},
					},
					{
						Name: label.TiKVLabelVal + "-add-vol",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "tikv-add-vol-test-db-tikv-" + strconv.Itoa(i),
							},
						},
					},
				},
			},
		})

		pvcs = append(pvcs, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tikv-test-db-tikv-" + strconv.Itoa(i),
				Labels: map[string]string{
					label.ComponentLabelKey: label.TiKVLabelVal,
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "pv-test-aaa" + strconv.Itoa(i),
			},
		})
		pvcs = append(pvcs, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tikv-add-vol-test-db-tikv-" + strconv.Itoa(i),
				Labels: map[string]string{
					label.ComponentLabelKey: label.TiKVLabelVal,
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "pv-test-bbb" + strconv.Itoa(i),
			},
		})

		pvs = append(pvs, &corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv-test-aaa" + strconv.Itoa(i),
				Labels: map[string]string{
					label.ComponentLabelKey: label.TiKVLabelVal,
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					CSI: &corev1.CSIPersistentVolumeSource{
						Driver:       "ebs.csi.aws.com",
						VolumeHandle: "vol-0e444aca5b73faaa" + strconv.Itoa(i),
					},
				},
			},
		})
		pvs = append(pvs, &corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv-test-bbb" + strconv.Itoa(i),
				Labels: map[string]string{
					label.ComponentLabelKey: label.TiKVLabelVal,
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					CSI: &corev1.CSIPersistentVolumeSource{
						Driver:       "ebs.csi.aws.com",
						VolumeHandle: "vol-0e444aca5b73fbbb" + strconv.Itoa(i),
					},
				},
			},
		})
	}

	return
}

type helper struct {
	testutils.Helper
}

func newHelper(t *testing.T) *helper {
	h := testutils.NewHelper(t)
	return &helper{*h}
}
