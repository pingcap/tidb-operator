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

package aws

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	klog "k8s.io/klog/v2"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tidb-operator/pkg/manager/volumes/delegation"
)

var defaultWaitDuration = time.Hour * 6

const (
	paramKeyThroughput = "throughput"
	paramKeyIOPS       = "iops"
	paramKeyType       = "type"

	// See https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_ModifyVolume.html
	// TODO: dynamically depend on type
	maxSize = 16384
	minSize = 1

	errCodeNotFound = "InvalidVolumeModification.NotFound"
)

type EC2VolumeAPI interface {
	ModifyVolume(ctx context.Context, param *ec2.ModifyVolumeInput, optFns ...func(*ec2.Options)) (*ec2.ModifyVolumeOutput, error)
	DescribeVolumesModifications(ctx context.Context, param *ec2.DescribeVolumesModificationsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesModificationsOutput, error)
}

type EBSModifier struct {
	c EC2VolumeAPI
}

type Volume struct {
	VolumeId   string
	Size       *int32
	IOPS       *int32
	Throughput *int32
	Type       types.VolumeType

	IsCompleted bool
	IsFaild     bool
}

func NewEBSModifier(cfg aws.Config) delegation.VolumeModifier {
	return &EBSModifier{
		c: ec2.NewFromConfig(cfg),
	}
}

func (m *EBSModifier) Name() string {
	return "ebs.csi.aws.com"
}

// TODO: add more validation to avoid call aws api too frequent
func (m *EBSModifier) Validate(spvc, dpvc *corev1.PersistentVolumeClaim, ssc, dsc *storagev1.StorageClass) error {
	if ssc.Provisioner != dsc.Provisioner {
		return fmt.Errorf("provisioner should not be changed, now from %s to %s", ssc.Provisioner, dsc.Provisioner)
	}

	return nil
}

func (m *EBSModifier) ModifyVolume(ctx context.Context, pvc *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume, sc *storagev1.StorageClass) ( /*wait*/ bool, error) {
	desired, err := m.getExpectedVolume(pvc, pv, sc)
	if err != nil {
		return false, err
	}

	actual, err := m.getCurrentVolumeStatus(ctx, desired.VolumeId)
	if err != nil {
		return false, err
	}

	if actual != nil {
		// current one is matched with the desired
		if !m.diffVolume(actual, desired) {
			if actual.IsCompleted {
				return false, nil
			}
			if !actual.IsFaild {
				return true, nil
			}
		}
	}

	klog.V(2).Infof("call aws api to modify volume for pvc %s/%s", pvc.Namespace, pvc.Name)

	// retry to modify the volume
	if _, err := m.c.ModifyVolume(ctx, &ec2.ModifyVolumeInput{
		VolumeId:   &desired.VolumeId,
		Size:       desired.Size,
		Iops:       desired.IOPS,
		Throughput: desired.Throughput,
		VolumeType: desired.Type,
	}); err != nil {
		return false, err
	}

	return true, nil
}

func (m *EBSModifier) diffVolume(actual, desired *Volume) bool {
	if diffInt32(actual.IOPS, desired.IOPS) {
		return true
	}
	if diffInt32(actual.Throughput, desired.Throughput) {
		return true
	}
	if diffInt32(actual.Size, desired.Size) {
		return true
	}
	if actual.Type != desired.Type {
		return true
	}

	return false
}

func diffInt32(a, b *int32) bool {
	if a == nil && b == nil {
		return false
	}

	if a == nil || b == nil {
		return true
	}

	if *a == *b {
		return false
	}

	return true
}

func (m *EBSModifier) getCurrentVolumeStatus(ctx context.Context, id string) (*Volume, error) {
	res, err := m.c.DescribeVolumesModifications(ctx, &ec2.DescribeVolumesModificationsInput{
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
	for _, s := range res.VolumesModifications {
		if s.VolumeId == nil || *s.VolumeId != id {
			continue
		}
		v := Volume{
			VolumeId:   *s.VolumeId,
			Size:       s.TargetSize,
			IOPS:       s.TargetIops,
			Throughput: s.TargetThroughput,
			Type:       s.TargetVolumeType,
		}
		switch s.ModificationState {
		case types.VolumeModificationStateCompleted:
			v.IsCompleted = true
		case types.VolumeModificationStateFailed:
			v.IsFaild = true
		case types.VolumeModificationStateModifying:
		case types.VolumeModificationStateOptimizing:
			v.IsCompleted = true
		}

		return &v, nil
	}

	return nil, nil
}

func (m *EBSModifier) getExpectedVolume(pvc *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume, sc *storagev1.StorageClass) (*Volume, error) {
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

func (m *EBSModifier) MinWaitDuration() time.Duration {
	return defaultWaitDuration
}

func (m *EBSModifier) setArgsFromPVC(v *Volume, pvc *corev1.PersistentVolumeClaim) error {
	size, err := getSizeFromPVC(pvc)
	if err != nil {
		return err
	}
	v.Size = pointer.Int32Ptr(int32(size))
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

func (m *EBSModifier) setArgsFromPV(v *Volume, pv *corev1.PersistentVolume) error {
	v.VolumeId = pv.Spec.CSI.VolumeHandle
	return nil
}

func (m *EBSModifier) setArgsFromStorageClass(v *Volume, sc *storagev1.StorageClass) error {
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
		return nil, fmt.Errorf("can't parse %v param in storage class: %v", key, err)
	}

	return pointer.Int32Ptr(int32(param)), nil
}
