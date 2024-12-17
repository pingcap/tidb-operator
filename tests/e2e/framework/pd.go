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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

func (f *Framework) WaitForPDGroupReady(ctx context.Context, pdg *v1alpha1.PDGroup) {
	// TODO: maybe wait for cluster ready
	f.Must(waiter.WaitForPDsHealthy(ctx, f.Client, pdg, waiter.LongTaskTimeout))
	f.Must(waiter.WaitForPodsReady(ctx, f.Client, runtime.FromPDGroup(pdg), waiter.LongTaskTimeout))
}

func (f *Framework) WaitForPDGroupSuspended(ctx context.Context, pdg *v1alpha1.PDGroup) {
	f.Must(waiter.WaitForListDeleted(ctx, f.Client, &corev1.PodList{}, waiter.LongTaskTimeout, client.InNamespace(f.Cluster.Namespace)))
	f.Must(waiter.WaitForObjectCondition(
		ctx,
		f.Client,
		runtime.FromPDGroup(pdg),
		v1alpha1.PDGroupCondSuspended,
		metav1.ConditionTrue,
		waiter.ShortTaskTimeout,
	))
}

func (f *Framework) WaitForPDGroupReadyAndNotSuspended(ctx context.Context, pdg *v1alpha1.PDGroup) {
	f.Must(waiter.WaitForObjectCondition(
		ctx,
		f.Client,
		runtime.FromPDGroup(pdg),
		v1alpha1.PDGroupCondSuspended,
		metav1.ConditionFalse,
		waiter.ShortTaskTimeout,
	))
	f.WaitForPDGroupReady(ctx, pdg)
}
