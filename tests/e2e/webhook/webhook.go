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

package webhook

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

// TestCase represents a test case for volume storage validation
type TestCase struct {
	Description     string
	GroupType       string
	StorageValue    string
	ShouldFail      bool
	ExpectedError   string
	CreateGroupFunc func(ns string, storage string) (client.Object, error)
}

// UpdateTestCase represents a test case for storage update validation
type UpdateTestCase struct {
	Description       string
	GroupType         string
	InitialStorage    string
	UpdatedStorage    string
	ShouldFail        bool
	ExpectedError     string
	CreateGroupFunc   func(ns string, storage string) (client.Object, error)
	UpdateStorageFunc func(obj client.Object, storage string) error
}

var _ = ginkgo.Describe("ValidatingAdmissionPolicy", label.P0, func() {
	f := framework.New()
	f.Setup(framework.WithSkipClusterCreation())

	ginkgo.Context("Volume Storage Validation", func() {
		ginkgo.DescribeTableSubtree("Storage value validation for CREATE operations",
			func(testCase TestCase) {
				ginkgo.It(fmt.Sprintf("should validate %s", testCase.Description), func(ctx context.Context) {
					obj, err := testCase.CreateGroupFunc(f.Namespace.Name, testCase.StorageValue)
					gomega.Expect(err).ToNot(gomega.HaveOccurred())

					ginkgo.By(fmt.Sprintf("Creating %s with storage %s", testCase.GroupType, testCase.StorageValue))
					err = f.Client.Create(ctx, obj)

					if testCase.ShouldFail {
						gomega.Expect(err).To(gomega.HaveOccurred())
						gomega.Expect(errors.IsInvalid(err)).To(gomega.BeTrue())
						if testCase.ExpectedError != "" {
							gomega.Expect(err.Error()).To(gomega.ContainSubstring(testCase.ExpectedError))
						}
					} else {
						gomega.Expect(err).ToNot(gomega.HaveOccurred())

						// Wait for object to be created
						err = waiter.WaitForObject(ctx, f.Client, obj, func() error {
							return nil
						}, waiter.ShortTaskTimeout)
						gomega.Expect(err).ToNot(gomega.HaveOccurred())

						// Clean up
						gomega.Expect(f.Client.Delete(ctx, obj)).To(gomega.Succeed())
					}
				})
			},
			ginkgo.Entry("TiKV group with zero storage", TestCase{
				Description:   "TiKV group with zero storage should be rejected",
				GroupType:     "TiKVGroup",
				StorageValue:  "0Gi",
				ShouldFail:    true,
				ExpectedError: "Volume storage must be greater than 0",
				CreateGroupFunc: func(ns string, storage string) (client.Object, error) {
					return data.NewTiKVGroup(ns, data.GroupPatchFunc[*v1alpha1.TiKVGroup](func(group *v1alpha1.TiKVGroup) {
						group.Spec.Template.Spec.Volumes = []v1alpha1.Volume{
							{
								Name:    "data",
								Storage: resource.MustParse(storage),
								Mounts:  []v1alpha1.VolumeMount{{Type: "data"}},
							},
						}
					})), nil
				},
			}),
			ginkgo.Entry("TiDB group with negative storage", TestCase{
				Description:   "TiDB group with negative storage should be rejected",
				GroupType:     "TiDBGroup",
				StorageValue:  "-1Gi",
				ShouldFail:    true,
				ExpectedError: "Volume storage must be greater than 0",
				CreateGroupFunc: func(ns string, storage string) (client.Object, error) {
					return data.NewTiDBGroup(ns, data.GroupPatchFunc[*v1alpha1.TiDBGroup](func(group *v1alpha1.TiDBGroup) {
						group.Spec.Template.Spec.Volumes = []v1alpha1.Volume{
							{
								Name:    "log",
								Storage: resource.MustParse(storage),
								Mounts:  []v1alpha1.VolumeMount{{Type: "log"}},
							},
						}
					})), nil
				},
			}),
			ginkgo.Entry("PD group with valid positive storage", TestCase{
				Description:   "PD group with valid positive storage should be accepted",
				GroupType:     "PDGroup",
				StorageValue:  "10Gi",
				ShouldFail:    false,
				ExpectedError: "",
				CreateGroupFunc: func(ns string, storage string) (client.Object, error) {
					return data.NewPDGroup(ns, data.GroupPatchFunc[*v1alpha1.PDGroup](func(group *v1alpha1.PDGroup) {
						group.Spec.Template.Spec.Volumes = []v1alpha1.Volume{
							{
								Name:    "data",
								Storage: resource.MustParse(storage),
								Mounts:  []v1alpha1.VolumeMount{{Type: "data"}},
							},
						}
					})), nil
				},
			}),
			ginkgo.Entry("TSO group with zero storage", TestCase{
				Description:   "TSO group with zero storage should be rejected",
				GroupType:     "TSOGroup",
				StorageValue:  "0Gi",
				ShouldFail:    true,
				ExpectedError: "Volume storage must be greater than 0",
				CreateGroupFunc: func(ns string, storage string) (client.Object, error) {
					return data.NewTSOGroup(ns, data.GroupPatchFunc[*v1alpha1.TSOGroup](func(group *v1alpha1.TSOGroup) {
						group.Spec.Template.Spec.Volumes = []v1alpha1.Volume{
							{
								Name:    "data",
								Storage: resource.MustParse(storage),
								Mounts:  []v1alpha1.VolumeMount{{Type: "data"}},
							},
						}
					})), nil
				},
			}),
			ginkgo.Entry("TiProxy group with zero storage", TestCase{
				Description:   "TiProxy group with zero storage should be rejected",
				GroupType:     "TiProxyGroup",
				StorageValue:  "0Gi",
				ShouldFail:    true,
				ExpectedError: "Volume storage must be greater than 0",
				CreateGroupFunc: func(ns string, storage string) (client.Object, error) {
					return data.NewTiProxyGroup(ns, data.GroupPatchFunc[*v1alpha1.TiProxyGroup](func(group *v1alpha1.TiProxyGroup) {
						group.Spec.Template.Spec.Volumes = []v1alpha1.Volume{
							{
								Name:    "log",
								Storage: resource.MustParse(storage),
								Mounts:  []v1alpha1.VolumeMount{{Type: "log"}},
							},
						}
					})), nil
				},
			}),
		)

		ginkgo.DescribeTableSubtree("Storage value validation for UPDATE operations",
			func(testCase UpdateTestCase) {
				ginkgo.It(fmt.Sprintf("should validate %s", testCase.Description), func(ctx context.Context) {
					// Create object with initial storage
					obj, err := testCase.CreateGroupFunc(f.Namespace.Name, testCase.InitialStorage)
					gomega.Expect(err).ToNot(gomega.HaveOccurred())

					ginkgo.By(fmt.Sprintf("Creating %s with initial storage %s", testCase.GroupType, testCase.InitialStorage))
					err = f.Client.Create(ctx, obj)
					gomega.Expect(err).ToNot(gomega.HaveOccurred())

					// Wait for object to be created
					err = waiter.WaitForObject(ctx, f.Client, obj, func() error {
						return nil
					}, waiter.ShortTaskTimeout)
					gomega.Expect(err).ToNot(gomega.HaveOccurred())

					ginkgo.By(fmt.Sprintf("Updating storage from %s to %s", testCase.InitialStorage, testCase.UpdatedStorage))
					// Get the current object
					key := client.ObjectKeyFromObject(obj)
					err = f.Client.Get(ctx, key, obj)
					gomega.Expect(err).ToNot(gomega.HaveOccurred())

					// Update storage
					err = testCase.UpdateStorageFunc(obj, testCase.UpdatedStorage)
					gomega.Expect(err).ToNot(gomega.HaveOccurred())

					// Try to update the object
					err = f.Client.Update(ctx, obj)

					if testCase.ShouldFail {
						gomega.Expect(err).To(gomega.HaveOccurred())
						gomega.Expect(errors.IsInvalid(err)).To(gomega.BeTrue())
						if testCase.ExpectedError != "" {
							gomega.Expect(err.Error()).To(gomega.ContainSubstring(testCase.ExpectedError))
						}
					} else {
						gomega.Expect(err).ToNot(gomega.HaveOccurred())

						// Verify the update
						err = f.Client.Get(ctx, key, obj)
						gomega.Expect(err).ToNot(gomega.HaveOccurred())
					}

					// Clean up - get the original object for deletion
					err = f.Client.Get(ctx, key, obj)
					if err == nil {
						gomega.Expect(f.Client.Delete(ctx, obj)).To(gomega.Succeed())
					}
				})
			},
			ginkgo.Entry("TiKV storage decrease should be rejected", UpdateTestCase{
				Description:    "decreasing TiKV storage should be rejected",
				GroupType:      "TiKVGroup",
				InitialStorage: "20Gi",
				UpdatedStorage: "10Gi",
				ShouldFail:     true,
				ExpectedError:  "Volume storage can only be increased, not decreased",
				CreateGroupFunc: func(ns string, storage string) (client.Object, error) {
					return data.NewTiKVGroup(ns, data.GroupPatchFunc[*v1alpha1.TiKVGroup](func(group *v1alpha1.TiKVGroup) {
						group.Spec.Template.Spec.Volumes = []v1alpha1.Volume{
							{
								Name:    "data",
								Storage: resource.MustParse(storage),
								Mounts:  []v1alpha1.VolumeMount{{Type: "data"}},
							},
						}
					})), nil
				},
				UpdateStorageFunc: func(obj client.Object, storage string) error {
					tikvGroup := obj.(*v1alpha1.TiKVGroup)
					tikvGroup.Spec.Template.Spec.Volumes[0].Storage = resource.MustParse(storage)
					return nil
				},
			}),
			ginkgo.Entry("TiFlash storage increase should be accepted", UpdateTestCase{
				Description:    "increasing TiFlash storage should be accepted",
				GroupType:      "TiFlashGroup",
				InitialStorage: "15Gi",
				UpdatedStorage: "25Gi",
				ShouldFail:     false,
				ExpectedError:  "",
				CreateGroupFunc: func(ns string, storage string) (client.Object, error) {
					return data.NewTiFlashGroup(ns, data.GroupPatchFunc[*v1alpha1.TiFlashGroup](func(group *v1alpha1.TiFlashGroup) {
						group.Spec.Template.Spec.Volumes = []v1alpha1.Volume{
							{
								Name:    "data",
								Storage: resource.MustParse(storage),
								Mounts:  []v1alpha1.VolumeMount{{Type: "data"}},
							},
						}
					})), nil
				},
				UpdateStorageFunc: func(obj client.Object, storage string) error {
					tiFlashGroup := obj.(*v1alpha1.TiFlashGroup)
					tiFlashGroup.Spec.Template.Spec.Volumes[0].Storage = resource.MustParse(storage)
					return nil
				},
			}),
			ginkgo.Entry("PD storage same value should be accepted", UpdateTestCase{
				Description:    "keeping PD storage the same should be accepted",
				GroupType:      "PDGroup",
				InitialStorage: "5Gi",
				UpdatedStorage: "5Gi",
				ShouldFail:     false,
				ExpectedError:  "",
				CreateGroupFunc: func(ns string, storage string) (client.Object, error) {
					return data.NewPDGroup(ns, data.GroupPatchFunc[*v1alpha1.PDGroup](func(group *v1alpha1.PDGroup) {
						group.Spec.Template.Spec.Volumes = []v1alpha1.Volume{
							{
								Name:    "data",
								Storage: resource.MustParse(storage),
								Mounts:  []v1alpha1.VolumeMount{{Type: "data"}},
							},
						}
					})), nil
				},
				UpdateStorageFunc: func(obj client.Object, storage string) error {
					pdGroup := obj.(*v1alpha1.PDGroup)
					pdGroup.Spec.Template.Spec.Volumes[0].Storage = resource.MustParse(storage)
					return nil
				},
			}),
		)

		ginkgo.Describe("Special scenarios", func() {
			ginkgo.It("should validate all volumes in a group with multiple volumes", func(ctx context.Context) {
				// Create TiKV group with multiple volumes, one invalid
				invalidTiKVGroup := data.NewTiKVGroup(f.Namespace.Name, data.GroupPatchFunc[*v1alpha1.TiKVGroup](func(group *v1alpha1.TiKVGroup) {
					group.Spec.Template.Spec.Volumes = []v1alpha1.Volume{
						{
							Name:    "data",
							Storage: resource.MustParse("10Gi"), // Valid
							Mounts:  []v1alpha1.VolumeMount{{Type: "data"}},
						},
						{
							Name:    "log",
							Storage: resource.MustParse("0Gi"), // Invalid: zero storage
							Mounts:  []v1alpha1.VolumeMount{{Type: "log"}},
						},
					}
				}))

				ginkgo.By("Creating TiKV group with mixed valid/invalid volumes should fail")
				err := f.Client.Create(ctx, invalidTiKVGroup)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(errors.IsInvalid(err)).To(gomega.BeTrue())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("Volume storage must be greater than 0"))
			})

			ginkgo.It("should handle adding new volumes during update", func(ctx context.Context) {
				// Create TiKV group with single volume
				tikvGroup := data.NewTiKVGroup(f.Namespace.Name, data.GroupPatchFunc[*v1alpha1.TiKVGroup](func(group *v1alpha1.TiKVGroup) {
					group.Spec.Template.Spec.Volumes = []v1alpha1.Volume{
						{
							Name:    "data",
							Storage: resource.MustParse("10Gi"),
							Mounts:  []v1alpha1.VolumeMount{{Type: "data"}},
						},
					}
				}))

				ginkgo.By("Creating TiKV group with single volume")
				err := f.Client.Create(ctx, tikvGroup)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				err = waiter.WaitForObject(ctx, f.Client, tikvGroup, func() error {
					return nil
				}, waiter.ShortTaskTimeout)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				ginkgo.By("Adding a new volume should succeed")
				key := client.ObjectKeyFromObject(tikvGroup)
				err = f.Client.Get(ctx, key, tikvGroup)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				// Add a new volume
				tikvGroup.Spec.Template.Spec.Volumes = append(tikvGroup.Spec.Template.Spec.Volumes, v1alpha1.Volume{
					Name:    "log",
					Storage: resource.MustParse("5Gi"), // Valid new volume
					Mounts:  []v1alpha1.VolumeMount{{Type: "log"}},
				})

				err = f.Client.Update(ctx, tikvGroup)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				ginkgo.By("Cleaning up the TiKV group")
				gomega.Expect(f.Client.Delete(ctx, tikvGroup)).To(gomega.Succeed())
			})

			ginkgo.It("should allow resources without volumes field", func(ctx context.Context) {
				dbg := data.NewTiDBGroup(f.Namespace.Name)
				err := f.Client.Create(ctx, dbg)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(f.Client.Delete(ctx, dbg)).To(gomega.Succeed())
			})
		})
	})
})
