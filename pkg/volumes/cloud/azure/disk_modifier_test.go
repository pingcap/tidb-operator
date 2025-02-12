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

package azure

import (
	"context"
	"errors"
	"testing"

	azruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/pkg/volumes/cloud"
)

func TestModifyVolume(t *testing.T) {
	sc := &storagev1.StorageClass{
		Parameters: map[string]string{
			paramKeyThroughput: "100",
			paramKeyIOPS:       "500",
			paramKeyType:       "Premium_LRS",
		},
	}

	tests := []struct {
		name           string
		pvc            *corev1.PersistentVolumeClaim
		pv             *corev1.PersistentVolume
		FakeDiskClient *FakeDiskClient
		expectedWait   bool
		expectedError  error
	}{
		{
			name: "successful modification",
			pvc:  cloud.NewTestPVC("10Gi"),
			pv:   cloud.NewTestPV("/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/disks/disk1"),
			FakeDiskClient: &FakeDiskClient{
				GetFunc: func(_ context.Context, _, _ string, _ *armcompute.DisksClientGetOptions) (armcompute.DisksClientGetResponse, error) {
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
				BeginUpdateFunc: func(_ context.Context, _, _ string, _ armcompute.DiskUpdate, _ *armcompute.DisksClientBeginUpdateOptions) (*azruntime.Poller[armcompute.DisksClientUpdateResponse], error) {
					return nil, nil
				},
			},
			expectedWait:  false,
			expectedError: nil,
		},
		{
			name:           "failed to get disk info",
			pvc:            cloud.NewTestPVC("10Gi"),
			pv:             cloud.NewTestPV("invalid/volume/handle"),
			FakeDiskClient: &FakeDiskClient{},
			expectedWait:   false,
			expectedError:  errors.New("failed to get Azure disk info from PV: invalid volumeHandle format"),
		},
		{
			name: "volume modification is failed, modify again",
			pvc:  cloud.NewTestPVC("10Gi"),
			pv:   cloud.NewTestPV("/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/disks/disk1"),
			FakeDiskClient: &FakeDiskClient{
				GetFunc: func(_ context.Context, _, _ string, _ *armcompute.DisksClientGetOptions) (armcompute.DisksClientGetResponse, error) {
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
				BeginUpdateFunc: func(_ context.Context, _, _ string, _ armcompute.DiskUpdate, _ *armcompute.DisksClientBeginUpdateOptions) (*azruntime.Poller[armcompute.DisksClientUpdateResponse], error) {
					return nil, errors.New("begin update failed, please try again")
				},
			},
			expectedWait:  false,
			expectedError: errors.New("failed to update disk: begin update failed, please try again"),
		},
		{
			name: "volume modification is completed, no need to wait",
			pvc:  cloud.NewTestPVC("10Gi"),
			pv:   cloud.NewTestPV("/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/disks/disk1"),
			FakeDiskClient: &FakeDiskClient{
				GetFunc: func(_ context.Context, _, _ string, _ *armcompute.DisksClientGetOptions) (armcompute.DisksClientGetResponse, error) {
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
			pvc:  cloud.NewTestPVC("10Gi"),
			pv:   cloud.NewTestPV("/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/disks/disk1"),
			FakeDiskClient: &FakeDiskClient{
				GetFunc: func(_ context.Context, _, _ string, _ *armcompute.DisksClientGetOptions) (armcompute.DisksClientGetResponse, error) {
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
				BeginUpdateFunc: func(_ context.Context, _, _ string, _ armcompute.DiskUpdate, _ *armcompute.DisksClientBeginUpdateOptions) (*azruntime.Poller[armcompute.DisksClientUpdateResponse], error) {
					return nil, nil
				},
			},
			expectedWait:  true,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifier := &DiskModifier{
				DiskClient: tt.FakeDiskClient,
				Logger:     logr.Discard(),
			}
			wait, err := modifier.Modify(context.TODO(), tt.pvc, tt.pv, sc)
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

func TestGetDiskInfoFromVolumeID(t *testing.T) {
	tests := []struct {
		name             string
		volumeID         string
		expectedDiskName string
		expectedSubID    string
		expectedRGName   string
		wantError        bool
	}{
		{
			name:             "valid volume ID",
			volumeID:         "/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/disks/disk1",
			expectedDiskName: "disk1",
			expectedSubID:    "123",
			expectedRGName:   "rg1",
		},
		{
			name:             "invalid volume ID format",
			volumeID:         "invalid/volume/handle",
			expectedDiskName: "",
			expectedSubID:    "",
			expectedRGName:   "",
			wantError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diskName, subID, rgName, err := getDiskInfoFromVolumeID(tt.volumeID)
			if (err != nil) != tt.wantError {
				t.Errorf("expected error %v, got %v", tt.wantError, err)
			}
			if diskName != tt.expectedDiskName {
				t.Errorf("expected diskName %v, got %v", tt.expectedDiskName, diskName)
			}
			if subID != tt.expectedSubID {
				t.Errorf("expected subscriptionID %v, got %v", tt.expectedSubID, subID)
			}
			if rgName != tt.expectedRGName {
				t.Errorf("expected resourceGroupName %v, got %v", tt.expectedRGName, rgName)
			}
		})
	}
}
