// Copyright 2020 PingCAP, Inc.
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
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func newMockPVC(name, storageClass, storageRequest, capacity string) *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: v1.NamespaceDefault,
			Name:      name,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse(storageRequest),
				},
			},
			StorageClassName: pointer.StringPtr(storageClass),
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase: v1.ClaimBound,
			Capacity: v1.ResourceList{
				v1.ResourceStorage: resource.MustParse(capacity),
			},
		},
	}
}

func newFullPVC(name, component, storageClass, storageRequest, capacity, nameLabel, instance string) *v1.PersistentVolumeClaim {
	pvc := newMockPVC(name, storageClass, storageRequest, capacity)
	pvc.Labels = map[string]string{
		label.NameLabelKey:      nameLabel,
		label.ManagedByLabelKey: label.TiDBOperator,
		label.InstanceLabelKey:  instance,
		label.ComponentLabelKey: component,
	}
	return pvc
}

func newPVCWithStorage(name string, component string, storageClass, storageRequest string) *v1.PersistentVolumeClaim {
	return newFullPVC(name, component, storageClass, storageRequest, storageRequest, "tidb-cluster", "tc")
}

func newDMPVCWithStorage(name string, component string, storageClass, storageRequest string) *v1.PersistentVolumeClaim {
	return newFullPVC(name, component, storageClass, storageRequest, storageRequest, "dm-cluster", "dc")
}

func newStorageClass(name string, volumeExpansion bool) *storagev1.StorageClass {
	return &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		AllowVolumeExpansion: pointer.BoolPtr(volumeExpansion),
	}
}

func TestPVCResizer(t *testing.T) {
	tests := []struct {
		name     string
		tc       *v1alpha1.TidbCluster
		sc       *storagev1.StorageClass
		pvcs     []*v1.PersistentVolumeClaim
		wantPVCs []*v1.PersistentVolumeClaim
		wantErr  error
	}{
		{
			name: "no PVCs",
			tc: &v1alpha1.TidbCluster{
				Spec: v1alpha1.TidbClusterSpec{},
			},
		},
		{
			name: "resize PD PVCs",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: v1.NamespaceDefault,
					Name:      "tc",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("2Gi"),
							},
						},
						StorageClassName: pointer.StringPtr("sc"),
						Replicas:         3,
						StorageVolumes: []v1alpha1.StorageVolume{
							{
								Name:        "log",
								StorageSize: "2Gi",
							},
						},
					},
				},
			},
			sc: newStorageClass("sc", true),
			pvcs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "1Gi"),
				newPVCWithStorage("pd-tc-pd-1", label.PDLabelVal, "sc", "1Gi"),
				newPVCWithStorage("pd-tc-pd-2", label.PDLabelVal, "sc", "1Gi"),
				newPVCWithStorage("pd-log-tc-pd-0", label.PDLabelVal, "sc", "1Gi"),
				newPVCWithStorage("pd-log-tc-pd-1", label.PDLabelVal, "sc", "1Gi"),
				newPVCWithStorage("pd-log-tc-pd-2", label.PDLabelVal, "sc", "1Gi"),
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "2Gi"),
				newPVCWithStorage("pd-tc-pd-1", label.PDLabelVal, "sc", "2Gi"),
				newPVCWithStorage("pd-tc-pd-2", label.PDLabelVal, "sc", "2Gi"),
				newPVCWithStorage("pd-log-tc-pd-0", label.PDLabelVal, "sc", "2Gi"),
				newPVCWithStorage("pd-log-tc-pd-1", label.PDLabelVal, "sc", "2Gi"),
				newPVCWithStorage("pd-log-tc-pd-2", label.PDLabelVal, "sc", "2Gi"),
			},
		},
		{
			name: "resize TiDB PVCs",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: v1.NamespaceDefault,
					Name:      "tc",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						StorageVolumes: []v1alpha1.StorageVolume{
							{
								Name:        "log",
								StorageSize: "2Gi",
							},
						},
						Replicas: 3,
					},
				},
			},
			sc: newStorageClass("sc", true),
			pvcs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("tidb-log-tc-tidb-0", label.TiDBLabelVal, "sc", "1Gi"),
				newPVCWithStorage("tidb-log-tc-tidb-1", label.TiDBLabelVal, "sc", "1Gi"),
				newPVCWithStorage("tidb-log-tc-tidb-2", label.TiDBLabelVal, "sc", "1Gi"),
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("tidb-log-tc-tidb-0", label.TiDBLabelVal, "sc", "2Gi"),
				newPVCWithStorage("tidb-log-tc-tidb-1", label.TiDBLabelVal, "sc", "2Gi"),
				newPVCWithStorage("tidb-log-tc-tidb-2", label.TiDBLabelVal, "sc", "2Gi"),
			},
		},
		{
			name: "resize TiKV PVCs",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: v1.NamespaceDefault,
					Name:      "tc",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiKV: &v1alpha1.TiKVSpec{
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("2Gi"),
							},
						},
						StorageClassName: pointer.StringPtr("sc"),
						StorageVolumes: []v1alpha1.StorageVolume{
							{
								Name:        "log",
								StorageSize: "2Gi",
							},
						},
						Replicas: 3,
					},
				},
			},
			sc: newStorageClass("sc", true),
			pvcs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("tikv-tc-tikv-0", label.TiKVLabelVal, "sc", "1Gi"),
				newPVCWithStorage("tikv-tc-tikv-1", label.TiKVLabelVal, "sc", "1Gi"),
				newPVCWithStorage("tikv-tc-tikv-2", label.TiKVLabelVal, "sc", "1Gi"),
				newPVCWithStorage("tikv-log-tc-tikv-0", label.TiKVLabelVal, "sc", "1Gi"),
				newPVCWithStorage("tikv-log-tc-tikv-1", label.TiKVLabelVal, "sc", "1Gi"),
				newPVCWithStorage("tikv-log-tc-tikv-2", label.TiKVLabelVal, "sc", "1Gi"),
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("tikv-tc-tikv-0", label.TiKVLabelVal, "sc", "2Gi"),
				newPVCWithStorage("tikv-tc-tikv-1", label.TiKVLabelVal, "sc", "2Gi"),
				newPVCWithStorage("tikv-tc-tikv-2", label.TiKVLabelVal, "sc", "2Gi"),
				newPVCWithStorage("tikv-log-tc-tikv-0", label.TiKVLabelVal, "sc", "2Gi"),
				newPVCWithStorage("tikv-log-tc-tikv-1", label.TiKVLabelVal, "sc", "2Gi"),
				newPVCWithStorage("tikv-log-tc-tikv-2", label.TiKVLabelVal, "sc", "2Gi"),
			},
		},
		{
			name: "resize TiFlash PVCs",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: v1.NamespaceDefault,
					Name:      "tc",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiFlash: &v1alpha1.TiFlashSpec{
						Replicas: 1,
						StorageClaims: []v1alpha1.StorageClaim{
							{
								Resources: v1.ResourceRequirements{
									Requests: v1.ResourceList{
										v1.ResourceStorage: resource.MustParse("2Gi"),
									},
								},
								StorageClassName: pointer.StringPtr("sc"),
							},
							{
								Resources: v1.ResourceRequirements{
									Requests: v1.ResourceList{
										v1.ResourceStorage: resource.MustParse("3Gi"),
									},
								},
								StorageClassName: pointer.StringPtr("sc"),
							},
						},
					},
				},
			},
			sc: newStorageClass("sc", true),
			pvcs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("data0-tc-tiflash-0", label.TiFlashLabelVal, "sc", "1Gi"),
				newPVCWithStorage("data1-tc-tiflash-0", label.TiFlashLabelVal, "sc", "1Gi"),
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("data0-tc-tiflash-0", label.TiFlashLabelVal, "sc", "2Gi"),
				newPVCWithStorage("data1-tc-tiflash-0", label.TiFlashLabelVal, "sc", "3Gi"),
			},
		},
		{
			name: "resize TiCDC PVCs",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: v1.NamespaceDefault,
					Name:      "tc",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiCDC: &v1alpha1.TiCDCSpec{
						Replicas: 3,
						StorageVolumes: []v1alpha1.StorageVolume{
							{
								Name:        "sort-dir",
								StorageSize: "2Gi",
							},
						},
					},
				},
			},
			sc: newStorageClass("sc", true),
			pvcs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("ticdc-sort-dir-tc-ticdc-0", label.TiCDCLabelVal, "sc", "1Gi"),
				newPVCWithStorage("ticdc-sort-dir-tc-ticdc-1", label.TiCDCLabelVal, "sc", "1Gi"),
				newPVCWithStorage("ticdc-sort-dir-tc-ticdc-2", label.TiCDCLabelVal, "sc", "1Gi"),
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("ticdc-sort-dir-tc-ticdc-0", label.TiCDCLabelVal, "sc", "2Gi"),
				newPVCWithStorage("ticdc-sort-dir-tc-ticdc-1", label.TiCDCLabelVal, "sc", "2Gi"),
				newPVCWithStorage("ticdc-sort-dir-tc-ticdc-2", label.TiCDCLabelVal, "sc", "2Gi"),
			},
		},
		{
			name: "resize Pump PVCs",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: v1.NamespaceDefault,
					Name:      "tc",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Pump: &v1alpha1.PumpSpec{
						StorageClassName: pointer.StringPtr("sc"),
						Replicas:         1,
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("2Gi"),
							},
						},
					},
				},
			},
			sc: newStorageClass("sc", true),
			pvcs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("data-tc-pump-0", label.PumpLabelVal, "sc", "1Gi"),
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("data-tc-pump-0", label.PumpLabelVal, "sc", "2Gi"),
			},
		},
		{
			name: "storage class is missing",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: v1.NamespaceDefault,
					Name:      "tc",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						Replicas:         1,
						StorageClassName: pointer.StringPtr("sc"),
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("2Gi"),
							},
						},
					},
				},
			},
			pvcs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "1Gi"),
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "1Gi"),
			},
			wantErr: apierrors.NewNotFound(storagev1.Resource("storageclass"), "sc"),
		},
		{
			name: "storage class does not support volume expansion",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: v1.NamespaceDefault,
					Name:      "tc",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						Replicas:         1,
						StorageClassName: pointer.StringPtr("sc"),
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("2Gi"),
							},
						},
					},
				},
			},
			sc: newStorageClass("sc", false),
			pvcs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-0", label.PDLabelVal, "sc", "1Gi"),
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-0", label.PDLabelVal, "sc", "1Gi"),
			},
			wantErr: nil,
		},
		{
			name: "shrinking is not supported",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: v1.NamespaceDefault,
					Name:      "tc",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						Replicas:         1,
						StorageClassName: pointer.StringPtr("sc"),
						ResourceRequirements: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
			sc: newStorageClass("sc", false),
			pvcs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-0", label.PDLabelVal, "sc", "2Gi"),
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-0", label.PDLabelVal, "sc", "2Gi"),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fakeDeps := controller.NewFakeDependencies()
			for _, pvc := range tt.pvcs {
				fakeDeps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
			}
			if tt.sc != nil {
				fakeDeps.KubeClientset.StorageV1().StorageClasses().Create(context.TODO(), tt.sc, metav1.CreateOptions{})
			}
			for _, pod := range newPodsFromTC(tt.tc) {
				fakeDeps.KubeClientset.CoreV1().Pods(tt.tc.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
			}

			resizer := NewPVCResizer(fakeDeps)

			informerFactory := fakeDeps.KubeInformerFactory
			informerFactory.Start(ctx.Done())
			informerFactory.WaitForCacheSync(ctx.Done())

			err := resizer.Sync(tt.tc)
			if tt.wantErr != nil {
				g.Expect(err.Error()).To(gomega.ContainSubstring(tt.wantErr.Error()))
			} else {
				g.Expect(err).To(gomega.Succeed())
			}

			for i, pvc := range tt.pvcs {
				wantPVC := tt.wantPVCs[i]
				got, err := fakeDeps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(context.TODO(), pvc.Name, metav1.GetOptions{})
				g.Expect(err).To(gomega.Succeed())
				diff := cmp.Diff(wantPVC, got)
				g.Expect(diff).To(gomega.BeEmpty(), "unexpected (-want, +got): %s", diff)
			}
		})
	}
}

func TestDMPVCResizer(t *testing.T) {
	tests := []struct {
		name     string
		dc       *v1alpha1.DMCluster
		sc       *storagev1.StorageClass
		pvcs     []*v1.PersistentVolumeClaim
		wantPVCs []*v1.PersistentVolumeClaim
		wantErr  error
	}{
		{
			name: "no PVCs",
			dc: &v1alpha1.DMCluster{
				Spec: v1alpha1.DMClusterSpec{},
			},
		},
		{
			name: "resize dm-master PVCs",
			dc: &v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: v1.NamespaceDefault,
					Name:      "dc",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						StorageSize:      "2Gi",
						StorageClassName: pointer.StringPtr("sc"),
						Replicas:         3,
					},
				},
			},
			sc: newStorageClass("sc", true),
			pvcs: []*v1.PersistentVolumeClaim{
				newDMPVCWithStorage("dm-master-dc-dm-master-0", label.DMMasterLabelVal, "sc", "1Gi"),
				newDMPVCWithStorage("dm-master-dc-dm-master-1", label.DMMasterLabelVal, "sc", "1Gi"),
				newDMPVCWithStorage("dm-master-dc-dm-master-2", label.DMMasterLabelVal, "sc", "1Gi"),
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newDMPVCWithStorage("dm-master-dc-dm-master-0", label.DMMasterLabelVal, "sc", "2Gi"),
				newDMPVCWithStorage("dm-master-dc-dm-master-1", label.DMMasterLabelVal, "sc", "2Gi"),
				newDMPVCWithStorage("dm-master-dc-dm-master-2", label.DMMasterLabelVal, "sc", "2Gi"),
			},
		},
		{
			name: "resize dm-worker PVCs",
			dc: &v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: v1.NamespaceDefault,
					Name:      "dc",
				},
				Spec: v1alpha1.DMClusterSpec{
					Worker: &v1alpha1.WorkerSpec{
						StorageSize:      "2Gi",
						StorageClassName: pointer.StringPtr("sc"),
						Replicas:         3,
					},
				},
			},
			sc: newStorageClass("sc", true),
			pvcs: []*v1.PersistentVolumeClaim{
				newDMPVCWithStorage("dm-worker-dc-dm-worker-0", label.DMWorkerLabelVal, "sc", "1Gi"),
				newDMPVCWithStorage("dm-worker-dc-dm-worker-1", label.DMWorkerLabelVal, "sc", "1Gi"),
				newDMPVCWithStorage("dm-worker-dc-dm-worker-2", label.DMWorkerLabelVal, "sc", "1Gi"),
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newDMPVCWithStorage("dm-worker-dc-dm-worker-0", label.DMWorkerLabelVal, "sc", "2Gi"),
				newDMPVCWithStorage("dm-worker-dc-dm-worker-1", label.DMWorkerLabelVal, "sc", "2Gi"),
				newDMPVCWithStorage("dm-worker-dc-dm-worker-2", label.DMWorkerLabelVal, "sc", "2Gi"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fakeDeps := controller.NewFakeDependencies()

			for _, pvc := range tt.pvcs {
				fakeDeps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
			}
			if tt.sc != nil {
				fakeDeps.KubeClientset.StorageV1().StorageClasses().Create(context.TODO(), tt.sc, metav1.CreateOptions{})
			}
			for _, pod := range newPodsFromDC(tt.dc) {
				fakeDeps.KubeClientset.CoreV1().Pods(tt.dc.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
			}

			informerFactory := fakeDeps.KubeInformerFactory
			resizer := NewPVCResizer(fakeDeps)

			informerFactory.Start(ctx.Done())
			informerFactory.WaitForCacheSync(ctx.Done())

			err := resizer.SyncDM(tt.dc)
			if tt.wantErr != nil {
				g.Expect(err.Error()).To(gomega.ContainSubstring(tt.wantErr.Error()))
			} else {
				g.Expect(err).To(gomega.Succeed())
			}

			for i, pvc := range tt.pvcs {
				wantPVC := tt.wantPVCs[i]
				got, err := fakeDeps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(context.TODO(), pvc.Name, metav1.GetOptions{})
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(wantPVC, got); diff != "" {
					t.Errorf("unexpected (-want, +got): %s", diff)
				}
			}
		})
	}
}

func TestUpdateVolumeStatus(t *testing.T) {
	scName := "sc"

	tests := []struct {
		name                  string
		desiredVolumeQuantity map[string]resource.Quantity
		actualPodVolumes      []*podVolumeContext
		expect                map[string]*v1alpha1.StorageVolumeStatus
	}{
		{
			name: "volumes start to resize",
			desiredVolumeQuantity: map[string]resource.Quantity{
				"volume-1": resource.MustParse("2Gi"),
				"volume-2": resource.MustParse("2Gi"),
			},
			actualPodVolumes: []*podVolumeContext{
				{
					volToPVCs: map[string]*v1.PersistentVolumeClaim{
						"volume-1": newMockPVC("volume-1-pvc-1", scName, "2Gi", "1Gi"),
						"volume-2": newMockPVC("volume-2-pvc-1", scName, "2Gi", "1Gi"),
					},
				},
				{
					volToPVCs: map[string]*v1.PersistentVolumeClaim{
						"volume-1": newMockPVC("volume-1-pvc-2", scName, "2Gi", "1Gi"),
						"volume-2": newMockPVC("volume-2-pvc-2", scName, "2Gi", "1Gi"),
					},
				},
				{
					volToPVCs: map[string]*v1.PersistentVolumeClaim{
						"volume-1": newMockPVC("volume-1-pvc-3", scName, "2Gi", "1Gi"),
						"volume-2": newMockPVC("volume-2-pvc-3", scName, "2Gi", "1Gi"),
					},
				},
			},
			expect: map[string]*v1alpha1.StorageVolumeStatus{
				"volume-1": {
					Name:            "volume-1",
					BoundCount:      3,
					CurrentCount:    3,
					ResizedCount:    0,
					CurrentCapacity: resource.MustParse("1Gi"),
					ResizedCapacity: resource.MustParse("2Gi"),
				},
				"volume-2": {
					Name:            "volume-2",
					BoundCount:      3,
					CurrentCount:    3,
					ResizedCount:    0,
					CurrentCapacity: resource.MustParse("1Gi"),
					ResizedCapacity: resource.MustParse("2Gi"),
				},
			},
		},
		{
			name: "some volumes is resizing",
			desiredVolumeQuantity: map[string]resource.Quantity{
				"volume-1": resource.MustParse("2Gi"),
				"volume-2": resource.MustParse("2Gi"),
			},
			actualPodVolumes: []*podVolumeContext{
				{
					volToPVCs: map[string]*v1.PersistentVolumeClaim{
						"volume-1": newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi"), // resized
						"volume-2": newMockPVC("volume-2-pvc-1", scName, "2Gi", "1Gi"),
					},
				},
				{
					volToPVCs: map[string]*v1.PersistentVolumeClaim{
						"volume-1": newMockPVC("volume-1-pvc-2", scName, "2Gi", "1Gi"),
						"volume-2": newMockPVC("volume-2-pvc-2", scName, "2Gi", "2Gi"), // resized
					},
				},
				{
					volToPVCs: map[string]*v1.PersistentVolumeClaim{
						"volume-1": newMockPVC("volume-1-pvc-3", scName, "2Gi", "1Gi"),
						"volume-2": newMockPVC("volume-2-pvc-3", scName, "2Gi", "2Gi"), // resized
					},
				},
			},
			expect: map[string]*v1alpha1.StorageVolumeStatus{
				"volume-1": {
					Name:            "volume-1",
					BoundCount:      3,
					CurrentCount:    2,
					ResizedCount:    1,
					CurrentCapacity: resource.MustParse("1Gi"),
					ResizedCapacity: resource.MustParse("2Gi"),
				},
				"volume-2": {
					Name:            "volume-2",
					BoundCount:      3,
					CurrentCount:    1,
					ResizedCount:    2,
					CurrentCapacity: resource.MustParse("1Gi"),
					ResizedCapacity: resource.MustParse("2Gi"),
				},
			},
		},
		{
			name: "all volumes is resized",
			desiredVolumeQuantity: map[string]resource.Quantity{
				"volume-1": resource.MustParse("2Gi"),
				"volume-2": resource.MustParse("2Gi"),
			},
			actualPodVolumes: []*podVolumeContext{
				{
					volToPVCs: map[string]*v1.PersistentVolumeClaim{
						"volume-1": newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi"),
						"volume-2": newMockPVC("volume-2-pvc-1", scName, "2Gi", "2Gi"),
					},
				},
				{
					volToPVCs: map[string]*v1.PersistentVolumeClaim{
						"volume-1": newMockPVC("volume-1-pvc-2", scName, "2Gi", "2Gi"),
						"volume-2": newMockPVC("volume-2-pvc-2", scName, "2Gi", "2Gi"),
					},
				},
				{
					volToPVCs: map[string]*v1.PersistentVolumeClaim{
						"volume-1": newMockPVC("volume-1-pvc-3", scName, "2Gi", "2Gi"),
						"volume-2": newMockPVC("volume-2-pvc-3", scName, "2Gi", "2Gi"),
					},
				},
			},
			expect: map[string]*v1alpha1.StorageVolumeStatus{
				"volume-1": {
					Name:            "volume-1",
					BoundCount:      3,
					CurrentCount:    3,
					ResizedCount:    3,
					CurrentCapacity: resource.MustParse("2Gi"),
					ResizedCapacity: resource.MustParse("2Gi"),
				},
				"volume-2": {
					Name:            "volume-2",
					BoundCount:      3,
					CurrentCount:    3,
					ResizedCount:    3,
					CurrentCapacity: resource.MustParse("2Gi"),
					ResizedCapacity: resource.MustParse("2Gi"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)
			resizer := &pvcResizer{
				deps: controller.NewFakeDependencies(),
			}

			ctx := &componentVolumeContext{
				desiredVolumeQuantity: tt.desiredVolumeQuantity,
				actualPodVolumes:      tt.actualPodVolumes,
			}
			ctx.updateVolumeStatusFn = func(m map[string]*v1alpha1.StorageVolumeStatus) {
				diff := cmp.Diff(m, tt.expect)
				g.Expect(diff).Should(gomega.BeEmpty(), "unexpected (-want, +got)")
			}

			resizer.updateVolumeStatus(ctx)
		})
	}
}

func newPodsFromTC(tc *v1alpha1.TidbCluster) []*corev1.Pod {
	pods := []*corev1.Pod{}

	addPods := func(replica int32, comp v1alpha1.MemberType,
		dataVolumeStorageClass *string, storageVolumes []v1alpha1.StorageVolume, storageClaims []v1alpha1.StorageClaim) {
		for i := int32(0); i < replica; i++ {
			pods = append(pods, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: func() string {
						name, _ := MemberPodName(tc.Name, v1alpha1.TiDBClusterKind, i, comp)
						return name
					}(),
					Namespace: tc.GetNamespace(),
					Labels: map[string]string{
						label.NameLabelKey:      "tidb-cluster",
						label.ComponentLabelKey: comp.String(),
						label.ManagedByLabelKey: label.TiDBOperator,
						label.InstanceLabelKey:  tc.GetInstanceName(),
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{},
				},
			})

		}
		addVolumeForPod(pods, comp, dataVolumeStorageClass, storageVolumes, storageClaims)
	}

	if tc.Spec.PD != nil {
		addPods(tc.Spec.PD.Replicas, v1alpha1.PDMemberType, tc.Spec.PD.StorageClassName, tc.Spec.PD.StorageVolumes, nil)
	}
	if tc.Spec.TiDB != nil {
		addPods(tc.Spec.TiDB.Replicas, v1alpha1.TiDBMemberType, nil, tc.Spec.TiDB.StorageVolumes, nil)
	}
	if tc.Spec.TiKV != nil {
		addPods(tc.Spec.TiKV.Replicas, v1alpha1.TiKVMemberType, tc.Spec.TiKV.StorageClassName, tc.Spec.TiKV.StorageVolumes, nil)
	}
	if tc.Spec.TiFlash != nil {
		addPods(tc.Spec.TiFlash.Replicas, v1alpha1.TiFlashMemberType, nil, nil, tc.Spec.TiFlash.StorageClaims)
	}
	if tc.Spec.TiCDC != nil {
		addPods(tc.Spec.TiCDC.Replicas, v1alpha1.TiCDCMemberType, nil, tc.Spec.TiCDC.StorageVolumes, nil)
	}
	if tc.Spec.Pump != nil {
		addPods(tc.Spec.Pump.Replicas, v1alpha1.PumpMemberType, tc.Spec.Pump.StorageClassName, nil, nil)
	}

	return pods
}

func newPodsFromDC(dc *v1alpha1.DMCluster) []*corev1.Pod {
	pods := []*corev1.Pod{}

	addPods := func(replica int32, comp v1alpha1.MemberType,
		dataVolumeStorageClass *string, storageVolumes []v1alpha1.StorageVolume) {
		for i := int32(0); i < replica; i++ {
			pods = append(pods, &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s-%d", dc.Name, comp.String(), i),
					Namespace: dc.GetNamespace(),
					Labels: map[string]string{
						label.NameLabelKey:      "dm-cluster",
						label.ComponentLabelKey: comp.String(),
						label.ManagedByLabelKey: label.TiDBOperator,
						label.InstanceLabelKey:  dc.GetInstanceName(),
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{},
				},
			})

		}
		addVolumeForPod(pods, comp, dataVolumeStorageClass, storageVolumes, nil)
	}

	addPods(dc.Spec.Master.Replicas, v1alpha1.DMMasterMemberType, dc.Spec.Master.StorageClassName, nil)
	if dc.Spec.Worker != nil {
		addPods(dc.Spec.Worker.Replicas, v1alpha1.DMWorkerMemberType, dc.Spec.Worker.StorageClassName, nil)
	}

	return pods
}

func addVolumeForPod(pods []*corev1.Pod, comp v1alpha1.MemberType,
	dataVolumeStorageClass *string, storageVolumes []v1alpha1.StorageVolume, storageClaims []v1alpha1.StorageClaim) {
	volumeNames := []string{}

	if dataVolumeStorageClass != nil {
		volumeNames = append(volumeNames, v1alpha1.GetPVCTemplateName("", comp))
	}
	for _, sv := range storageVolumes {
		volumeNames = append(volumeNames, v1alpha1.GetPVCTemplateName(sv.Name, comp))
	}
	for i := range storageClaims {
		volumeNames = append(volumeNames, v1alpha1.GetPVCTemplateNameForTiFlash(i))
	}

	for _, volName := range volumeNames {
		for _, pod := range pods {
			volume := corev1.Volume{}
			volume.Name = volName
			volume.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: fmt.Sprintf("%s-%s", volume.Name, pod.Name),
			}
			pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		}

	}
}
