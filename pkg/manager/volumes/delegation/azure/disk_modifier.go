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
	"fmt"
	"strconv"
	"strings"
	"time"

	azruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	klog "k8s.io/klog/v2"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/pkg/manager/volumes/delegation"
)

var defaultWaitDuration = time.Hour * 6

const (
	paramKeyThroughput = "DiskMBpsReadWrite"
	paramKeyIOPS       = "DiskIOPSReadWrite"
	paramKeyType       = "skuName"

	maxSize = 32767 // Azure Disk max size in GiB
	minSize = 1
)

type AzureDiskModifier struct {
	// for unit test, add switch for fake client
	// because we need to get subscription id from volume ID, so we cannot initialize it in constructor
	DiskClient DiskClient
}

type DiskClient interface {
	Get(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientGetOptions) (armcompute.DisksClientGetResponse, error)
	BeginUpdate(ctx context.Context, resourceGroupName string, diskName string, parameters armcompute.DiskUpdate, options *armcompute.DisksClientBeginUpdateOptions) (*azruntime.Poller[armcompute.DisksClientUpdateResponse], error)
}

type Volume struct {
	VolumeId   string
	Size       *int32
	IOPS       *int64
	Throughput *int64
	Type       string
}

func NewAzureDiskModifier() delegation.VolumeModifier {
	return &AzureDiskModifier{}
}

func (m *AzureDiskModifier) Name() string {
	return "disk.csi.azure.com"
}

func (m *AzureDiskModifier) Validate(spvc, dpvc *corev1.PersistentVolumeClaim, ssc, dsc *storagev1.StorageClass) error {
	if ssc.Provisioner != dsc.Provisioner {
		return fmt.Errorf("provisioner should not be changed, now from %s to %s", ssc.Provisioner, dsc.Provisioner)
	}
	return nil
}

func (m *AzureDiskModifier) ModifyVolume(ctx context.Context, pvc *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume, sc *storagev1.StorageClass) ( /*wait*/ bool, error) {
	klog.V(4).Infof("Starting ModifyVolume for PVC %s/%s", pvc.Namespace, pvc.Name)

	if pv == nil {
		klog.V(4).Infof("Persistent volume is nil, skip modifying PV for %s. This may be caused by no relevant permissions", pvc.Spec.VolumeName)
		return false, nil
	}
	// Getting expected volume for PVC
	desired, err := m.getExpectedVolume(pvc, pv, sc)
	if err != nil {
		return false, fmt.Errorf("error getting expected volume: %w", err)
	}

	diskName, subscriptionID, resourceGroupName, err := getDiskInfoFromVolumeID(desired.VolumeId)
	if err != nil {
		return false, fmt.Errorf("failed to get Azure disk info from PV: %w", err)
	}

	if err := m.setDiskClientForPV(subscriptionID); err != nil {
		return false, fmt.Errorf("failed to create disk client: %w", err)
	}

	// Getting current volume status for PVC
	actual, err := m.getCurrentVolumeStatus(ctx, desired.VolumeId)
	if err != nil {
		return false, fmt.Errorf("getting current volume status: %w", err)
	}

	if actual != nil {
		if !m.diffVolume(actual, desired) {
			// if actual volume is equal to desired volume, return completed
			klog.V(4).Infof("Volume modification is already completed for PVC %s/%s", pvc.Namespace, pvc.Name)
			return false, nil
		}
	}

	klog.V(4).Infof("Beginning disk update for PVC %s/%s", pvc.Namespace, pvc.Name)

	if _, err = m.DiskClient.BeginUpdate(ctx, resourceGroupName, diskName, armcompute.DiskUpdate{
		Properties: &armcompute.DiskUpdateProperties{
			DiskIOPSReadWrite: desired.IOPS,
			DiskSizeGB:        desired.Size,
			DiskMBpsReadWrite: desired.Throughput,
		},
	}, nil); err != nil {
		return false, fmt.Errorf("failed to update disk: %v", err)
	}

	klog.V(4).Infof("Successfully modified volume for PVC %s/%s", pvc.Namespace, pvc.Name)
	return true, nil
}

// if actual volume is equal to desired volume, return false
func (m *AzureDiskModifier) diffVolume(actual, desired *Volume) bool {
	if diffInt64(actual.IOPS, desired.IOPS) {
		return true
	}
	if diffInt64(actual.Throughput, desired.Throughput) {
		return true
	}
	if diffInt32(actual.Size, desired.Size) {
		return true
	}
	if actual.Type == "" || desired.Type == "" {
		return false
	}
	if actual.Type != desired.Type {
		return true
	}

	return false
}

func diffInt64(a, b *int64) bool {
	if a == nil || b == nil {
		return false
	}

	if *a == *b {
		return false
	}

	return true
}

func diffInt32(a, b *int32) bool {
	if a == nil || b == nil {
		return false
	}

	if *a == *b {
		return false
	}

	return true
}

func (m *AzureDiskModifier) getCurrentVolumeStatus(ctx context.Context, volumeID string) (*Volume, error) {
	diskName, _, resourceGroupName, err := getDiskInfoFromVolumeID(volumeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure disk info from PV: %v", err)
	}

	diskResponse, err := m.DiskClient.Get(ctx, resourceGroupName, diskName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk: %v", err)
	}

	disk := diskResponse.Disk
	return &Volume{
		VolumeId:   *disk.ID,
		Size:       disk.Properties.DiskSizeGB,
		IOPS:       disk.Properties.DiskIOPSReadWrite,
		Throughput: disk.Properties.DiskMBpsReadWrite,
		Type:       string(*disk.SKU.Name),
	}, nil
}

func (m *AzureDiskModifier) getExpectedVolume(pvc *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume, sc *storagev1.StorageClass) (*Volume, error) {
	v := Volume{}
	if err := utilerrors.NewAggregate([]error{
		m.setArgsFromPVC(&v, pvc),
		m.setArgsFromPV(&v, pv),
		m.setArgsFromStorageClass(&v, sc),
	}); err != nil {
		return nil, err
	}

	return &v, nil
}

func (m *AzureDiskModifier) MinWaitDuration() time.Duration {
	return defaultWaitDuration
}

func (m *AzureDiskModifier) setArgsFromPVC(v *Volume, pvc *corev1.PersistentVolumeClaim) error {
	size, err := getSizeFromPVC(pvc)
	if err != nil {
		return err
	}
	v.Size = ptr.To(int32(size))
	return nil
}

func getSizeFromPVC(pvc *corev1.PersistentVolumeClaim) (int64, error) {
	quantity := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	sizeBytes := quantity.ScaledValue(0)
	size := sizeBytes / 1024 / 1024 / 1024

	if size < minSize || size > maxSize {
		return 0, fmt.Errorf("invalid storage size: %v", quantity)
	}
	return size, nil
}

func (m *AzureDiskModifier) setArgsFromPV(v *Volume, pv *corev1.PersistentVolume) error {
	v.VolumeId = pv.Spec.CSI.VolumeHandle
	return nil
}

func (m *AzureDiskModifier) setArgsFromStorageClass(v *Volume, sc *storagev1.StorageClass) error {
	if sc == nil {
		return nil
	}
	throughput, err := getParamInt64(sc.Parameters, paramKeyThroughput)
	if err != nil {
		return err
	}
	v.Throughput = throughput

	iops, err := getParamInt64(sc.Parameters, paramKeyIOPS)
	if err != nil {
		return err
	}
	v.IOPS = iops

	v.Type = sc.Parameters[paramKeyType]
	return nil
}

func getParamInt64(params map[string]string, key string) (*int64, error) {
	str, ok := params[key]
	if !ok || str == "" {
		return nil, nil
	}
	param, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("can't parse %v param in storage class: %v", key, err)
	}

	return ptr.To(param), nil
}

func getDiskInfoFromVolumeID(volumeID string) (diskName string, subscriptionID string, resourceGroupName string, err error) {
	// get diskName, subscriptionID, resourceGroupName from volumeHandle
	// example: /subscriptions/xxxx/resourceGroups/xxxx/providers/Microsoft.Compute/disks/xxxx
	parts := strings.Split(volumeID, "/")
	if len(parts) != 9 {
		return "", "", "", fmt.Errorf("invalid volumeHandle format")
	}
	subscriptionID = parts[2]
	resourceGroupName = parts[4]
	diskName = parts[len(parts)-1]

	return diskName, subscriptionID, resourceGroupName, nil
}

func (m *AzureDiskModifier) setDiskClientForPV(subscriptionID string) error {
	if m.DiskClient != nil {
		return nil
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return fmt.Errorf("failed to obtain a credential: %v", err)
	}

	diskClient, err := armcompute.NewDisksClient(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create disk client: %v", err)
	}

	m.DiskClient = diskClient

	return nil
}
