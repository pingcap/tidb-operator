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

package util

import (
	"context"
	"fmt"
	"sort"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

func FirstTikvGroup(ctx context.Context, cli client.Client, ns, cluster string) (*v1alpha1.TiKVGroup, error) {
	tikvGroups, err := apicall.ListGroups[scope.TiKVGroup](ctx, cli, ns, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to list tikv groups: %w", err)
	}
	if len(tikvGroups) == 0 {
		return nil, fmt.Errorf("no tikv groups found in cluster %s", cluster)
	}
	sort.Slice(tikvGroups, func(i, j int) bool {
		return tikvGroups[i].CreationTimestamp.Before(&tikvGroups[j].CreationTimestamp)
	})
	tikvGroup := tikvGroups[0]
	return tikvGroup, nil
}

func FirstPDGroup(ctx context.Context, cli client.Client, ns, cluster string) (*v1alpha1.PDGroup, error) {
	pdGroups, err := apicall.ListGroups[scope.PDGroup](ctx, cli, ns, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to list pd groups: %w", err)
	}
	if len(pdGroups) == 0 {
		return nil, fmt.Errorf("no pd groups found in cluster %s", cluster)
	}
	sort.Slice(pdGroups, func(i, j int) bool {
		return pdGroups[i].CreationTimestamp.Before(&pdGroups[j].CreationTimestamp)
	})
	pdGroup := pdGroups[0]
	return pdGroup, nil
}
