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
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

func (f *Framework) WaitForSchedulingGroupReady(ctx context.Context, sg *v1alpha1.SchedulingGroup) {
	// TODO: maybe wait for cluster ready
	ginkgo.By("wait for scheduling group ready")
	f.Must(waiter.WaitForObjectCondition[runtime.SchedulingGroupTuple](
		ctx,
		f.Client,
		sg,
		v1alpha1.CondReady,
		metav1.ConditionTrue,
		waiter.LongTaskTimeout,
	))
	f.Must(waiter.WaitForSchedulingsHealthy(ctx, f.Client, sg, waiter.LongTaskTimeout))
	f.Must(waiter.WaitForPodsReady(ctx, f.Client, runtime.FromSchedulingGroup(sg), waiter.LongTaskTimeout))
}
