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

package framework

import (
	"context"

	"github.com/pingcap/tidb-operator/pkg/apicall"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

func (f *Framework) WaitForAllReady(ctx context.Context, cn string) {
	f.Must(waiter.WaitForClusterReady(ctx, f.Client, f.Namespace.Name, cn, waiter.LongTaskTimeout))

	pdgs, err := apicall.ListGroups[scope.PDGroup](ctx, f.Client, f.Namespace.Name, cn)
	f.Must(err)
	for _, pdg := range pdgs {
		f.WaitForPDGroupReady(ctx, pdg)
	}

	kvgs, err := apicall.ListGroups[scope.TiKVGroup](ctx, f.Client, f.Namespace.Name, cn)
	f.Must(err)
	for _, kvg := range kvgs {
		f.WaitForTiKVGroupReady(ctx, kvg)
	}

	dbgs, err := apicall.ListGroups[scope.TiDBGroup](ctx, f.Client, f.Namespace.Name, cn)
	f.Must(err)
	for _, dbg := range dbgs {
		f.WaitForTiDBGroupReady(ctx, dbg)
	}

	fgs, err := apicall.ListGroups[scope.TiFlashGroup](ctx, f.Client, f.Namespace.Name, cn)
	f.Must(err)
	for _, fg := range fgs {
		f.WaitForTiFlashGroupReady(ctx, fg)
	}

	cgs, err := apicall.ListGroups[scope.TiCDCGroup](ctx, f.Client, f.Namespace.Name, cn)
	f.Must(err)
	for _, cg := range cgs {
		f.WaitForTiCDCGroupReady(ctx, cg)
	}
}
