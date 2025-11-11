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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

func (f *Framework) MustCreateTiKV(ctx context.Context, ps ...data.GroupPatch[*v1alpha1.TiKVGroup]) *v1alpha1.TiKVGroup {
	kvg := data.NewTiKVGroup(f.Namespace.Name, ps...)
	ginkgo.By("Creating a tikv group")
	f.Must(f.Client.Create(ctx, kvg))

	return kvg
}

func (f *Framework) WaitForTiKVGroupReady(ctx context.Context, kvg *v1alpha1.TiKVGroup) {
	// TODO: maybe wait for cluster ready
	ginkgo.By("wait for tikv group ready")
	f.Must(waiter.WaitForObjectCondition[scope.TiKVGroup](
		ctx,
		f.Client,
		kvg,
		v1alpha1.CondReady,
		metav1.ConditionTrue,
		waiter.LongTaskTimeout,
	))
	f.Must(waiter.WaitForTiKVsHealthy(ctx, f.Client, kvg, waiter.LongTaskTimeout))
	f.Must(waiter.WaitForPodsReady(ctx, f.Client, runtime.FromTiKVGroup(kvg), waiter.LongTaskTimeout))
}

func (f *Framework) WaitTiKVPreStopHookSuccess(ctx context.Context, kv *v1alpha1.TiKV) {
	waitInstanceLogContains[scope.TiKV](ctx, f, kv, "all leaders have been evicted")
}

func (f *Framework) RestartTiKVPod(ctx context.Context, kv *v1alpha1.TiKV) {
	restartInstancePod[scope.TiKV](ctx, f, kv, client.GracePeriodSeconds(30))
}
