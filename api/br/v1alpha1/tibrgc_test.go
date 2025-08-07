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

package v1alpha1

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestTiBRGCSpec_GetT2Strategy(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Run("should return T2 strategy when it exists", func(t *testing.T) {
		spec := newTiBRGCSpecWithStrategies()
		strategy := spec.GetT2Strategy()

		g.Expect(strategy).ToNot(BeNil())
		g.Expect(strategy.Name).To(Equal(TieredStorageStrategyNameToT2Storage))
		g.Expect(strategy.TimeThresholdDays).To(Equal(uint32(7)))
	})

	t.Run("should return nil when T2 strategy does not exist", func(t *testing.T) {
		spec := &TiBRGCSpec{
			GCStrategy: TiBRGCStrategy{
				Type: TiBRGCStrategyTypeTieredStorage,
				TieredStrategies: []TieredStorageStrategy{
					{
						Name:              TieredStorageStrategyNameToT3Storage,
						TimeThresholdDays: 30,
					},
				},
			},
		}
		strategy := spec.GetT2Strategy()

		g.Expect(strategy).To(BeNil())
	})

	t.Run("should return nil when no strategies exist", func(t *testing.T) {
		spec := &TiBRGCSpec{
			GCStrategy: TiBRGCStrategy{
				Type:             TiBRGCStrategyTypeTieredStorage,
				TieredStrategies: []TieredStorageStrategy{},
			},
		}
		strategy := spec.GetT2Strategy()

		g.Expect(strategy).To(BeNil())
	})
}

func TestTiBRGCSpec_GetT3Strategy(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Run("should return T3 strategy when it exists", func(t *testing.T) {
		spec := newTiBRGCSpecWithStrategies()
		strategy := spec.GetT3Strategy()

		g.Expect(strategy).ToNot(BeNil())
		g.Expect(strategy.Name).To(Equal(TieredStorageStrategyNameToT3Storage))
		g.Expect(strategy.TimeThresholdDays).To(Equal(uint32(30)))
	})

	t.Run("should return nil when T3 strategy does not exist", func(t *testing.T) {
		spec := &TiBRGCSpec{
			GCStrategy: TiBRGCStrategy{
				Type: TiBRGCStrategyTypeTieredStorage,
				TieredStrategies: []TieredStorageStrategy{
					{
						Name:              TieredStorageStrategyNameToT2Storage,
						TimeThresholdDays: 7,
					},
				},
			},
		}
		strategy := spec.GetT3Strategy()

		g.Expect(strategy).To(BeNil())
	})

	t.Run("should return nil when no strategies exist", func(t *testing.T) {
		spec := &TiBRGCSpec{
			GCStrategy: TiBRGCStrategy{
				Type:             TiBRGCStrategyTypeTieredStorage,
				TieredStrategies: []TieredStorageStrategy{},
			},
		}
		strategy := spec.GetT3Strategy()

		g.Expect(strategy).To(BeNil())
	})
}

func TestTiBRGCSpec_getStrategy(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Run("should return correct strategy for given name", func(t *testing.T) {
		spec := newTiBRGCSpecWithStrategies()

		// Test T2 strategy
		t2Strategy := spec.getStrategy(TieredStorageStrategyNameToT2Storage)
		g.Expect(t2Strategy).ToNot(BeNil())
		g.Expect(t2Strategy.Name).To(Equal(TieredStorageStrategyNameToT2Storage))
		g.Expect(t2Strategy.TimeThresholdDays).To(Equal(uint32(7)))

		// Test T3 strategy
		t3Strategy := spec.getStrategy(TieredStorageStrategyNameToT3Storage)
		g.Expect(t3Strategy).ToNot(BeNil())
		g.Expect(t3Strategy.Name).To(Equal(TieredStorageStrategyNameToT3Storage))
		g.Expect(t3Strategy.TimeThresholdDays).To(Equal(uint32(30)))
	})

	t.Run("should return nil for non-existent strategy", func(t *testing.T) {
		spec := newTiBRGCSpecWithStrategies()
		strategy := spec.getStrategy("non-existent-strategy")

		g.Expect(strategy).To(BeNil())
	})

	t.Run("should return nil when strategies slice is empty", func(t *testing.T) {
		spec := &TiBRGCSpec{
			GCStrategy: TiBRGCStrategy{
				Type:             TiBRGCStrategyTypeTieredStorage,
				TieredStrategies: []TieredStorageStrategy{},
			},
		}
		strategy := spec.getStrategy(TieredStorageStrategyNameToT2Storage)

		g.Expect(strategy).To(BeNil())
	})
}

func TestTiBRGCSpec_GetPVCOverlay(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Run("should return PVC overlay when it exists", func(t *testing.T) {
		spec := newTiBRGCSpecWithOverlays()
		pvcOverlay := spec.GetPVCOverlay(TieredStorageStrategyNameToT2Storage, "test-pvc")

		g.Expect(pvcOverlay).ToNot(BeNil())
		g.Expect(pvcOverlay.Spec.StorageClassName).ToNot(BeNil())
		g.Expect(*pvcOverlay.Spec.StorageClassName).To(Equal("fast-ssd"))
	})

	t.Run("should return nil when PVC overlay does not exist", func(t *testing.T) {
		spec := newTiBRGCSpecWithOverlays()
		pvcOverlay := spec.GetPVCOverlay(TieredStorageStrategyNameToT2Storage, "non-existent-pvc")

		g.Expect(pvcOverlay).To(BeNil())
	})

	t.Run("should return nil when strategy overlay does not exist", func(t *testing.T) {
		spec := newTiBRGCSpecWithOverlays()
		pvcOverlay := spec.GetPVCOverlay("non-existent-strategy", "test-pvc")

		g.Expect(pvcOverlay).To(BeNil())
	})

	t.Run("should return nil when overlay is nil", func(t *testing.T) {
		spec := &TiBRGCSpec{
			Overlay: TiBRGCOverlay{
				Pods: []TiBRGCPodOverlay{
					{
						Name:    TieredStorageStrategyNameToT2Storage,
						Overlay: nil,
					},
				},
			},
		}
		pvcOverlay := spec.GetPVCOverlay(TieredStorageStrategyNameToT2Storage, "test-pvc")

		g.Expect(pvcOverlay).To(BeNil())
	})

	t.Run("should return nil when PersistentVolumeClaims is nil", func(t *testing.T) {
		spec := &TiBRGCSpec{
			Overlay: TiBRGCOverlay{
				Pods: []TiBRGCPodOverlay{
					{
						Name: TieredStorageStrategyNameToT2Storage,
						Overlay: &v1alpha1.Overlay{
							PersistentVolumeClaims: nil,
						},
					},
				},
			},
		}
		pvcOverlay := spec.GetPVCOverlay(TieredStorageStrategyNameToT2Storage, "test-pvc")

		g.Expect(pvcOverlay).To(BeNil())
	})
}

func TestTiBRGCSpec_GetOverlay(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Run("should return overlay when it exists", func(t *testing.T) {
		spec := newTiBRGCSpecWithOverlays()
		overlay := spec.GetOverlay(TieredStorageStrategyNameToT2Storage)

		g.Expect(overlay).ToNot(BeNil())
		g.Expect(overlay.PersistentVolumeClaims).ToNot(BeNil())
		g.Expect(len(overlay.PersistentVolumeClaims)).To(Equal(1))
	})

	t.Run("should return nil when overlay does not exist", func(t *testing.T) {
		spec := newTiBRGCSpecWithOverlays()
		overlay := spec.GetOverlay("non-existent-strategy")

		g.Expect(overlay).To(BeNil())
	})

	t.Run("should return nil when pods slice is empty", func(t *testing.T) {
		spec := &TiBRGCSpec{
			Overlay: TiBRGCOverlay{
				Pods: []TiBRGCPodOverlay{},
			},
		}
		overlay := spec.GetOverlay(TieredStorageStrategyNameToT2Storage)

		g.Expect(overlay).To(BeNil())
	})

	t.Run("should return overlay even when it's nil", func(t *testing.T) {
		spec := &TiBRGCSpec{
			Overlay: TiBRGCOverlay{
				Pods: []TiBRGCPodOverlay{
					{
						Name:    TieredStorageStrategyNameToT2Storage,
						Overlay: nil,
					},
				},
			},
		}
		overlay := spec.GetOverlay(TieredStorageStrategyNameToT2Storage)

		g.Expect(overlay).To(BeNil())
	})
}

// Helper functions for creating test data

func newTiBRGCSpecWithStrategies() *TiBRGCSpec {
	return &TiBRGCSpec{
		GCStrategy: TiBRGCStrategy{
			Type: TiBRGCStrategyTypeTieredStorage,
			TieredStrategies: []TieredStorageStrategy{
				{
					Name:              TieredStorageStrategyNameToT2Storage,
					TimeThresholdDays: 7,
					Schedule:          "0 2 * * *",
					Resources: v1alpha1.ResourceRequirements{
						CPU:    &[]resource.Quantity{resource.MustParse("100m")}[0],
						Memory: &[]resource.Quantity{resource.MustParse("128Mi")}[0],
					},
				},
				{
					Name:              TieredStorageStrategyNameToT3Storage,
					TimeThresholdDays: 30,
					Schedule:          "0 3 * * *",
					Resources: v1alpha1.ResourceRequirements{
						CPU:    &[]resource.Quantity{resource.MustParse("200m")}[0],
						Memory: &[]resource.Quantity{resource.MustParse("256Mi")}[0],
					},
				},
			},
		},
	}
}

func newTiBRGCSpecWithOverlays() *TiBRGCSpec {
	storageClassName := "fast-ssd"

	return &TiBRGCSpec{
		Overlay: TiBRGCOverlay{
			Pods: []TiBRGCPodOverlay{
				{
					Name: TieredStorageStrategyNameToT2Storage,
					Overlay: &v1alpha1.Overlay{
						PersistentVolumeClaims: []v1alpha1.NamedPersistentVolumeClaimOverlay{
							{
								Name: "test-pvc",
								PersistentVolumeClaim: v1alpha1.PersistentVolumeClaimOverlay{
									Spec: &corev1.PersistentVolumeClaimSpec{
										StorageClassName: &storageClassName,
										Resources: corev1.VolumeResourceRequirements{
											Requests: corev1.ResourceList{
												corev1.ResourceStorage: resource.MustParse("100Gi"),
											},
										},
									},
								},
							},
						},
					},
				},
				{
					Name: TieredStorageStrategyNameToT3Storage,
					Overlay: &v1alpha1.Overlay{
						PersistentVolumeClaims: []v1alpha1.NamedPersistentVolumeClaimOverlay{
							{
								Name: "test-pvc-t3",
								PersistentVolumeClaim: v1alpha1.PersistentVolumeClaimOverlay{
									Spec: &corev1.PersistentVolumeClaimSpec{
										StorageClassName: &storageClassName,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
