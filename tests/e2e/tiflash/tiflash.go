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

package tiflash

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apicall"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("TiFlash", label.TiFlash, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("when scaling in TiFlash with two-step deletion", func() {
		workload := f.SetupWorkload()

		ginkgo.PIt("should complete the full scale-in flow", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx, data.WithReplicas[*runtime.TiKVGroup](3))
			fg := f.MustCreateTiFlash(ctx, data.WithReplicas[*runtime.TiFlashGroup](2))
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiFlashGroupReady(ctx, fg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			workload.MustImportData(ctx, data.DefaultTiDBServiceName)

			ginkgo.By("Scaling in TiFlash from 2 to 1 replica")
			patch := client.MergeFrom(fg.DeepCopy())
			fg.Spec.Replicas = ptr.To[int32](1)
			f.Must(f.Client.Patch(ctx, fg, patch))

			ginkgo.By("Verifying 1 TiFlash instance is marked for offline")
			offliningFlashes := findOffliningTiFlashes(ctx, f, fg, 1)
			gomega.Expect(offliningFlashes).To(gomega.HaveLen(1), "Expected 1 TiFlash instance to be marked for offline")
			offlineTiFlash := offliningFlashes[0]
			gomega.Expect(offlineTiFlash.GetDeletionTimestamp().IsZero()).To(gomega.BeTrue(), "should not delete the tiflash instance when it's going offline")

			ginkgo.By("Waiting for offline operations to complete")
			nctx, cancel := context.WithCancel(ctx)
			ch := make(chan struct{})
			synced := make(chan struct{})
			go func() {
				defer close(ch)
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WatchUntilInstanceList[scope.TiFlashGroup](
					nctx,
					f.Client,
					fg.DeepCopy(),
					waiter.WaitForTiFlashOfflineCompleted(offlineTiFlash),
					waiter.LongTaskTimeout,
					synced,
				))
			}()
			// wait until cache is synced
			<-synced

			ginkgo.By("Verifying offlined TiFlash instance is deleted")
			waitForTiFlashInstancesDeleted(ctx, f, offliningFlashes, 5*time.Minute)
			cancel()
			<-ch

			ginkgo.By("Verifying TiFlashGroup reaches desired state of 1 replica")
			f.WaitForTiFlashGroupReady(ctx, fg)
			finalFlashes, err := apicall.ListInstances[scope.TiFlashGroup](ctx, f.Client, fg)
			f.Must(err)
			gomega.Expect(finalFlashes).To(gomega.HaveLen(1), "Should have 1 TiFlash instance after scale in")
		})
	})
})

// findOffliningTiFlashes waits and finds a specific number of TiFlash instances that are being offlined.
func findOffliningTiFlashes(ctx context.Context, f *framework.Framework, fg *v1alpha1.TiFlashGroup, expectedCount int) []*v1alpha1.TiFlash {
	var offliningFlashes []*v1alpha1.TiFlash
	gomega.Eventually(func() bool {
		allFlashes, err := apicall.ListInstances[scope.TiFlashGroup](ctx, f.Client, fg)
		if err != nil {
			return false
		}
		offliningFlashes = nil
		for _, flash := range allFlashes {
			if flash.Spec.Offline {
				offliningFlashes = append(offliningFlashes, flash)
			}
		}
		return len(offliningFlashes) == expectedCount
	}, 5*time.Minute, 5*time.Second).Should(gomega.BeTrue(), fmt.Sprintf("timed out waiting for %d offlining TiFlash instances", expectedCount))
	return offliningFlashes
}

// waitForTiFlashInstancesDeleted waits for a list of TiFlash instances to be completely removed.
func waitForTiFlashInstancesDeleted(ctx context.Context, f *framework.Framework, flashes []*v1alpha1.TiFlash, timeout time.Duration) {
	for _, flash := range flashes {
		f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, flash, timeout))
	}
}
