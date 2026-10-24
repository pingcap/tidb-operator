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

package scale

import (
	"context"
	"slices"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/action"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/desc"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("Scale TiDB", label.TiDB, label.P0, label.Scale, func() {
	f := framework.New()
	f.Setup()

	ginkgo.It("prefer instance priority before Pod deletion cost when scaling in", func(ctx context.Context) {
		o := desc.DefaultOptions()
		pdg := action.MustCreatePD(ctx, f, o)
		kvg := action.MustCreateTiKV(ctx, f, o)
		dbg := action.MustCreateTiDB(ctx, f, o, data.WithReplicas[scope.TiDBGroup](1))
		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.WaitForTiDBGroupReady(ctx, dbg)

		// Create the other instances after the first is ready, giving it an older timestamp.
		action.MustScale[scope.TiDBGroup](ctx, f, dbg, 3)
		f.WaitForTiDBGroupReady(ctx, dbg)

		instances, err := apicall.ListInstances[scope.TiDBGroup](ctx, f.Client, dbg)
		f.Must(err)
		gomega.Expect(instances).To(gomega.HaveLen(3))
		slices.SortFunc(instances, func(a, b *v1alpha1.TiDB) int {
			return a.CreationTimestamp.Compare(b.CreationTimestamp.Time)
		})
		// Prefer the oldest Pod by cost, so creation time alone cannot explain the choice.
		cheap, survivor, priority := instances[0], instances[1], instances[2]
		patch := client.MergeFrom(priority.DeepCopy())
		if priority.Annotations == nil {
			priority.Annotations = map[string]string{}
		}
		priority.Annotations[v1alpha1.AnnoKeyPriority] = "0"
		f.Must(f.Client.Patch(ctx, priority, patch))

		pods := make([]*corev1.Pod, 2)
		for i, in := range []*v1alpha1.TiDB{priority, cheap} {
			pods[i], err = apicall.GetPod[scope.TiDB](ctx, f.Client, in)
			f.Must(err)
			patch := client.MergeFrom(pods[i].DeepCopy())
			if pods[i].Annotations == nil {
				pods[i].Annotations = map[string]string{}
			}
			pods[i].Annotations[corev1.PodDeletionCost] = []string{"100", "-100"}[i]
			f.Must(f.Client.Patch(ctx, pods[i], patch))
		}

		ginkgo.By("scale from 3 to 2: instance priority overrides the higher Pod deletion cost")
		action.MustScale[scope.TiDBGroup](ctx, f, dbg, 2)
		f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, priority, waiter.LongTaskTimeout))
		f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, pods[0], waiter.LongTaskTimeout))
		f.WaitForTiDBGroupReady(ctx, dbg)
		remaining, err := apicall.ListInstances[scope.TiDBGroup](ctx, f.Client, dbg)
		f.Must(err)
		gomega.Expect(remaining).To(gomega.ConsistOf(
			gomega.HaveField("UID", cheap.UID), gomega.HaveField("UID", survivor.UID),
		))

		ginkgo.By("scale from 2 to 1: the negative Pod deletion cost wins over the unset cost")
		action.MustScale[scope.TiDBGroup](ctx, f, dbg, 1)
		f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, cheap, waiter.LongTaskTimeout))
		f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, pods[1], waiter.LongTaskTimeout))
		f.WaitForTiDBGroupReady(ctx, dbg)
		remaining, err = apicall.ListInstances[scope.TiDBGroup](ctx, f.Client, dbg)
		f.Must(err)
		gomega.Expect(remaining).To(gomega.ConsistOf(gomega.HaveField("UID", survivor.UID)))
	})

	ginkgo.It("support scale from 3 to 5 and rolling restart at same time with maxSurge 2", label.Update, func(ctx context.Context) {
		o := desc.DefaultOptions()

		pdg := action.MustCreatePD(ctx, f, o)
		kvg := action.MustCreateTiKV(ctx, f, o)
		dbg := action.MustCreateTiDB(ctx, f, o,
			data.WithReplicas[scope.TiDBGroup](3),
			data.WithTiDBMaxSurge(2),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.WaitForTiDBGroupReady(ctx, dbg)

		nctx, cancel := context.WithCancel(ctx)
		done := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiDBGroup](nctx, f, dbg, 5, false)
		defer func() { <-done }()
		defer cancel()

		// Record the latest old pod creation timestamp so we can verify the rolling restart
		// actually creates new pods instead of only scaling out.
		changeTime, err := waiter.MaxPodsCreateTimestamp[scope.TiDBGroup](ctx, f.Client, dbg)
		f.Must(err)

		// Patch replicas and pod template at the same time to force the updater into the
		// "scale out + rolling restart" path where spec.maxSurge should be honored.
		action.MustScaleAndRollingRestart[scope.TiDBGroup](ctx, f, dbg, 5)

		f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiDBGroup(dbg), *changeTime, waiter.LongTaskTimeout))
		f.WaitForTiDBGroupReady(ctx, dbg)
	})
})
