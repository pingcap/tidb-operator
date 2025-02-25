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

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

func (f *Framework) MustCreateTiCDC(ctx context.Context, ps ...data.GroupPatch[*runtime.TiCDCGroup]) *v1alpha1.TiCDCGroup {
	cg := data.NewTiCDCGroup(f.Namespace.Name, ps...)
	ginkgo.By("Creating a ticdc group")
	f.Must(f.Client.Create(ctx, cg))

	return cg
}

func (f *Framework) WaitForTiCDCGroupReady(ctx context.Context, cg *v1alpha1.TiCDCGroup) {
	// TODO: maybe wait for cluster ready
	ginkgo.By("wait for ticdc group ready")
	f.Must(waiter.WaitForTiCDCsHealthy(ctx, f.Client, cg, waiter.LongTaskTimeout))
	f.Must(waiter.WaitForPodsReady(ctx, f.Client, runtime.FromTiCDCGroup(cg), waiter.LongTaskTimeout))
}

func (f *Framework) WaitTiCDCPreStopHookSuccess(ctx context.Context, cdc *v1alpha1.TiCDC) {
	waitInstanceLogContains[scope.TiCDC](ctx, f, cdc, "prestop hook is successfully executed")
}

func (f *Framework) RestartTiCDCPod(ctx context.Context, cdc *v1alpha1.TiCDC) {
	restartInstancePod[scope.TiCDC](ctx, f, cdc)
}
