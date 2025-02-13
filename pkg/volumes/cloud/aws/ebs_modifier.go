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

package aws

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/pkg/utils"
	"github.com/pingcap/tidb-operator/pkg/volumes/cloud"
)

var (
	// defaultWaitDuration is the cooldown period for EBS ModifyVolume.
	// Multiple ModifyVolume calls for the same volume within a 6-hour period will fail.
	defaultWaitDuration = time.Hour * 6

	// maxStorageSizeInGiB is the maximum size of EBS volume in GiB.
	// Ref: https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_ModifyVolume.html#API_ModifyVolume_RequestParameters
	maxStorageSizeInGiB = map[types.VolumeType]int{
		types.VolumeTypeGp2:      16384,
		types.VolumeTypeGp3:      16384,
		types.VolumeTypeIo1:      16384,
		types.VolumeTypeIo2:      65536,
		types.VolumeTypeSc1:      16384,
		types.VolumeTypeSt1:      16384,
		types.VolumeTypeStandard: 1024,
	}
	minStorageSizeInGiB = map[types.VolumeType]int{
		types.VolumeTypeGp2:      1,
		types.VolumeTypeGp3:      1,
		types.VolumeTypeIo1:      4,
		types.VolumeTypeIo2:      4,
		types.VolumeTypeSc1:      125,
		types.VolumeTypeSt1:      125,
		types.VolumeTypeStandard: 1,
	}

	maxIOPS = map[types.VolumeType]int{
		types.VolumeTypeGp3: 16000,
		types.VolumeTypeIo1: 64000,
		types.VolumeTypeIo2: 256000,
	}
	minIOPS = map[types.VolumeType]int{
		types.VolumeTypeGp3: 3000,
		types.VolumeTypeIo1: 100,
		types.VolumeTypeIo2: 100,
	}

	maxThroughput = 1000
	minThroughput = 125
)

const (
	paramKeyThroughput = "throughput"
	paramKeyIOPS       = "iops"
	paramKeyType       = "type"

	errCodeNotFound = "InvalidVolumeModification.NotFound"
)

type EC2VolumeAPI interface {
	ModifyVolume(ctx context.Context, param *ec2.ModifyVolumeInput, optFns ...func(*ec2.Options)) (*ec2.ModifyVolumeOutput, error)
	DescribeVolumesModifications(ctx context.Context, param *ec2.DescribeVolumesModificationsInput,
		optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesModificationsOutput, error)
}

type EBSModifier struct {
	cli    EC2VolumeAPI
	logger logr.Logger
}

type Volume struct {
	VolumeID   string
	Size       *int32
	IOPS       *int32
	Throughput *int32
	Type       types.VolumeType

	IsCompleted bool
	IsFailed    bool
}

// NewEBSModifier creates a new EBS volume modifier.
func NewEBSModifier(logger logr.Logger) cloud.VolumeModifier {
	return &EBSModifier{
		logger: logger,
	}
}

func (*EBSModifier) Name() string {
	return "ebs.csi.aws.com"
}

func (m *EBSModifier) Validate(_, _ *corev1.PersistentVolumeClaim, ssc, dsc *storagev1.StorageClass) error {
	if ssc != nil && dsc != nil {
		if ssc.Provisioner != dsc.Provisioner {
			return fmt.Errorf("provisioner should not be changed, now from %s to %s", ssc.Provisioner, dsc.Provisioner)
		}
		if ssc.Provisioner != m.Name() {
			return fmt.Errorf("provisioner should be %s, now is %s", m.Name(), ssc.Provisioner)
		}
	} else {
		m.logger.Info("storage class is nil, skip validation")
	}

	return nil
}

func NewClient(ctx context.Context) (EC2VolumeAPI, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return ec2.NewFromConfig(cfg), nil
}

func (m *EBSModifier) Modify(ctx context.Context, pvc *corev1.PersistentVolumeClaim,
	pv *corev1.PersistentVolume, sc *storagev1.StorageClass,
) ( /*wait*/ bool, error) {
	if pv == nil {
		m.logger.Info("Persistent volume is nil, skip modifying PV. This may be caused by no relevant permissions", "pv", pvc.Spec.VolumeName)
		return false, nil
	}

	desired, err := m.getExpectedVolume(pvc, pv, sc)
	if err != nil {
		return false, err
	}

	if m.cli == nil {
		cli, err := NewClient(ctx)
		if err != nil {
			return false, fmt.Errorf("cannot new aws client: %w", err)
		}
		m.cli = cli
	}

	actual, err := m.getCurrentVolumeStatus(ctx, desired.VolumeID)
	if err != nil {
		return false, err
	}

	if actual != nil {
		// current one is matched with the desired
		if !m.diffVolume(actual, desired) {
			if actual.IsCompleted {
				return false, nil
			}
			if !actual.IsFailed {
				return true, nil
			}
		}
	}

	m.logger.Info("call aws api to modify volume for pvc", "pvc_namespace", pvc.Namespace, "pvc_name", pvc.Name)

	// retry to modify the volume
	if _, err := m.cli.ModifyVolume(ctx, &ec2.ModifyVolumeInput{
		VolumeId:   &desired.VolumeID,
		Size:       desired.Size,
		Iops:       desired.IOPS,
		Throughput: desired.Throughput,
		VolumeType: desired.Type,
	}); err != nil {
		return false, err
	}

	return true, nil
}

// If some params are not set, assume they are equal.
func (*EBSModifier) diffVolume(actual, desired *Volume) bool {
	if utils.ValuesDiffer(actual.IOPS, desired.IOPS) {
		return true
	}
	if utils.ValuesDiffer(actual.Throughput, desired.Throughput) {
		return true
	}
	if utils.ValuesDiffer(actual.Size, desired.Size) {
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

func (m *EBSModifier) getCurrentVolumeStatus(ctx context.Context, id string) (*Volume, error) {
	res, err := m.cli.DescribeVolumesModifications(ctx, &ec2.DescribeVolumesModificationsInput{
		VolumeIds: []string{id},
	})
	if err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) {
			if ae.ErrorCode() == errCodeNotFound {
				return nil, nil
			}
		}
		return nil, err
	}

	// TODO: maybe cool down time should also be returned to avoid
	// recalling ModifyVolume too many times
	for i := range res.VolumesModifications {
		vm := &res.VolumesModifications[i]
		if vm.VolumeId == nil || *vm.VolumeId != id {
			continue
		}
		v := Volume{
			VolumeID:   *vm.VolumeId,
			Size:       vm.TargetSize,
			IOPS:       vm.TargetIops,
			Throughput: vm.TargetThroughput,
			Type:       vm.TargetVolumeType,
		}
		switch vm.ModificationState {
		case types.VolumeModificationStateCompleted:
			v.IsCompleted = true
		case types.VolumeModificationStateFailed:
			v.IsFailed = true
		case types.VolumeModificationStateModifying:
		case types.VolumeModificationStateOptimizing:
			v.IsCompleted = true
		}

		return &v, nil
	}

	return nil, nil
}

func (m *EBSModifier) getExpectedVolume(pvc *corev1.PersistentVolumeClaim,
	pv *corev1.PersistentVolume, sc *storagev1.StorageClass,
) (*Volume, error) {
	// get storage size in GiB from PVC
	quantity := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	sizeBytes := quantity.ScaledValue(0)
	size := sizeBytes / 1024 / 1024 / 1024

	v := &Volume{}
	//nolint:gosec // expected type conversion
	v.Size = ptr.To(int32(size))
	v.VolumeID = pv.Spec.CSI.VolumeHandle
	if err := m.setArgsFromStorageClass(v, sc); err != nil {
		return nil, err
	}

	return v, validateVolume(v)
}

func (*EBSModifier) MinWaitDuration() time.Duration {
	return defaultWaitDuration
}

func (*EBSModifier) setArgsFromStorageClass(v *Volume, sc *storagev1.StorageClass) error {
	if sc == nil {
		return nil
	}
	throughput, err := getParamInt32(sc.Parameters, paramKeyThroughput)
	if err != nil {
		return err
	}
	v.Throughput = throughput

	iops, err := getParamInt32(sc.Parameters, paramKeyIOPS)
	if err != nil {
		return err
	}
	v.IOPS = iops

	typ := sc.Parameters[paramKeyType]
	v.Type = types.VolumeType(typ)

	return nil
}

func getParamInt32(params map[string]string, key string) (*int32, error) {
	str, ok := params[key]
	if !ok {
		return nil, nil
	}
	if str == "" {
		return nil, nil
	}
	param, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("can't parse %v param in storage class: %w", key, err)
	}

	return ptr.To(int32(param)), nil
}

//nolint:gocyclo // refactor if possible
func validateVolume(desired *Volume) error {
	if desired == nil || string(desired.Type) == "" {
		return nil
	}

	if desired.Size != nil {
		size := int(*desired.Size)
		minSize, ok1 := minStorageSizeInGiB[desired.Type]
		maxSize, ok2 := maxStorageSizeInGiB[desired.Type]
		if ok1 && ok2 && (size < minSize || size > maxSize) {
			return fmt.Errorf("size %d is out of range [%d, %d] for volume type %s", size, minSize, maxSize, desired.Type)
		}
	}

	if desired.IOPS != nil {
		iops := int(*desired.IOPS)
		minIops, ok1 := minIOPS[desired.Type]
		maxIops, ok2 := maxIOPS[desired.Type]
		if !ok1 || !ok2 {
			return fmt.Errorf("modifying IOPS for volume type %s is not supported", desired.Type)
		}
		if iops < minIops || iops > maxIops {
			return fmt.Errorf("iops %d is out of range [%d, %d] for volume type %s", iops, minIops, maxIops, desired.Type)
		}
	}

	if desired.Throughput != nil {
		throughput := int(*desired.Throughput)
		if throughput < minThroughput || throughput > maxThroughput {
			return fmt.Errorf("throughput %d is out of range [%d, %d]", throughput, minThroughput, maxThroughput)
		}
	}

	return nil
}
