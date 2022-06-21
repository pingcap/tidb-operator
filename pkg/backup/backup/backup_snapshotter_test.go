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
	"testing"

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

}

func TestPrepareCSBStoresMeta(t *testing.T) {

}

func TestPrepareBackupMetadata(t *testing.T) {

}
