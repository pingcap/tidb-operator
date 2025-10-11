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
	"fmt"
	"strconv"
	"strings"
	"time"

	azruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/pkg/utils"
	"github.com/pingcap/tidb-operator/pkg/volumes/cloud"
)

const (
	paramKeyThroughput = "DiskMBpsReadWrite"
	paramKeyIOPS       = "DiskIOPSReadWrite"
	paramKeyType       = "skuName"

	maxSize = 32767 // Azure Disk max size in GiB
	minSize = 1

	volumeIDPartsLength = 9

	defaultWaitDuration = 15 * time.Second
)

type DiskModifier struct {
	// for unit test, add switch for fake client
	// because we need to get subscription id from volume ID, so we cannot initialize it in constructor
	DiskClient DiskClient
	Logger     logr.Logger
}

type DiskClient interface {
	Get(
		ctx context.Context,
		resourceGroupName, diskName string,
		opts *armcompute.DisksClientGetOptions,
	) (armcompute.DisksClientGetResponse, error)

	BeginUpdate(ctx context.Context,
		resourceGroupName, diskName string,
		parameters armcompute.DiskUpdate,
		options *armcompute.DisksClientBeginUpdateOptions,
	) (*azruntime.Poller[armcompute.DisksClientUpdateResponse], error)
}

type Volume struct {
	VolumeID   string
	Size       *int32
	IOPS       *int64
	Throughput *int64
	Type       string
}

func NewDiskModifier(logger logr.Logger) cloud.VolumeModifier {
	return &DiskModifier{
		Logger: logger,
	}
}

func (*DiskModifier) Name() string {
	return "disk.csi.azure.com"
}

func (m *DiskModifier) Validate(_, _ *corev1.PersistentVolumeClaim, ssc, dsc *storagev1.StorageClass) error {
	if ssc != nil && dsc != nil {
		if ssc.Provisioner != dsc.Provisioner {
			return fmt.Errorf("provisioner should not be changed, now from %s to %s", ssc.Provisioner, dsc.Provisioner)
		}
		if ssc.Provisioner != m.Name() {
			return fmt.Errorf("provisioner should be %s, now is %s", m.Name(), ssc.Provisioner)
		}
	} else {
		m.Logger.Info("storage class is nil, skip validation")
	}
	return nil
}

func (m *DiskModifier) Modify(
	ctx context.Context,
	pvc *corev1.PersistentVolumeClaim,
	pv *corev1.PersistentVolume,
	sc *storagev1.StorageClass,
) ( /*wait*/ bool, error) {
	logger := m.Logger.WithValues("namespace", pvc.Namespace, "pvc", pvc.Name)
	logger.Info("Starting Modify for PVC")

	if pv == nil {
		m.Logger.Info("Persistent volume is nil, skip modifying PV. This may be caused by no relevant permissions", "pv", pvc.Spec.VolumeName)
		return false, nil
	}

	// Getting expected volume for PVC
	desired, err := getExpectedVolume(pvc, pv, sc)
	if err != nil {
		return false, fmt.Errorf("error getting expected volume: %w", err)
	}

	diskName, subscriptionID, resourceGroupName, err := getDiskInfoFromVolumeID(desired.VolumeID)
	if err != nil {
		return false, fmt.Errorf("failed to get Azure disk info from PV: %w", err)
	}

	if err = m.setDiskClientForPV(subscriptionID); err != nil {
		return false, fmt.Errorf("failed to create disk client: %w", err)
	}

	// Getting current volume status for PVC
	actual, err := m.getCurrentVolumeStatus(ctx, desired.VolumeID)
	if err != nil {
		return false, fmt.Errorf("getting current volume status: %w", err)
	}

	if actual != nil {
		if !diffVolume(actual, desired) {
			// if actual volume is equal to desired volume, return completed
			logger.Info("Volume modification is already completed for PVC")
			return false, nil
		}
	}

	logger.Info("Begin disk update for PVC")
	if _, err = m.DiskClient.BeginUpdate(ctx, resourceGroupName, diskName, armcompute.DiskUpdate{
		Properties: &armcompute.DiskUpdateProperties{
			DiskIOPSReadWrite: desired.IOPS,
			DiskSizeGB:        desired.Size,
			DiskMBpsReadWrite: desired.Throughput,
		},
	}, nil); err != nil {
		return false, fmt.Errorf("failed to update disk: %w", err)
	}

	logger.Info("Successfully modified volume for PVC")
	return true, nil
}

// diffVolume checks if the actual volume is equal to the desired volume.
// If actual volume is equal to desired volume, return false.
func diffVolume(actual, desired *Volume) bool {
	return utils.ValuesDiffer(actual.IOPS, desired.IOPS) ||
		utils.ValuesDiffer(actual.Throughput, desired.Throughput) ||
		utils.ValuesDiffer(actual.Size, desired.Size) ||
		(actual.Type != "" && desired.Type != "" && actual.Type != desired.Type)
}

func (m *DiskModifier) getCurrentVolumeStatus(ctx context.Context, volumeID string) (*Volume, error) {
	diskName, _, resourceGroupName, err := getDiskInfoFromVolumeID(volumeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure disk info from PV: %w", err)
	}

	diskResponse, err := m.DiskClient.Get(ctx, resourceGroupName, diskName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk: %w", err)
	}

	disk := diskResponse.Disk
	return &Volume{
		VolumeID:   *disk.ID,
		Size:       disk.Properties.DiskSizeGB,
		IOPS:       disk.Properties.DiskIOPSReadWrite,
		Throughput: disk.Properties.DiskMBpsReadWrite,
		Type:       string(*disk.SKU.Name),
	}, nil
}

func getExpectedVolume(
	pvc *corev1.PersistentVolumeClaim,
	pv *corev1.PersistentVolume,
	sc *storagev1.StorageClass,
) (*Volume, error) {
	v := Volume{
		VolumeID: pv.Spec.CSI.VolumeHandle,
	}
	if err := utilerrors.NewAggregate([]error{
		setArgsFromPVC(&v, pvc),
		setArgsFromStorageClass(&v, sc),
	}); err != nil {
		return nil, err
	}
	return &v, nil
}

func (*DiskModifier) MinWaitDuration() time.Duration {
	return defaultWaitDuration
}

func setArgsFromPVC(v *Volume, pvc *corev1.PersistentVolumeClaim) error {
	size, err := getSizeFromPVC(pvc)
	if err != nil {
		return err
	}
	v.Size = ptr.To(int32(size)) //nolint: gosec // by design
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

func setArgsFromStorageClass(v *Volume, sc *storagev1.StorageClass) error {
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
		return nil, fmt.Errorf("can't parse %v param in storage class: %w", key, err)
	}

	return ptr.To(param), nil
}

func getDiskInfoFromVolumeID(volumeID string) (diskName, subscriptionID, resourceGroupName string, err error) {
	// get diskName, subscriptionID, resourceGroupName from volumeHandle
	// example: /subscriptions/xxxx/resourceGroups/xxxx/providers/Microsoft.Compute/disks/xxxx
	parts := strings.Split(volumeID, "/")
	if len(parts) != volumeIDPartsLength {
		return "", "", "", fmt.Errorf("invalid volumeHandle format")
	}
	subscriptionID = parts[2]
	resourceGroupName = parts[4]
	diskName = parts[len(parts)-1]

	return diskName, subscriptionID, resourceGroupName, nil
}

func (m *DiskModifier) setDiskClientForPV(subscriptionID string) error {
	if m.DiskClient != nil {
		return nil
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return fmt.Errorf("failed to obtain a credential: %w", err)
	}

	diskClient, err := armcompute.NewDisksClient(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create disk client: %w", err)
	}

	m.DiskClient = diskClient

	return nil
}
