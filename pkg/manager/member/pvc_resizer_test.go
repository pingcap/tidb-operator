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
	newResourceList := func(storageRequest string) v1.ResourceList {
		return v1.ResourceList{v1.ResourceStorage: resource.MustParse(storageRequest)}
	}

	tests := []struct {
		name     string
		setup    func(tc *v1alpha1.TidbCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim
		wantPVCs []*v1.PersistentVolumeClaim
		wantErr  error
		expect   func(g *gomega.WithT, tc *v1alpha1.TidbCluster)
	}{
		{
			name: "no PVCs",
			setup: func(tc *v1alpha1.TidbCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				return []*v1.PersistentVolumeClaim{}
			},
		},
		{
			name: "resize PD PVCs",
			setup: func(tc *v1alpha1.TidbCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				tc.Spec.PD = &v1alpha1.PDSpec{}
				tc.Spec.PD.Requests = newResourceList("2Gi")
				tc.Spec.PD.Replicas = 1
				tc.Spec.PD.StorageClassName = &sc.Name
				tc.Spec.PD.StorageVolumes = []v1alpha1.StorageVolume{
					{
						Name:        "log",
						StorageSize: "2Gi",
					},
				}
				pvcs := []*v1.PersistentVolumeClaim{
					newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "1Gi"),
					newPVCWithStorage("pd-log-tc-pd-0", label.PDLabelVal, "sc", "1Gi"),
				}
				return pvcs
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "2Gi"),
				newPVCWithStorage("pd-log-tc-pd-0", label.PDLabelVal, "sc", "2Gi"),
			},
			expect: func(g *gomega.WithT, tc *v1alpha1.TidbCluster) {
				expectStatus := map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
					"pd": {
						Name: "pd",
						ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
							BoundCount:      1,
							CurrentCount:    1,
							ResizedCount:    0,
							CurrentCapacity: resource.MustParse("1Gi"),
							ResizedCapacity: resource.MustParse("2Gi"),
						},
					},
					"pd-log": {
						Name: "pd-log",
						ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
							BoundCount:      1,
							CurrentCount:    1,
							ResizedCount:    0,
							CurrentCapacity: resource.MustParse("1Gi"),
							ResizedCapacity: resource.MustParse("2Gi"),
						},
					},
				}
				diff := cmp.Diff(expectStatus, tc.Status.PD.Volumes)
				g.Expect(diff).To(gomega.BeEmpty(), "unexpected (-want, +got): %s", diff)
			},
		},
		{
			name: "resize TiDB PVCs",
			setup: func(tc *v1alpha1.TidbCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				tc.Spec.TiDB = &v1alpha1.TiDBSpec{}
				tc.Spec.TiDB.Replicas = 1
				tc.Spec.TiDB.StorageVolumes = []v1alpha1.StorageVolume{
					{
						Name:        "log",
						StorageSize: "2Gi",
					},
				}
				pvcs := []*v1.PersistentVolumeClaim{
					newPVCWithStorage("tidb-log-tc-tidb-0", label.TiDBLabelVal, "sc", "1Gi"),
				}
				return pvcs
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("tidb-log-tc-tidb-0", label.TiDBLabelVal, "sc", "2Gi"),
			},
			expect: func(g *gomega.WithT, tc *v1alpha1.TidbCluster) {
				expectStatus := map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
					"tidb-log": {
						Name: "tidb-log",
						ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
							BoundCount:      1,
							CurrentCount:    1,
							ResizedCount:    0,
							CurrentCapacity: resource.MustParse("1Gi"),
							ResizedCapacity: resource.MustParse("2Gi"),
						},
					},
				}
				diff := cmp.Diff(expectStatus, tc.Status.TiDB.Volumes)
				g.Expect(diff).To(gomega.BeEmpty(), "unexpected (-want, +got): %s", diff)
			},
		},
		{
			name: "resize TiKV PVCs",
			setup: func(tc *v1alpha1.TidbCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				tc.Spec.TiKV = &v1alpha1.TiKVSpec{}
				tc.Spec.TiKV.Replicas = 1
				tc.Spec.TiKV.Requests = newResourceList("2Gi")
				tc.Spec.TiKV.StorageClassName = &sc.Name
				tc.Spec.TiKV.StorageVolumes = []v1alpha1.StorageVolume{
					{
						Name:        "log",
						StorageSize: "2Gi",
					},
				}
				pvcs := []*v1.PersistentVolumeClaim{
					newPVCWithStorage("tikv-tc-tikv-0", label.TiKVLabelVal, "sc", "1Gi"),
					newPVCWithStorage("tikv-log-tc-tikv-0", label.TiKVLabelVal, "sc", "1Gi"),
				}
				return pvcs
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("tikv-tc-tikv-0", label.TiKVLabelVal, "sc", "2Gi"),
				newPVCWithStorage("tikv-log-tc-tikv-0", label.TiKVLabelVal, "sc", "2Gi"),
			},
			expect: func(g *gomega.WithT, tc *v1alpha1.TidbCluster) {
				expectStatus := map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
					"tikv": {
						Name: "tikv",
						ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
							BoundCount:      1,
							CurrentCount:    1,
							ResizedCount:    0,
							CurrentCapacity: resource.MustParse("1Gi"),
							ResizedCapacity: resource.MustParse("2Gi"),
						},
					},
					"tikv-log": {
						Name: "tikv-log",
						ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
							BoundCount:      1,
							CurrentCount:    1,
							ResizedCount:    0,
							CurrentCapacity: resource.MustParse("1Gi"),
							ResizedCapacity: resource.MustParse("2Gi"),
						},
					},
				}
				diff := cmp.Diff(expectStatus, tc.Status.TiKV.Volumes)
				g.Expect(diff).To(gomega.BeEmpty(), "unexpected (-want, +got): %s", diff)
			},
		},
		{
			name: "resize TiFlash PVCs",
			setup: func(tc *v1alpha1.TidbCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				tc.Spec.TiFlash = &v1alpha1.TiFlashSpec{}
				tc.Spec.TiFlash.Replicas = 1
				tc.Spec.TiFlash.Requests = newResourceList("2Gi")
				tc.Spec.TiFlash.StorageClaims = []v1alpha1.StorageClaim{
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
				}
				pvcs := []*v1.PersistentVolumeClaim{
					newPVCWithStorage("data0-tc-tiflash-0", label.TiFlashLabelVal, "sc", "1Gi"),
					newPVCWithStorage("data1-tc-tiflash-0", label.TiFlashLabelVal, "sc", "1Gi"),
				}
				return pvcs
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("data0-tc-tiflash-0", label.TiFlashLabelVal, "sc", "2Gi"),
				newPVCWithStorage("data1-tc-tiflash-0", label.TiFlashLabelVal, "sc", "3Gi"),
			},
			expect: func(g *gomega.WithT, tc *v1alpha1.TidbCluster) {
				expectStatus := map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
					"data0": {
						Name: "data0",
						ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
							BoundCount:      1,
							CurrentCount:    1,
							ResizedCount:    0,
							CurrentCapacity: resource.MustParse("1Gi"),
							ResizedCapacity: resource.MustParse("2Gi"),
						},
					},
					"data1": {
						Name: "data1",
						ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
							BoundCount:      1,
							CurrentCount:    1,
							ResizedCount:    0,
							CurrentCapacity: resource.MustParse("1Gi"),
							ResizedCapacity: resource.MustParse("3Gi"),
						},
					},
				}
				diff := cmp.Diff(expectStatus, tc.Status.TiFlash.Volumes)
				g.Expect(diff).To(gomega.BeEmpty(), "unexpected (-want, +got): %s", diff)
			},
		},
		{
			name: "resize TiCDC PVCs",
			setup: func(tc *v1alpha1.TidbCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				tc.Spec.TiCDC = &v1alpha1.TiCDCSpec{}
				tc.Spec.TiCDC.Replicas = 1
				tc.Spec.TiCDC.StorageVolumes = []v1alpha1.StorageVolume{
					{
						Name:        "sort-dir",
						StorageSize: "2Gi",
					},
				}
				pvcs := []*v1.PersistentVolumeClaim{
					newPVCWithStorage("ticdc-sort-dir-tc-ticdc-0", label.TiCDCLabelVal, "sc", "1Gi"),
				}
				return pvcs
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("ticdc-sort-dir-tc-ticdc-0", label.TiCDCLabelVal, "sc", "2Gi"),
			},
			expect: func(g *gomega.WithT, tc *v1alpha1.TidbCluster) {
				expectStatus := map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
					"ticdc-sort-dir": {
						Name: "ticdc-sort-dir",
						ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
							BoundCount:      1,
							CurrentCount:    1,
							ResizedCount:    0,
							CurrentCapacity: resource.MustParse("1Gi"),
							ResizedCapacity: resource.MustParse("2Gi"),
						},
					},
				}
				diff := cmp.Diff(expectStatus, tc.Status.TiCDC.Volumes)
				g.Expect(diff).To(gomega.BeEmpty(), "unexpected (-want, +got): %s", diff)
			},
		},
		{
			name: "resize Pump PVCs",
			setup: func(tc *v1alpha1.TidbCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				tc.Spec.Pump = &v1alpha1.PumpSpec{}
				tc.Spec.Pump.Replicas = 1
				tc.Spec.Pump.Requests = newResourceList("2Gi")
				tc.Spec.Pump.StorageClassName = &sc.Name
				pvcs := []*v1.PersistentVolumeClaim{
					newPVCWithStorage("data-tc-pump-0", label.PumpLabelVal, "sc", "1Gi"),
				}
				return pvcs
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("data-tc-pump-0", label.PumpLabelVal, "sc", "2Gi"),
			},
			expect: func(g *gomega.WithT, tc *v1alpha1.TidbCluster) {
				expectStatus := map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
					"data": {
						Name: "data",
						ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
							BoundCount:      1,
							CurrentCount:    1,
							ResizedCount:    0,
							CurrentCapacity: resource.MustParse("1Gi"),
							ResizedCapacity: resource.MustParse("2Gi"),
						},
					},
				}
				diff := cmp.Diff(expectStatus, tc.Status.Pump.Volumes)
				g.Expect(diff).To(gomega.BeEmpty(), "unexpected (-want, +got): %s", diff)
			},
		},
		{
			name: "storage class is missing",
			setup: func(tc *v1alpha1.TidbCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				tc.Spec.PD = &v1alpha1.PDSpec{}
				tc.Spec.PD.Replicas = 1
				tc.Spec.PD.Requests = newResourceList("2Gi")
				tc.Spec.PD.StorageClassName = &sc.Name
				sc.Name = "other-sc"
				pvcs := []*v1.PersistentVolumeClaim{
					newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "1Gi"),
				}
				return pvcs
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "1Gi"),
			},
			wantErr: apierrors.NewNotFound(storagev1.Resource("storageclass"), "sc"),
		},
		{
			name: "storage class does not support volume expansion",
			setup: func(tc *v1alpha1.TidbCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				tc.Spec.PD = &v1alpha1.PDSpec{}
				tc.Spec.PD.Replicas = 1
				tc.Spec.PD.Requests = newResourceList("2Gi")
				tc.Spec.PD.StorageClassName = &sc.Name
				pvcs := []*v1.PersistentVolumeClaim{
					newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "1Gi"),
				}
				sc.AllowVolumeExpansion = pointer.BoolPtr(false)
				return pvcs
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "1Gi"),
			},
			wantErr: nil,
		},
		{
			name: "shrinking is not supported",
			setup: func(tc *v1alpha1.TidbCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				tc.Spec.PD = &v1alpha1.PDSpec{}
				tc.Spec.PD.Replicas = 1
				tc.Spec.PD.Requests = newResourceList("1Gi")
				tc.Spec.PD.StorageClassName = &sc.Name
				pvcs := []*v1.PersistentVolumeClaim{
					newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "2Gi"),
				}
				return pvcs
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "2Gi"),
			},
			wantErr: nil,
		},
		{
			name: "skip when any pvc is resizing",
			setup: func(tc *v1alpha1.TidbCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				tc.Spec.PD = &v1alpha1.PDSpec{}
				tc.Spec.PD.Replicas = 1
				tc.Spec.PD.Requests = newResourceList("2Gi")
				tc.Spec.PD.StorageClassName = &sc.Name
				pvcs := []*v1.PersistentVolumeClaim{
					newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "1Gi"),
				}
				pvcs[0].Status.Conditions = []v1.PersistentVolumeClaimCondition{
					{
						Type: v1.PersistentVolumeClaimResizing,
					},
				}
				return pvcs
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "1Gi"),
			},
			wantErr: nil,
		},
		{
			name: "resize one pvc only",
			setup: func(tc *v1alpha1.TidbCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				tc.Spec.PD = &v1alpha1.PDSpec{}
				tc.Spec.PD.Replicas = 1
				tc.Spec.PD.Requests = newResourceList("2Gi")
				tc.Spec.PD.StorageClassName = &sc.Name
				pvcs := []*v1.PersistentVolumeClaim{
					newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "1Gi"),
					newPVCWithStorage("pd-tc-pd-1", label.PDLabelVal, "sc", "1Gi"),
				}
				return pvcs
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newPVCWithStorage("pd-tc-pd-0", label.PDLabelVal, "sc", "2Gi"),
				newPVCWithStorage("pd-tc-pd-1", label.PDLabelVal, "sc", "1Gi"),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

			tc := &v1alpha1.TidbCluster{}
			tc.Name = "tc"
			tc.Namespace = v1.NamespaceDefault
			sc := newStorageClass("sc", true)
			pvcs := tt.setup(tc, sc)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fakeDeps := controller.NewFakeDependencies()
			for _, pvc := range pvcs {
				fakeDeps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
			}
			fakeDeps.KubeClientset.StorageV1().StorageClasses().Create(context.TODO(), sc, metav1.CreateOptions{})
			for _, pod := range newPodsFromTC(tc) {
				fakeDeps.KubeClientset.CoreV1().Pods(tc.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
			}

			resizer := NewPVCResizer(fakeDeps)

			informerFactory := fakeDeps.KubeInformerFactory
			informerFactory.Start(ctx.Done())
			informerFactory.WaitForCacheSync(ctx.Done())

			err := resizer.Sync(tc)
			if tt.wantErr != nil {
				g.Expect(err).To(gomega.HaveOccurred())
				g.Expect(err.Error()).To(gomega.ContainSubstring(tt.wantErr.Error()))
			} else {
				g.Expect(err).To(gomega.Succeed())
			}

			for i, pvc := range pvcs {
				wantPVC := tt.wantPVCs[i]
				got, err := fakeDeps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(context.TODO(), pvc.Name, metav1.GetOptions{})
				g.Expect(err).To(gomega.Succeed())
				got.Status = wantPVC.Status // to ignore resource status
				diff := cmp.Diff(wantPVC, got)
				g.Expect(diff).To(gomega.BeEmpty(), "unexpected (-want, +got): %s", diff)
			}

			if tt.expect != nil {
				tt.expect(g, tc)
			}
		})
	}
}

func TestDMPVCResizer(t *testing.T) {

	tests := []struct {
		name     string
		setup    func(dc *v1alpha1.DMCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim
		wantPVCs []*v1.PersistentVolumeClaim
		wantErr  error
		expect   func(g *gomega.WithT, dc *v1alpha1.DMCluster)
	}{
		{
			name: "no PVCs",
			setup: func(dc *v1alpha1.DMCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				return []*v1.PersistentVolumeClaim{}
			},
		},
		{
			name: "resize dm-master PVCs",
			setup: func(dc *v1alpha1.DMCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				dc.Spec.Master = v1alpha1.MasterSpec{}
				dc.Spec.Master.StorageSize = "2Gi"
				dc.Spec.Master.Replicas = 1
				dc.Spec.Master.StorageClassName = &sc.Name
				pvcs := []*v1.PersistentVolumeClaim{
					newDMPVCWithStorage("dm-master-dc-dm-master-0", label.DMMasterLabelVal, "sc", "1Gi"),
				}
				return pvcs
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newDMPVCWithStorage("dm-master-dc-dm-master-0", label.DMMasterLabelVal, "sc", "2Gi"),
			},
			expect: func(g *gomega.WithT, dc *v1alpha1.DMCluster) {
				expectStatus := map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
					"dm-master": {
						Name: "dm-master",
						ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
							BoundCount:      1,
							CurrentCount:    1,
							ResizedCount:    0,
							CurrentCapacity: resource.MustParse("1Gi"),
							ResizedCapacity: resource.MustParse("2Gi"),
						},
					},
				}
				diff := cmp.Diff(expectStatus, dc.Status.Master.Volumes)
				g.Expect(diff).To(gomega.BeEmpty(), "unexpected (-want, +got): %s", diff)
			},
		},
		{
			name: "resize dm-worker PVCs",
			setup: func(dc *v1alpha1.DMCluster, sc *storagev1.StorageClass) []*v1.PersistentVolumeClaim {
				dc.Spec.Worker = &v1alpha1.WorkerSpec{}
				dc.Spec.Worker.StorageSize = "2Gi"
				dc.Spec.Worker.Replicas = 1
				dc.Spec.Worker.StorageClassName = &sc.Name
				pvcs := []*v1.PersistentVolumeClaim{
					newDMPVCWithStorage("dm-worker-dc-dm-worker-0", label.DMWorkerLabelVal, "sc", "1Gi"),
				}
				return pvcs
			},
			wantPVCs: []*v1.PersistentVolumeClaim{
				newDMPVCWithStorage("dm-worker-dc-dm-worker-0", label.DMWorkerLabelVal, "sc", "2Gi"),
			},
			expect: func(g *gomega.WithT, dc *v1alpha1.DMCluster) {
				expectStatus := map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
					"dm-worker": {
						Name: "dm-worker",
						ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
							BoundCount:      1,
							CurrentCount:    1,
							ResizedCount:    0,
							CurrentCapacity: resource.MustParse("1Gi"),
							ResizedCapacity: resource.MustParse("2Gi"),
						},
					},
				}
				diff := cmp.Diff(expectStatus, dc.Status.Worker.Volumes)
				g.Expect(diff).To(gomega.BeEmpty(), "unexpected (-want, +got): %s", diff)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

			dc := &v1alpha1.DMCluster{}
			dc.Name = "dc"
			dc.Namespace = v1.NamespaceDefault
			sc := newStorageClass("sc", true)
			pvcs := tt.setup(dc, sc)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fakeDeps := controller.NewFakeDependencies()
			for _, pvc := range pvcs {
				fakeDeps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
			}
			fakeDeps.KubeClientset.StorageV1().StorageClasses().Create(context.TODO(), sc, metav1.CreateOptions{})
			for _, pod := range newPodsFromDC(dc) {
				fakeDeps.KubeClientset.CoreV1().Pods(dc.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
			}

			informerFactory := fakeDeps.KubeInformerFactory
			resizer := NewPVCResizer(fakeDeps)

			informerFactory.Start(ctx.Done())
			informerFactory.WaitForCacheSync(ctx.Done())

			err := resizer.SyncDM(dc)
			if tt.wantErr != nil {
				g.Expect(err.Error()).To(gomega.ContainSubstring(tt.wantErr.Error()))
			} else {
				g.Expect(err).To(gomega.Succeed())
			}

			for i, pvc := range pvcs {
				wantPVC := tt.wantPVCs[i]
				got, err := fakeDeps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(context.TODO(), pvc.Name, metav1.GetOptions{})
				g.Expect(err).To(gomega.Succeed())
				got.Status.Capacity[v1.ResourceStorage] = got.Spec.Resources.Requests[v1.ResourceStorage] // to ignore resource status
				diff := cmp.Diff(wantPVC, got)
				g.Expect(diff).To(gomega.BeEmpty(), "unexpected (-want, +got): %s", diff)
			}

			if tt.expect != nil {
				tt.expect(g, dc)
			}
		})
	}
}

func TestAssembleObserveVolumeStatus(t *testing.T) {
	scName := "sc"

	tests := []struct {
		name   string
		setup  func(ctx *componentVolumeContext)
		expect map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus
	}{
		{
			name: "volumes start to resize",
			setup: func(ctx *componentVolumeContext) {
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
					"volume-2": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = map[v1alpha1.StorageVolumeName][]*podVolumeContext{
					"volume-1": {
						{pvc: newMockPVC("volume-1-pvc-1", scName, "2Gi", "1Gi")},
						{pvc: newMockPVC("volume-1-pvc-2", scName, "2Gi", "1Gi")},
						{pvc: newMockPVC("volume-1-pvc-3", scName, "2Gi", "1Gi")},
					},
					"volume-2": {
						{pvc: newMockPVC("volume-2-pvc-1", scName, "2Gi", "1Gi")},
						{pvc: newMockPVC("volume-2-pvc-2", scName, "2Gi", "1Gi")},
						{pvc: newMockPVC("volume-2-pvc-3", scName, "2Gi", "1Gi")},
					},
				}
			},
			expect: map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus{
				"volume-1": {
					BoundCount:      3,
					CurrentCount:    3,
					ResizedCount:    0,
					CurrentCapacity: resource.MustParse("1Gi"),
					ResizedCapacity: resource.MustParse("2Gi"),
				},
				"volume-2": {
					BoundCount:      3,
					CurrentCount:    3,
					ResizedCount:    0,
					CurrentCapacity: resource.MustParse("1Gi"),
					ResizedCapacity: resource.MustParse("2Gi"),
				},
			},
		},
		{
			name: "some volumes is not resized",
			setup: func(ctx *componentVolumeContext) {
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
					"volume-2": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = map[v1alpha1.StorageVolumeName][]*podVolumeContext{
					"volume-1": {
						{pvc: newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi")},
						{pvc: newMockPVC("volume-1-pvc-2", scName, "2Gi", "1Gi")}, // not resized
						{pvc: newMockPVC("volume-1-pvc-3", scName, "2Gi", "1Gi")}, // not resized
					},
					"volume-2": {
						{pvc: newMockPVC("volume-2-pvc-1", scName, "2Gi", "1Gi")}, // not resized
						{pvc: newMockPVC("volume-2-pvc-2", scName, "2Gi", "2Gi")},
						{pvc: newMockPVC("volume-2-pvc-3", scName, "2Gi", "2Gi")},
					},
				}
			},
			expect: map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus{
				"volume-1": {
					BoundCount:      3,
					CurrentCount:    2,
					ResizedCount:    1,
					CurrentCapacity: resource.MustParse("1Gi"),
					ResizedCapacity: resource.MustParse("2Gi"),
				},
				"volume-2": {
					BoundCount:      3,
					CurrentCount:    1,
					ResizedCount:    2,
					CurrentCapacity: resource.MustParse("1Gi"),
					ResizedCapacity: resource.MustParse("2Gi"),
				},
			},
		},
		{
			name: "some volumes is resizing",
			setup: func(ctx *componentVolumeContext) {
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
					"volume-2": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = map[v1alpha1.StorageVolumeName][]*podVolumeContext{
					"volume-1": {
						{pvc: newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi")},
						{pvc: newMockPVC("volume-1-pvc-2", scName, "2Gi", "2Gi")},
						{pvc: newMockPVC("volume-1-pvc-3", scName, "2Gi", "1Gi")}, // not resized
					},
					"volume-2": {
						{pvc: newMockPVC("volume-2-pvc-1", scName, "2Gi", "1Gi")}, // not resized
						{pvc: newMockPVC("volume-2-pvc-2", scName, "2Gi", "1Gi")}, // not resized
						{pvc: newMockPVC("volume-2-pvc-3", scName, "2Gi", "2Gi")},
					},
				}
				volumes := ctx.actualPodVolumes["volume-2"]
				volumes[0].pvc.Status.Conditions = []v1.PersistentVolumeClaimCondition{
					{Type: v1.PersistentVolumeClaimResizing, Status: v1.ConditionTrue},
				}
				volumes[1].pvc.Status.Conditions = []v1.PersistentVolumeClaimCondition{
					{Type: v1.PersistentVolumeClaimFileSystemResizePending, Status: v1.ConditionTrue},
				}
			},
			expect: map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus{
				"volume-1": {
					BoundCount:      3,
					CurrentCount:    1,
					ResizedCount:    2,
					CurrentCapacity: resource.MustParse("1Gi"),
					ResizedCapacity: resource.MustParse("2Gi"),
				},
				"volume-2": {
					BoundCount:      3,
					ResizingCount:   2,
					CurrentCount:    2,
					ResizedCount:    1,
					CurrentCapacity: resource.MustParse("1Gi"),
					ResizedCapacity: resource.MustParse("2Gi"),
				},
			},
		},
		{
			name: "all volumes is resized",
			setup: func(ctx *componentVolumeContext) {
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
					"volume-2": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = map[v1alpha1.StorageVolumeName][]*podVolumeContext{
					"volume-1": {
						{pvc: newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi")},
						{pvc: newMockPVC("volume-1-pvc-2", scName, "2Gi", "2Gi")},
						{pvc: newMockPVC("volume-1-pvc-3", scName, "2Gi", "2Gi")},
					},
					"volume-2": {
						{pvc: newMockPVC("volume-2-pvc-1", scName, "2Gi", "2Gi")},
						{pvc: newMockPVC("volume-2-pvc-2", scName, "2Gi", "2Gi")},
						{pvc: newMockPVC("volume-2-pvc-3", scName, "2Gi", "2Gi")},
					},
				}
			},
			expect: map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus{
				"volume-1": {
					BoundCount:      3,
					CurrentCount:    3,
					ResizedCount:    3,
					CurrentCapacity: resource.MustParse("2Gi"),
					ResizedCapacity: resource.MustParse("2Gi"),
				},
				"volume-2": {
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

			ctx := &componentVolumeContext{}
			if tt.setup != nil {
				tt.setup(ctx)
			}

			observedStatus := resizer.assembleObserveVolumeStatus(ctx)
			diff := cmp.Diff(observedStatus, tt.expect)
			g.Expect(diff).Should(gomega.BeEmpty(), "unexpected (-want, +got)")
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
		volumeNames = append(volumeNames, string(v1alpha1.GetStorageVolumeName("", comp)))
	}
	for _, sv := range storageVolumes {
		volumeNames = append(volumeNames, string(v1alpha1.GetStorageVolumeName(sv.Name, comp)))
	}
	for i := range storageClaims {
		volumeNames = append(volumeNames, string(v1alpha1.GetStorageVolumeNameForTiFlash(i)))
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
