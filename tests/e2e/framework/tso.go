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
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

func (f *Framework) WaitForTSOGroupReady(ctx context.Context, tg *v1alpha1.TSOGroup) {
	// TODO: maybe wait for cluster ready
	ginkgo.By("wait for tso group ready")
	f.Must(waiter.WaitForObjectCondition[runtime.TSOGroupTuple](
		ctx,
		f.Client,
		tg,
		v1alpha1.CondReady,
		metav1.ConditionTrue,
		waiter.LongTaskTimeout,
	))
	f.Must(waiter.WaitForTSOsHealthy(ctx, f.Client, tg, waiter.LongTaskTimeout))
	f.Must(waiter.WaitForPodsReady(ctx, f.Client, runtime.FromTSOGroup(tg), waiter.LongTaskTimeout))
}

func (f *Framework) TestTSOAvailability(ctx context.Context, tg *v1alpha1.TSOGroup, w *Workload) {
	f.Must(waiter.WaitForClusterPDRegistered(ctx, f.Client, f.Cluster, waiter.LongTaskTimeout))
	// Prepare PD endpoints for region API access
	pdEndpoints := f.Cluster.Status.PD

	patch := client.MergeFrom(tg.DeepCopy())
	tg.Spec.Template.Annotations = map[string]string{
		"test": "test",
	}

	nctx, cancel := context.WithCancel(ctx)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer ginkgo.GinkgoRecover()
		f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTSOGroup(tg), int(*tg.Spec.Replicas), 0, waiter.LongTaskTimeout))
	}()

	done := w.MustRunPDRegionAccess(nctx, pdEndpoints)

	changeTime := time.Now()
	ginkgo.By("Rolling udpate the TSOGroup")
	f.Must(f.Client.Patch(ctx, tg, patch))
	f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTSOGroup(tg), changeTime, waiter.LongTaskTimeout))
	f.WaitForTSOGroupReady(ctx, tg)
	cancel()
	wg.Wait()
	<-done
}
