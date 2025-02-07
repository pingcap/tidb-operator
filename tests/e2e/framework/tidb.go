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
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

func (f *Framework) WaitForTiDBGroupReady(ctx context.Context, dbg *v1alpha1.TiDBGroup) {
	// TODO: maybe wait for cluster ready
	ginkgo.By("wait for tidb group ready")
	f.Must(waiter.WaitForTiDBsHealthy(ctx, f.Client, dbg, waiter.LongTaskTimeout))
	f.Must(waiter.WaitForPodsReady(ctx, f.Client, runtime.FromTiDBGroup(dbg), waiter.LongTaskTimeout))
}
