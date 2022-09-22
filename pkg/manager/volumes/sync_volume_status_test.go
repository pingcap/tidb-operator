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

package volumes

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestObserveVolumeStatus(t *testing.T) {
	desiredSC := "desired-sc"
	actualSC := "actual-sc"
	desiredSize := "20Gi"
	actualSize := "10Gi"

	type testcase struct {
		input  func(*FakePodVolumeModifier) ([]*v1.Pod, []DesiredVolume)
		expect func(*GomegaWithT, map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus)
	}

	cases := map[string]testcase{
		"all volumes are modified": {
			input: func(pvm *FakePodVolumeModifier) ([]*v1.Pod, []DesiredVolume) {
				pods := newPods("pod", 3)

				desiredVolumes := []DesiredVolume{
					{
						Name:         "vol1",
						Size:         resource.MustParse(desiredSize),
						StorageClass: newStorageClass(desiredSC, true),
					},
					{
						Name:         "vol2",
						Size:         resource.MustParse(desiredSize),
						StorageClass: newStorageClass(desiredSC, true),
					},
				}
				pvm.GetActualVolumesFunc = func(pod *corev1.Pod, vs []DesiredVolume) ([]ActualVolume, error) {
					index := strings.Split(pod.Name, "-")[1]
					// all volumes are modified
					return []ActualVolume{
						{
							Desired: &desiredVolumes[0],
							PVC:     newPVC(fmt.Sprintf("vol1-%s", index), desiredSC, desiredSize, desiredSize),
						},
						{
							Desired: &desiredVolumes[1],
							PVC:     newPVC(fmt.Sprintf("vol2-%s", index), desiredSC, desiredSize, desiredSize),
						},
					}, nil
				}

				return pods, desiredVolumes
			},
			expect: func(g *GomegaWithT, observedStatus map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus) {
				expectStatus := map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus{
					"vol1": {
						BoundCount:           3,
						CurrentCount:         3,
						ModifiedCount:        3,
						CurrentCapacity:      resource.MustParse(desiredSize),
						ModifiedCapacity:     resource.MustParse(desiredSize),
						CurrentStorageClass:  desiredSC,
						ModifiedStorageClass: desiredSC,
						ResizedCount:         3,
						ResizedCapacity:      resource.MustParse(desiredSize),
					},
					"vol2": {
						BoundCount:           3,
						CurrentCount:         3,
						ModifiedCount:        3,
						CurrentCapacity:      resource.MustParse(desiredSize),
						ModifiedCapacity:     resource.MustParse(desiredSize),
						CurrentStorageClass:  desiredSC,
						ModifiedStorageClass: desiredSC,
						ResizedCount:         3,
						ResizedCapacity:      resource.MustParse(desiredSize),
					},
				}

				g.Expect(cmp.Diff(expectStatus, observedStatus)).To(BeEmpty(), "(-want, +got)")
			},
		},
		"some volumes is modifying": {
			input: func(pvm *FakePodVolumeModifier) ([]*v1.Pod, []DesiredVolume) {
				pods := newPods("pod", 3)

				desiredVolumes := []DesiredVolume{
					{
						Name:         "vol1",
						Size:         resource.MustParse(desiredSize),
						StorageClass: newStorageClass(desiredSC, true),
					},
					{
						Name:         "vol2",
						Size:         resource.MustParse(desiredSize),
						StorageClass: newStorageClass(desiredSC, true),
					},
					{
						Name:         "vol3",
						Size:         resource.MustParse(desiredSize),
						StorageClass: newStorageClass(desiredSC, true),
					},
				}
				pvm.GetActualVolumesFunc = func(pod *corev1.Pod, vs []DesiredVolume) ([]ActualVolume, error) {
					index := strings.Split(pod.Name, "-")[1]

					switch index {
					case "0": // volumes of pod 0 are modified
						return []ActualVolume{
							{
								Desired: &desiredVolumes[0],
								PVC:     newPVC(fmt.Sprintf("vol1-%s", index), desiredSC, desiredSize, desiredSize),
							},
							{
								Desired: &desiredVolumes[1],
								PVC:     newPVC(fmt.Sprintf("vol2-%s", index), desiredSC, desiredSize, desiredSize),
							},
							{
								Desired: &desiredVolumes[2],
								PVC:     newPVC(fmt.Sprintf("vol3-%s", index), desiredSC, desiredSize, desiredSize),
							},
						}, nil
					default: // other volumes of pod 1 are modifying or not modified
						return []ActualVolume{
							{
								Desired: &desiredVolumes[0],
								PVC:     newPVC(fmt.Sprintf("vol1-%s", index), desiredSC, actualSize, actualSize), // size is not modified
							},
							{
								Desired: &desiredVolumes[1],
								PVC:     newPVC(fmt.Sprintf("vol2-%s", index), actualSC, desiredSize, desiredSize), // sc is not modified
							},
							{
								Desired: &desiredVolumes[2],
								PVC:     newPVC(fmt.Sprintf("vol3-%s", index), actualSC, actualSize, actualSize), // size and sc are not modified
							},
						}, nil
					}

				}

				return pods, desiredVolumes
			},
			expect: func(g *GomegaWithT, observedStatus map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus) {
				expectStatus := map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus{
					"vol1": {
						BoundCount:           3,
						CurrentCount:         2,
						ModifiedCount:        1,
						CurrentCapacity:      resource.MustParse(actualSize),
						ModifiedCapacity:     resource.MustParse(desiredSize),
						CurrentStorageClass:  desiredSC,
						ModifiedStorageClass: desiredSC,
						ResizedCount:         1,
						ResizedCapacity:      resource.MustParse(desiredSize),
					},
					"vol2": {
						BoundCount:           3,
						CurrentCount:         2,
						ModifiedCount:        1,
						CurrentCapacity:      resource.MustParse(desiredSize),
						ModifiedCapacity:     resource.MustParse(desiredSize),
						CurrentStorageClass:  actualSC,
						ModifiedStorageClass: desiredSC,
						ResizedCount:         1,
						ResizedCapacity:      resource.MustParse(desiredSize),
					},
					"vol3": {
						BoundCount:           3,
						CurrentCount:         2,
						ModifiedCount:        1,
						CurrentCapacity:      resource.MustParse(actualSize),
						ModifiedCapacity:     resource.MustParse(desiredSize),
						CurrentStorageClass:  actualSC,
						ModifiedStorageClass: desiredSC,
						ResizedCount:         1,
						ResizedCapacity:      resource.MustParse(desiredSize),
					},
				}

				g.Expect(cmp.Diff(expectStatus, observedStatus)).To(BeEmpty(), "(-want, +got)")
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			pvm := &FakePodVolumeModifier{}
			pods, desiredVolumes := c.input(pvm)
			observedStatus := observeVolumeStatus(pvm, pods, desiredVolumes)
			c.expect(g, observedStatus)
		})
	}
}

func newPods(baseName string, count int) []*v1.Pod {
	pods := make([]*v1.Pod, 0, count)
	for i := 0; i < count; i++ {
		pods = append(pods, &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%d", baseName, i),
				Namespace: "test",
			},
		})
	}
	return pods
}

func newPVC(name, storageClass, request, capacity string) *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: v1.NamespaceDefault,
			Name:      name,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse(request),
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

func newStorageClass(name string, expandable bool) *storagev1.StorageClass {
	return &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		AllowVolumeExpansion: pointer.BoolPtr(expandable),
	}
}
