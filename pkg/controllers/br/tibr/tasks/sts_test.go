// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tasks

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	v1alpha1br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestAssembleSts(t *testing.T) {
	volume := v1alpha1.Volume{
		Name: "data",
		Mounts: []v1alpha1.VolumeMount{
			{Type: "data", MountPath: v1alpha1br.VolumeTiBRDataDefaultMountPath},
		},
		Storage: resource.MustParse("10Gi"),
	}

	apiServerContainerOverlay := corev1.Container{Name: v1alpha1br.ContainerAPIServer, Image: "another-api-server-iamge"}
	autoBackupContainerOverlay := corev1.Container{Name: v1alpha1br.ContainerAutoBackup, Image: "another-auto-backup-iamge"}
	sidecar := corev1.Container{Name: "sidecar", Image: "busybox"}

	testCases := []struct {
		name                  string
		volumes               []v1alpha1.Volume
		overlay               *v1alpha1.Overlay
		expectPVCCount        int
		expectContainerNames  []string
		expectContainersImage map[string]string
	}{
		{
			name:                 "With Volumes definition",
			volumes:              []v1alpha1.Volume{volume},
			overlay:              nil,
			expectPVCCount:       1,
			expectContainerNames: []string{v1alpha1br.ContainerAPIServer, v1alpha1br.ContainerAutoBackup},
			expectContainersImage: map[string]string{
				v1alpha1br.ContainerAPIServer:  "default",
				v1alpha1br.ContainerAutoBackup: "default",
			},
		},
		{
			name:    "With Pod overlay",
			volumes: nil,
			overlay: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{
					Spec: &corev1.PodSpec{
						Containers: []corev1.Container{apiServerContainerOverlay, autoBackupContainerOverlay, sidecar},
					},
				},
			},
			expectPVCCount:       0,
			expectContainerNames: []string{v1alpha1br.ContainerAPIServer, v1alpha1br.ContainerAutoBackup, "sidecar"},
			expectContainersImage: map[string]string{
				v1alpha1br.ContainerAPIServer:  apiServerContainerOverlay.Image,
				v1alpha1br.ContainerAutoBackup: autoBackupContainerOverlay.Image,
				"sidecar":                      sidecar.Image,
			},
		},
		{
			name:    "With both Pod overlay and Volumes definition",
			volumes: []v1alpha1.Volume{volume},
			overlay: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{
					Spec: &corev1.PodSpec{
						Containers: []corev1.Container{apiServerContainerOverlay, autoBackupContainerOverlay, sidecar},
					},
				},
			},
			expectPVCCount:       1,
			expectContainerNames: []string{v1alpha1br.ContainerAPIServer, v1alpha1br.ContainerAutoBackup, "sidecar"},
			expectContainersImage: map[string]string{
				v1alpha1br.ContainerAPIServer:  apiServerContainerOverlay.Image,
				v1alpha1br.ContainerAutoBackup: autoBackupContainerOverlay.Image,
				"sidecar":                      sidecar.Image,
			},
		},
		{
			name:                 "Without Pod overlay and Volumes definition",
			volumes:              nil,
			overlay:              nil,
			expectPVCCount:       0,
			expectContainerNames: []string{v1alpha1br.ContainerAPIServer, v1alpha1br.ContainerAutoBackup},
			expectContainersImage: map[string]string{
				v1alpha1br.ContainerAPIServer:  "default",
				v1alpha1br.ContainerAutoBackup: "default",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tibr := &v1alpha1br.TiBR{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo",
					Namespace: "testing",
				},
				Spec: v1alpha1br.TiBRSpec{
					AutoSchedule: &v1alpha1br.TiBRAutoSchedule{
						Type: v1alpha1br.TiBRAutoScheduleTypePerMinute,
						At:   0,
					},
					Image:   ptr.To("default"),
					Config:  "[native-br]\nenable-lightweight-backup = true",
					Volumes: tc.volumes,
					Overlay: tc.overlay,
				},
			}

			cluster := &v1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "basic",
					Namespace: "testing",
				},
			}

			rtx := &ReconcileContext{
				cluster:        cluster,
				tibr:           tibr,
				namespacedName: types.NamespacedName{Name: "demo", Namespace: "testing"},
			}

			sts := assembleSts(rtx)

			assert.Equal(t, "demo-tibr-sts", sts.Name)
			assert.Equal(t, "testing", sts.Namespace)
			assert.Equal(t, "demo-tibr-headless", sts.Spec.ServiceName)
			assert.Equal(t, []metav1.OwnerReference{*metav1.NewControllerRef(rtx.TiBR(), v1alpha1br.SchemeGroupVersion.WithKind("TiBR"))}, sts.OwnerReferences)
			// labels
			assert.Equal(t, TiBRSubResourceLabels(tibr), sts.Labels)
			assert.Equal(t, TiBRSubResourceLabels(tibr), sts.Spec.Selector.MatchLabels)
			assert.Equal(t, TiBRSubResourceLabels(tibr), sts.Spec.Template.Labels)

			assert.Equal(t, tc.expectPVCCount, len(sts.Spec.VolumeClaimTemplates))

			var containerNames []string
			for _, c := range sts.Spec.Template.Spec.Containers {
				containerNames = append(containerNames, c.Name)
			}
			assert.ElementsMatch(t, tc.expectContainerNames, containerNames)

			for _, c := range sts.Spec.Template.Spec.Containers {
				assert.Equal(t, tc.expectContainersImage[c.Name], c.Image)
			}
		})
	}
}

func TestAssembleVolumeClaimIfNeeded(t *testing.T) {
	fakeStorageClass := "standard"
	fakeAttrClass := "ssd-fast"
	volumeName := "data"
	storage := resource.MustParse("1Gi")

	tests := []struct {
		name    string
		tibr    *v1alpha1br.TiBR
		wantPVC *corev1.PersistentVolumeClaim
	}{
		{
			name:    "no volumes",
			tibr:    &v1alpha1br.TiBR{},
			wantPVC: nil,
		},
		{
			name: "volume without overlay",
			tibr: &v1alpha1br.TiBR{
				Spec: v1alpha1br.TiBRSpec{
					Volumes: []v1alpha1.Volume{
						{
							Name:                      volumeName,
							Storage:                   storage,
							StorageClassName:          &fakeStorageClass,
							VolumeAttributesClassName: &fakeAttrClass,
						},
					},
				},
			},
			wantPVC: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: volumeName,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName:          &fakeStorageClass,
					VolumeAttributesClassName: &fakeAttrClass,
					AccessModes:               []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: storage,
						},
					},
				},
			},
		},
		{
			name: "volume with metadata overlay",
			tibr: &v1alpha1br.TiBR{
				Spec: v1alpha1br.TiBRSpec{
					Volumes: []v1alpha1.Volume{
						{
							Name:             volumeName,
							Storage:          storage,
							StorageClassName: &fakeStorageClass,
						},
					},
					Overlay: &v1alpha1.Overlay{
						PersistentVolumeClaims: []v1alpha1.NamedPersistentVolumeClaimOverlay{
							{
								Name: volumeName,
								PersistentVolumeClaim: v1alpha1.PersistentVolumeClaimOverlay{
									ObjectMeta: v1alpha1.ObjectMeta{
										Labels: map[string]string{"env": "test"},
									},
								},
							},
						},
					},
				},
			},
			wantPVC: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:   volumeName,
					Labels: map[string]string{"env": "test"},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: &fakeStorageClass,
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: storage,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assembleVolumeClaimIfNeeded(tt.tibr)

			if tt.wantPVC == nil {
				if got != nil {
					t.Errorf("expected nil PVC, got %v", got)
				}
				return
			}

			if got == nil {
				t.Errorf("expected PVC, got nil")
				return
			}

			if got.Name != tt.wantPVC.Name {
				t.Errorf("PVC name mismatch: got %s, want %s", got.Name, tt.wantPVC.Name)
			}

			if sc := got.Spec.StorageClassName; sc == nil || *sc != *tt.wantPVC.Spec.StorageClassName {
				t.Errorf("StorageClass mismatch: got %v, want %v", sc, tt.wantPVC.Spec.StorageClassName)
			}

			if vac := got.Spec.VolumeAttributesClassName; vac != nil && tt.wantPVC.Spec.VolumeAttributesClassName != nil && *vac != *tt.wantPVC.Spec.VolumeAttributesClassName {
				t.Errorf("VolumeAttributesClassName mismatch: got %v, want %v", *vac, *tt.wantPVC.Spec.VolumeAttributesClassName)
			}

			gotStorage := got.Spec.Resources.Requests[corev1.ResourceStorage]
			wantStorage := tt.wantPVC.Spec.Resources.Requests[corev1.ResourceStorage]
			if gotStorage.Cmp(wantStorage) != 0 {
				t.Errorf("Storage request mismatch: got %v, want %v", gotStorage.String(), wantStorage.String())
			}

			for k, v := range tt.wantPVC.Labels {
				if got.Labels[k] != v {
					t.Errorf("Label %q mismatch: got %q, want %q", k, got.Labels[k], v)
				}
			}
		})
	}
}

func TestAssemblePodSpec(t *testing.T) {
	testCases := []struct {
		name               string
		autoSchedule       bool
		expectedContainers []string
	}{
		{
			name:               "without autoschedule",
			autoSchedule:       false,
			expectedContainers: []string{v1alpha1br.ContainerAPIServer},
		},
		{
			name:               "with autoschedule",
			autoSchedule:       true,
			expectedContainers: []string{v1alpha1br.ContainerAPIServer, v1alpha1br.ContainerAutoBackup},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tibr := &v1alpha1br.TiBR{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo",
					Namespace: "testing",
				},
			}
			if tc.autoSchedule {
				tibr.Spec.AutoSchedule = &v1alpha1br.TiBRAutoSchedule{
					Type: v1alpha1br.TiBRAutoScheduleTypePerMinute,
				}
			}
			cluster := &v1alpha1.Cluster{
				Status: v1alpha1.ClusterStatus{
					PD: "pd-service:2379",
				},
			}
			rtx := &ReconcileContext{tibr: tibr, cluster: cluster}

			podSpec := assemblePodSpec(rtx)

			var containerNames []string
			for _, c := range podSpec.Containers {
				containerNames = append(containerNames, c.Name)
			}
			assert.ElementsMatch(t, tc.expectedContainers, containerNames)
		})
	}
}

func TestAssembleServerContainer(t *testing.T) {
	testCases := []struct {
		name             string
		tlsEnabled       bool
		dataVolume       *v1alpha1.Volume
		pdAddr           string
		expectedCmdParts []string
		expectedMounts   []string
	}{
		{
			name:       "without TLS and dataVolume",
			tlsEnabled: false,
			dataVolume: nil,
			pdAddr:     "pd.default.svc:2379",
			expectedCmdParts: []string{
				"/tikv-worker",
				"--config", ConfigMountPath + "/" + ConfigFileName,
				"--addr", fmt.Sprintf("0.0.0.0:%d", APIServerPort),
				"--pd-endpoints", "pd.default.svc:2379",
				"--data-dir", v1alpha1br.VolumeTiBRDataDefaultMountPath,
			},
			expectedMounts: []string{ConfigVolumeName},
		},
		{
			name:       "with TLS",
			tlsEnabled: true,
			dataVolume: nil,
			pdAddr:     "pd.secure.svc:2379",
			expectedCmdParts: append([]string{
				"/tikv-worker",
				"--config", ConfigMountPath + "/" + ConfigFileName,
				"--addr", fmt.Sprintf("0.0.0.0:%d", APIServerPort),
				"--pd-endpoints", "pd.secure.svc:2379",
				"--data-dir", v1alpha1br.VolumeTiBRDataDefaultMountPath,
			}, TLSCmdArgs...),
			expectedMounts: []string{ConfigVolumeName, TLSVolumeName},
		},
		{
			name:       "with dataVolume",
			tlsEnabled: false,
			dataVolume: &v1alpha1.Volume{
				Name: "data2",
				Mounts: []v1alpha1.VolumeMount{
					{MountPath: "dummyPath"},
				},
			},
			pdAddr: "pd.secure.svc:2379",
			expectedCmdParts: []string{
				"/tikv-worker",
				"--config", ConfigMountPath + "/" + ConfigFileName,
				"--addr", fmt.Sprintf("0.0.0.0:%d", APIServerPort),
				"--pd-endpoints", "pd.secure.svc:2379",
				"--data-dir", "dummyPath",
			},
			expectedMounts: []string{ConfigVolumeName, "data2"},
		},
		{
			name:       "with TLS and dataVolume",
			tlsEnabled: true,
			dataVolume: &v1alpha1.Volume{
				Name: "data2",
			},
			pdAddr: "pd.secure.svc:2379",
			expectedCmdParts: append([]string{
				"/tikv-worker",
				"--config", ConfigMountPath + "/" + ConfigFileName,
				"--addr", fmt.Sprintf("0.0.0.0:%d", APIServerPort),
				"--pd-endpoints", "pd.secure.svc:2379",
				"--data-dir", v1alpha1br.VolumeTiBRDataDefaultMountPath,
			}, TLSCmdArgs...),
			expectedMounts: []string{ConfigVolumeName, TLSVolumeName, "data2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rtx := &ReconcileContext{
				tibr: &v1alpha1br.TiBR{
					ObjectMeta: metav1.ObjectMeta{Name: "tibr"},
					Spec: v1alpha1br.TiBRSpec{
						Image: ptr.To("test-image"),
					},
				},
				cluster: &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						TLSCluster: &v1alpha1.TLSCluster{Enabled: tc.tlsEnabled},
					},
					Status: v1alpha1.ClusterStatus{PD: tc.pdAddr},
				},
			}
			if tc.dataVolume != nil {
				rtx.tibr.Spec.Volumes = []v1alpha1.Volume{*tc.dataVolume}
			}

			c := assembleServerContainer(rtx)

			assert.Equal(t, v1alpha1br.ContainerAPIServer, c.Name)
			assert.Equal(t, "test-image", c.Image)
			assert.Equal(t, tc.expectedCmdParts, c.Command)

			var mountNames []string
			for _, vm := range c.VolumeMounts {
				mountNames = append(mountNames, vm.Name)
			}
			assert.ElementsMatch(t, tc.expectedMounts, mountNames)
		})
	}
}

func TestGenUserDefinedVolumeMount(t *testing.T) {
	defaultName := v1alpha1br.VolumeTiBRData
	defaultPath := v1alpha1br.VolumeTiBRDataDefaultMountPath

	tests := []struct {
		name     string
		input    []v1alpha1.Volume
		expected *corev1.VolumeMount
	}{
		{
			name:     "empty volume list returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name: "default volume and mount path used",
			input: []v1alpha1.Volume{
				{},
			},
			expected: &corev1.VolumeMount{
				Name:      defaultName,
				MountPath: defaultPath,
			},
		},
		{
			name: "custom volume name",
			input: []v1alpha1.Volume{
				{
					Name: "custom-vol",
				},
			},
			expected: &corev1.VolumeMount{
				Name:      "custom-vol",
				MountPath: defaultPath,
			},
		},
		{
			name: "custom mount path",
			input: []v1alpha1.Volume{
				{
					Mounts: []v1alpha1.VolumeMount{
						{MountPath: "/custom/path"},
					},
				},
			},
			expected: &corev1.VolumeMount{
				Name:      defaultName,
				MountPath: "/custom/path",
			},
		},
		{
			name: "custom name and mount path",
			input: []v1alpha1.Volume{
				{
					Name: "data-vol",
					Mounts: []v1alpha1.VolumeMount{
						{MountPath: "/mnt/data"},
					},
				},
			},
			expected: &corev1.VolumeMount{
				Name:      "data-vol",
				MountPath: "/mnt/data",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := genUserDefinedVolumeMount(tt.input)
			if tt.expected == nil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Errorf("expected non-nil, got nil")
				return
			}
			if got.Name != tt.expected.Name {
				t.Errorf("volume mount name mismatch: got %q, want %q", got.Name, tt.expected.Name)
			}
			if got.MountPath != tt.expected.MountPath {
				t.Errorf("volume mount path mismatch: got %q, want %q", got.MountPath, tt.expected.MountPath)
			}
		})
	}
}

func TestAssembleAutoBackupContainer(t *testing.T) {
	testCases := []struct {
		name             string
		tlsEnabled       bool
		pdAddr           string
		scheduleType     v1alpha1br.TiBRAutoScheduleType
		expectedCmdParts []string
		expectedMounts   []string
	}{
		{
			name:         "without TLS - daily schedule",
			tlsEnabled:   false,
			pdAddr:       "pd.default.svc:2379",
			scheduleType: v1alpha1br.TiBRAutoScheduleTypePerDay,
			expectedCmdParts: []string{
				"/cse-ctl",
				"backup",
				"--lightweight",
				"--interval", "86400",
				"--tolerate-err", "1",
				"--pd", "pd.default.svc:2379",
			},
			expectedMounts: nil,
		},
		{
			name:         "with TLS - hourly schedule",
			tlsEnabled:   true,
			pdAddr:       "pd.secure.svc:2379",
			scheduleType: v1alpha1br.TiBRAutoScheduleTypePerHour,
			expectedCmdParts: append([]string{
				"/cse-ctl",
				"backup",
				"--lightweight",
				"--interval", "3600",
				"--tolerate-err", "1",
				"--pd", "pd.secure.svc:2379",
			}, TLSCmdArgs...),
			expectedMounts: []string{TLSVolumeName},
		},
		{
			name:         "with TLS - unknown schedule type (fallback to 60s)",
			tlsEnabled:   true,
			pdAddr:       "pd.secure.svc:2379",
			scheduleType: "unknown",
			expectedCmdParts: append([]string{
				"/cse-ctl",
				"backup",
				"--lightweight",
				"--interval", "60",
				"--tolerate-err", "1",
				"--pd", "pd.secure.svc:2379",
			}, TLSCmdArgs...),
			expectedMounts: []string{TLSVolumeName},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rtx := &ReconcileContext{
				tibr: &v1alpha1br.TiBR{
					ObjectMeta: metav1.ObjectMeta{Name: "tibr"},
					Spec: v1alpha1br.TiBRSpec{
						Image: ptr.To("test-image"),
						AutoSchedule: &v1alpha1br.TiBRAutoSchedule{
							Type: tc.scheduleType,
						},
					},
				},
				cluster: &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						TLSCluster: &v1alpha1.TLSCluster{Enabled: tc.tlsEnabled},
					},
					Status: v1alpha1.ClusterStatus{PD: tc.pdAddr},
				},
			}

			c := assembleAutoBackupContainer(rtx)

			assert.Equal(t, v1alpha1br.ContainerAutoBackup, c.Name)
			assert.Equal(t, "test-image", c.Image)
			assert.Equal(t, tc.expectedCmdParts, c.Command)

			var mountNames []string
			for _, vm := range c.VolumeMounts {
				mountNames = append(mountNames, vm.Name)
			}
			assert.ElementsMatch(t, tc.expectedMounts, mountNames)
		})
	}
}

func TestAssembleVolumes(t *testing.T) {
	testCases := []struct {
		name            string
		tlsEnabled      bool
		expectedVolumes []corev1.Volume
	}{
		{
			name:       "without TLS",
			tlsEnabled: false,
			expectedVolumes: []corev1.Volume{
				{
					Name: ConfigVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "tibr-tibr-config", // assuming ConfigMapName returns <name>-config
							},
						},
					},
				},
			},
		},
		{
			name:       "with TLS",
			tlsEnabled: true,
			expectedVolumes: []corev1.Volume{
				{
					Name: ConfigVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "tibr-tibr-config",
							},
						},
					},
				},
				{
					Name: TLSVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  "tibr-tibr-cluster-secret", // assuming SecretName returns <name>-secret
							DefaultMode: ptr.To(SecretAccessMode),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rtx := &ReconcileContext{
				tibr: &v1alpha1br.TiBR{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tibr",
						Namespace: "default",
					},
				},
				cluster: &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						TLSCluster: &v1alpha1.TLSCluster{
							Enabled: tc.tlsEnabled,
						},
					},
				},
			}

			actual := assembleVolumes(rtx)

			assert.Equal(t, len(tc.expectedVolumes), len(actual))
			for i, expectedVol := range tc.expectedVolumes {
				assert.Equal(t, expectedVol.Name, actual[i].Name)

				if expectedVol.ConfigMap != nil {
					assert.NotNil(t, actual[i].ConfigMap)
					assert.Equal(t, expectedVol.ConfigMap.Name, actual[i].ConfigMap.Name)
				}
				if expectedVol.Secret != nil {
					assert.NotNil(t, actual[i].Secret)
					assert.Equal(t, expectedVol.Secret.SecretName, actual[i].Secret.SecretName)
					assert.Equal(t, *expectedVol.Secret.DefaultMode, *actual[i].Secret.DefaultMode)
				}
			}
		})
	}
}
