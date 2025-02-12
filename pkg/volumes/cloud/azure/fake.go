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

	azruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
)

type FakeDiskClient struct {
	GetFunc func(
		ctx context.Context,
		resourceGroupName, diskName string,
		options *armcompute.DisksClientGetOptions,
	) (armcompute.DisksClientGetResponse, error)

	BeginUpdateFunc func(
		ctx context.Context,
		resourceGroupName, diskName string,
		parameters armcompute.DiskUpdate,
		options *armcompute.DisksClientBeginUpdateOptions,
	) (*azruntime.Poller[armcompute.DisksClientUpdateResponse], error)
}

func (m *FakeDiskClient) Get(
	ctx context.Context,
	resourceGroupName, diskName string,
	options *armcompute.DisksClientGetOptions,
) (armcompute.DisksClientGetResponse, error) {
	return m.GetFunc(ctx, resourceGroupName, diskName, options)
}

func (m *FakeDiskClient) BeginUpdate(
	ctx context.Context,
	resourceGroupName, diskName string,
	parameters armcompute.DiskUpdate,
	options *armcompute.DisksClientBeginUpdateOptions,
) (*azruntime.Poller[armcompute.DisksClientUpdateResponse], error) {
	return m.BeginUpdateFunc(ctx, resourceGroupName, diskName, parameters, options)
}
