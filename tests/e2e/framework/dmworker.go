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
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

func (f *Framework) MustCreateDMWorker(ctx context.Context, ps ...data.GroupPatch[*v1alpha1.DMWorkerGroup]) *v1alpha1.DMWorkerGroup {
	dwg := data.NewDMWorkerGroup(f.Namespace.Name, ps...)
	ginkgo.By("Creating a dm worker group")
	f.Must(f.Client.Create(ctx, dwg))

	return dwg
}

func (f *Framework) WaitForDMWorkerGroupReady(ctx context.Context, dwg *v1alpha1.DMWorkerGroup) {
	ginkgo.By("wait for dm worker group ready")
	f.Must(waiter.WaitForObjectCondition[scope.DMWorkerGroup](
		ctx,
		f.Client,
		dwg,
		v1alpha1.CondReady,
		metav1.ConditionTrue,
		waiter.LongTaskTimeout,
	))
	f.Must(waiter.WaitForDMWorkersHealthy(ctx, f.Client, dwg, waiter.LongTaskTimeout))
	f.Must(waiter.WaitForPodsReady(ctx, f.Client, runtime.FromDMWorkerGroup(dwg), waiter.LongTaskTimeout))
}
