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
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"context"
	"errors"
	"testing"

	azruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestModifyVolume(t *testing.T) {
	tests := []struct {
		name           string
		pvc            *corev1.PersistentVolumeClaim
		pv             *corev1.PersistentVolume
		sc             *storagev1.StorageClass
		mockDiskClient *MockDiskClient
		expectedWait   bool
		expectedError  error
	}{
		{
			name: "successful modification",
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "default",
					Name:      "pvc1",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
				},
			},
			pv: &corev1.PersistentVolume{
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							VolumeHandle: "/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/disks/disk1",
						},
					},
				},
			},
			sc: &storagev1.StorageClass{
				Parameters: map[string]string{
					paramKeyThroughput: "100",
					paramKeyIOPS:       "500",
					paramKeyType:       "Premium_LRS",
				},
			},
			mockDiskClient: &MockDiskClient{
				GetFunc: func(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientGetOptions) (armcompute.DisksClientGetResponse, error) {
					return armcompute.DisksClientGetResponse{
						Disk: armcompute.Disk{
							ID: to.Ptr("/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/disks/disk1"),
							Properties: &armcompute.DiskProperties{
								DiskSizeGB:        ptr.To[int32](10),
								DiskIOPSReadWrite: ptr.To[int64](500),
								DiskMBpsReadWrite: ptr.To[int64](100),
							},
							SKU: &armcompute.DiskSKU{
								Name: to.Ptr(armcompute.DiskStorageAccountTypesPremiumLRS),
							},
						},
					}, nil
				},
				BeginUpdateFunc: func(ctx context.Context, resourceGroupName string, diskName string, parameters armcompute.DiskUpdate, options *armcompute.DisksClientBeginUpdateOptions) (*azruntime.Poller[armcompute.DisksClientUpdateResponse], error) {
					return nil, nil
				},
			},
			expectedWait:  false,
			expectedError: nil,
		},
		{
			name: "failed to get disk info",
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "default",
					Name:      "pvc1",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
				},
			},
			pv: &corev1.PersistentVolume{
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							VolumeHandle: "invalid/volume/handle",
						},
					},
				},
			},
			sc: &storagev1.StorageClass{
				Parameters: map[string]string{
					paramKeyThroughput: "100",
					paramKeyIOPS:       "500",
					paramKeyType:       "Premium_LRS",
				},
			},
			mockDiskClient: &MockDiskClient{},
			expectedWait:   false,
			expectedError:  errors.New("failed to get Azure disk info from PV: invalid volumeHandle format"),
		},
		{
			name: "volume modification is failed, modify again",
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "default",
					Name:      "pvc1",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
				},
			},
			pv: &corev1.PersistentVolume{
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							VolumeHandle: "/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/disks/disk1",
						},
					},
				},
			},
			sc: &storagev1.StorageClass{
				Parameters: map[string]string{
					paramKeyThroughput: "100",
					paramKeyIOPS:       "500",
					paramKeyType:       "Premium_LRS",
				},
			},
			mockDiskClient: &MockDiskClient{
				GetFunc: func(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientGetOptions) (armcompute.DisksClientGetResponse, error) {
					return armcompute.DisksClientGetResponse{
						Disk: armcompute.Disk{
							ID: to.Ptr("/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/disks/disk1"),
							Properties: &armcompute.DiskProperties{
								DiskSizeGB:        ptr.To[int32](20),
								DiskIOPSReadWrite: ptr.To[int64](500),
								DiskMBpsReadWrite: ptr.To[int64](100),
							},
							SKU: &armcompute.DiskSKU{
								Name: to.Ptr(armcompute.DiskStorageAccountTypesPremiumLRS),
							},
						},
					}, nil
				},
				BeginUpdateFunc: func(ctx context.Context, resourceGroupName string, diskName string, parameters armcompute.DiskUpdate, options *armcompute.DisksClientBeginUpdateOptions) (*azruntime.Poller[armcompute.DisksClientUpdateResponse], error) {
					return nil, errors.New("begin update failed, please try again")
				},
			},
			expectedWait:  false,
			expectedError: errors.New("failed to update disk: begin update failed, please try again"),
		},
		{
			name: "volume modification is completed, no need to wait",
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "default",
					Name:      "pvc1",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
				},
			},
			pv: &corev1.PersistentVolume{
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							VolumeHandle: "/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/disks/disk1",
						},
					},
				},
			},
			sc: &storagev1.StorageClass{
				Parameters: map[string]string{
					paramKeyThroughput: "100",
					paramKeyIOPS:       "500",
					paramKeyType:       "Premium_LRS",
				},
			},
			mockDiskClient: &MockDiskClient{
				GetFunc: func(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientGetOptions) (armcompute.DisksClientGetResponse, error) {
					return armcompute.DisksClientGetResponse{
						Disk: armcompute.Disk{
							ID: to.Ptr("/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/disks/disk1"),
							Properties: &armcompute.DiskProperties{
								DiskSizeGB:        ptr.To[int32](10),
								DiskIOPSReadWrite: ptr.To[int64](500),
								DiskMBpsReadWrite: ptr.To[int64](100),
							},
							SKU: &armcompute.DiskSKU{
								Name: to.Ptr(armcompute.DiskStorageAccountTypesPremiumLRS),
							},
						},
					}, nil
				},
			},
			expectedWait:  false,
			expectedError: nil,
		},
		{
			name: "volume has not been modified, try to modify",
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "default",
					Name:      "pvc1",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
				},
			},
			pv: &corev1.PersistentVolume{
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							VolumeHandle: "/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/disks/disk1",
						},
					},
				},
			},
			sc: &storagev1.StorageClass{
				Parameters: map[string]string{
					paramKeyThroughput: "100",
					paramKeyIOPS:       "500",
					paramKeyType:       "Premium_LRS",
				},
			},
			mockDiskClient: &MockDiskClient{
				GetFunc: func(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientGetOptions) (armcompute.DisksClientGetResponse, error) {
					return armcompute.DisksClientGetResponse{
						Disk: armcompute.Disk{
							ID: to.Ptr("/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/disks/disk1"),
							Properties: &armcompute.DiskProperties{
								DiskSizeGB:        ptr.To[int32](5),
								DiskIOPSReadWrite: ptr.To[int64](300),
								DiskMBpsReadWrite: ptr.To[int64](50),
							},
							SKU: &armcompute.DiskSKU{
								Name: to.Ptr(armcompute.DiskStorageAccountTypesPremiumLRS),
							},
						},
					}, nil
				},
				BeginUpdateFunc: func(ctx context.Context, resourceGroupName string, diskName string, parameters armcompute.DiskUpdate, options *armcompute.DisksClientBeginUpdateOptions) (*azruntime.Poller[armcompute.DisksClientUpdateResponse], error) {
					return nil, nil
				},
			},
			expectedWait:  true,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifier := &AzureDiskModifier{
				DiskClient: tt.mockDiskClient,
			}
			wait, err := modifier.ModifyVolume(context.TODO(), tt.pvc, tt.pv, tt.sc)
			if wait != tt.expectedWait {
				t.Errorf("expected wait %v, got %v", tt.expectedWait, wait)
			}
			if err != nil && tt.expectedError == nil {
				t.Errorf("unexpected error: %v", err)
			}
			if err == nil && tt.expectedError != nil {
				t.Errorf("expected error: %v, got nil", tt.expectedError)
			}
			if err != nil && tt.expectedError != nil {
				if diff := cmp.Diff(tt.expectedError.Error(), err.Error()); diff != "" {
					t.Errorf("error mismatch (-expected +got):\n%s", diff)
				} else {
					t.Logf("error messages match: %v", err)
				}
			}
		})
	}
}

type MockDiskClient struct {
	GetFunc         func(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientGetOptions) (armcompute.DisksClientGetResponse, error)
	BeginUpdateFunc func(ctx context.Context, resourceGroupName string, diskName string, parameters armcompute.DiskUpdate, options *armcompute.DisksClientBeginUpdateOptions) (*azruntime.Poller[armcompute.DisksClientUpdateResponse], error)
}

func (m *MockDiskClient) Get(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientGetOptions) (armcompute.DisksClientGetResponse, error) {
	return m.GetFunc(ctx, resourceGroupName, diskName, options)
}

func (m *MockDiskClient) BeginUpdate(ctx context.Context, resourceGroupName string, diskName string, parameters armcompute.DiskUpdate, options *armcompute.DisksClientBeginUpdateOptions) (*azruntime.Poller[armcompute.DisksClientUpdateResponse], error) {
	return m.BeginUpdateFunc(ctx, resourceGroupName, diskName, parameters, options)
}
