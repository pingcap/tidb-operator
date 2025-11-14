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
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"

	"github.com/pingcap/tidb-operator/v2/pkg/volumes/cloud"
)

func NewFakeEBSModifier(f GetVolumeStateFunc) cloud.VolumeModifier {
	return &EBSModifier{
		cli: NewFakeEC2VolumeAPI(f),
	}
}

type GetVolumeStateFunc func(id string) types.VolumeModificationState

type FakeEC2VolumeAPI struct {
	vs []Volume
	f  GetVolumeStateFunc
}

func NewFakeEC2VolumeAPI(f GetVolumeStateFunc) *FakeEC2VolumeAPI {
	m := &FakeEC2VolumeAPI{
		f: f,
	}

	return m
}

func (m *FakeEC2VolumeAPI) ModifyVolume(
	_ context.Context, param *ec2.ModifyVolumeInput,
	_ ...func(*ec2.Options)) (*ec2.ModifyVolumeOutput, error) {
	for i := range m.vs {
		v := &m.vs[i]
		if v.VolumeID == *param.VolumeId {
			state := m.f(v.VolumeID)
			switch state {
			// NOTE(liubo02): I'm not sure the behavior to recall the aws api when the last modification
			// is in some states
			case types.VolumeModificationStateCompleted, types.VolumeModificationStateFailed:
				m.vs[i] = Volume{
					VolumeID:   *param.VolumeId,
					Size:       param.Size,
					IOPS:       param.Iops,
					Throughput: param.Throughput,
					Type:       param.VolumeType,
				}

				return &ec2.ModifyVolumeOutput{}, nil
			}

			return nil, fmt.Errorf("volume %s has been modified or modification is not finished", v.VolumeID)
		}
	}

	v := Volume{
		VolumeID:   *param.VolumeId,
		Size:       param.Size,
		IOPS:       param.Iops,
		Throughput: param.Throughput,
		Type:       param.VolumeType,
	}

	m.vs = append(m.vs, v)

	return &ec2.ModifyVolumeOutput{}, nil
}

func (m *FakeEC2VolumeAPI) DescribeVolumesModifications(
	_ context.Context, param *ec2.DescribeVolumesModificationsInput,
	_ ...func(*ec2.Options)) (*ec2.DescribeVolumesModificationsOutput, error) {
	var mods []types.VolumeModification
	for _, id := range param.VolumeIds {
		for i := range m.vs {
			v := m.vs[i]
			if v.VolumeID != id {
				continue
			}

			mods = append(mods, types.VolumeModification{
				VolumeId:          &v.VolumeID,
				TargetIops:        v.IOPS,
				TargetSize:        v.Size,
				TargetThroughput:  v.Throughput,
				TargetVolumeType:  v.Type,
				ModificationState: m.f(id),
			})
		}
	}

	if len(mods) == 0 {
		return nil, &smithy.GenericAPIError{
			Code: errCodeNotFound,
		}
	}

	return &ec2.DescribeVolumesModificationsOutput{
		VolumesModifications: mods,
	}, nil
}
