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

package tikv

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apicall"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
)

var _ = ginkgo.Describe("Two-Step Store Deletion", label.TiKV, label.P0, func() {
	f := framework.New()
	f.Setup(framework.WithSkipClusterDeletionWhenFailed())

	ginkgo.Context("when scaling in TiKV with two-step deletion", func() {
		workload := f.SetupWorkload()

		ginkgo.It("should complete the full scale-in flow", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx, data.WithReplicas[*runtime.TiKVGroup](4))
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			workload.MustImportData(ctx, data.DefaultTiDBServiceName, "root", "", "", 500)
			//ginkgo.Fail("debug")

			ginkgo.By("Scaling in TiKV from 4 to 3 replica")
			patch := client.MergeFrom(kvg.DeepCopy())
			kvg.Spec.Replicas = ptr.To[int32](3)
			f.Must(f.Client.Patch(ctx, kvg, patch))

			ginkgo.By("Verifying 1 TiKV instances are marked for offline")
			offliningKVs := findOffliningTiKVs(ctx, f, kvg, 1)
			gomega.Expect(offliningKVs).To(gomega.HaveLen(1), "Expected 1 TiKV instances to be marked for offline")
			offlineTiKV := offliningKVs[0]
			gomega.Expect(offlineTiKV.GetDeletionTimestamp().IsZero()).To(gomega.BeTrue(), "should not delete the tikv instance when it's going offline")

			ginkgo.By("Waiting for offline operations to complete")
			waitForTiKVOfflineCondition(ctx, f, offlineTiKV, v1alpha1.OfflineReasonCompleted, 5*time.Minute, true)

			ginkgo.By("Verifying offlined TiKV instances are deleted")
			waitForTiKVInstancesDeleted(ctx, f, offliningKVs, 5*time.Minute)

			ginkgo.By("Verifying TiKVGroup reaches desired state of 3 replica")
			f.WaitForTiKVGroupReady(ctx, kvg)
			finalTiKVs, err := apicall.ListInstances[scope.TiKVGroup](ctx, f.Client, kvg)
			f.Must(err)
			gomega.Expect(finalTiKVs).To(gomega.HaveLen(3), "Should have 3 TiKV instance after scale in")
		})

		ginkgo.It("should handle full cancellation of scale-in", func(ctx context.Context) {
			// Slow down the speed of data migration for testing
			pdg := f.MustCreatePD(ctx, func(pdg *runtime.PDGroup) {
				pdg.Spec.Template.Spec.Config = `[schedule]
region-schedule-limit = 32
replica-schedule-limit = 16
`
			})
			kvg := f.MustCreateTiKV(ctx, data.WithReplicas[*runtime.TiKVGroup](4))
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			workload.MustImportData(ctx, data.DefaultTiDBServiceName, "root", "", "", 500)

			ginkgo.By("Annotating one TiKV instance for deletion")
			allKVs, err := apicall.ListInstances[scope.TiKVGroup](ctx, f.Client, kvg)
			f.Must(err)
			sort.Slice(allKVs, func(i, j int) bool { return allKVs[i].Name < allKVs[j].Name })
			annotatedKV := allKVs[rand.IntN(len(allKVs))]
			patch := client.MergeFrom(annotatedKV.DeepCopy())
			metav1.SetMetaDataAnnotation(&annotatedKV.ObjectMeta, v1alpha1.AnnoKeyOfflineStore, "true")
			f.Must(f.Client.Patch(ctx, annotatedKV, patch))

			ginkgo.By("Scaling in TiKV from 4 to 3 replica")
			patch = client.MergeFrom(kvg.DeepCopy())
			kvg.Spec.Replicas = ptr.To[int32](3)
			f.Must(f.Client.Patch(ctx, kvg, patch))

			ginkgo.By("Verifying the annotated TiKV instances are marked for offline")
			offliningKVs := findOffliningTiKVs(ctx, f, kvg, 1)
			gomega.Expect(offliningKVs).To(gomega.HaveLen(1))
			gomega.Expect(offliningKVs[0].Name).To(gomega.Equal(annotatedKV.Name))

			ginkgo.By("Cancelling scale-in by scaling back to 4 replicas")
			patch = client.MergeFrom(kvg.DeepCopy())
			kvg.Spec.Replicas = ptr.To[int32](4)
			f.Must(f.Client.Patch(ctx, kvg, patch))

			ginkgo.By("Verifying offline operations are cancelled")
			waitForTiKVOfflineCondition(ctx, f, annotatedKV, v1alpha1.OfflineReasonCancelled, 5*time.Minute, false)

			ginkgo.By("Verifying TiKVGroup stabilizes at 4 replicas")
			f.WaitForTiKVGroupReady(ctx, kvg)
			finalTiKVs, err := apicall.ListInstances[scope.TiKVGroup](ctx, f.Client, kvg)
			f.Must(err)
			gomega.Expect(finalTiKVs).To(gomega.HaveLen(4))
			for _, kv := range finalTiKVs {
				gomega.Expect(kv.Spec.Offline).To(gomega.BeFalse())
			}
		})

		ginkgo.PIt("should handle partial cancellation of scale-in", func(ctx context.Context) {
			// Slow down the speed of data migration for testing
			pdg := f.MustCreatePD(ctx, func(pdg *runtime.PDGroup) {
				pdg.Spec.Template.Spec.Config = `[schedule]
region-schedule-limit = 32
replica-schedule-limit = 16
`
			})
			kvg := f.MustCreateTiKV(ctx, data.WithReplicas[*runtime.TiKVGroup](5))
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			workload.MustImportData(ctx, data.DefaultTiDBServiceName, "root", "", "", 500)

			ginkgo.By("Scaling in TiKV from 5 to 3 replicas")
			patch := client.MergeFrom(kvg.DeepCopy())
			kvg.Spec.Replicas = ptr.To[int32](3)
			f.Must(f.Client.Patch(ctx, kvg, patch))

			ginkgo.By("Verifying 2 TiKV instances are marked for offline")
			offliningKVs := findOffliningTiKVs(ctx, f, kvg, 2)
			gomega.Expect(offliningKVs).To(gomega.HaveLen(2))

			ginkgo.By("Partially cancelling by scaling up to 4 replicas")
			patch = client.MergeFrom(kvg.DeepCopy())
			kvg.Spec.Replicas = ptr.To[int32](4)
			f.Must(f.Client.Patch(ctx, kvg, patch))

			ginkgo.By("Verifying one offline operation is cancelled and the other continue")
			var cancelledKVs, remainingOffliningKVs []*v1alpha1.TiKV
			gomega.Eventually(func(g gomega.Gomega) {
				cancelledKVs = nil
				remainingOffliningKVs = nil
				for _, kv := range offliningKVs {
					instance := &v1alpha1.TiKV{}
					g.Expect(f.Client.Get(ctx, client.ObjectKeyFromObject(kv), instance)).To(gomega.Succeed())
					if !instance.Spec.Offline {
						cancelledKVs = append(cancelledKVs, instance)
					} else {
						remainingOffliningKVs = append(remainingOffliningKVs, instance)
					}
				}
				g.Expect(cancelledKVs).To(gomega.HaveLen(1), "Expected 1 offline operation to be cancelled")
				g.Expect(remainingOffliningKVs).To(gomega.HaveLen(1), "Expected 1 offline operations to continue")
			}, 5*time.Minute, 5*time.Second).Should(gomega.Succeed())

			ginkgo.By("Verifying the cancelled instance status is updated")
			waitForTiKVOfflineCondition(ctx, f, cancelledKVs[0], v1alpha1.OfflineReasonCancelled, 3*time.Minute, false)

			ginkgo.By("Waiting for the remaining offline operations to complete")
			waitForTiKVOfflineCondition(ctx, f, remainingOffliningKVs[0], v1alpha1.OfflineReasonCompleted, 5*time.Minute, true)

			ginkgo.By("Verifying offlined TiKV instances are deleted")
			waitForTiKVInstancesDeleted(ctx, f, remainingOffliningKVs, 5*time.Minute)

			ginkgo.By("Verifying TiKVGroup stabilizes at 4 replicas")
			f.WaitForTiKVGroupReady(ctx, kvg)
			finalTiKVs, err := apicall.ListInstances[scope.TiKVGroup](ctx, f.Client, kvg)
			f.Must(err)
			gomega.Expect(finalTiKVs).To(gomega.HaveLen(4))
		})
	})
})

// findOffliningTiKVs waits and finds a specific number of TiKV instances that are being offlined.
func findOffliningTiKVs(ctx context.Context, f *framework.Framework, kvg *v1alpha1.TiKVGroup, expectedCount int) []*v1alpha1.TiKV {
	var offliningKVs []*v1alpha1.TiKV
	gomega.Eventually(func() bool {
		allKVs, err := apicall.ListInstances[scope.TiKVGroup](ctx, f.Client, kvg)
		if err != nil {
			return false
		}
		offliningKVs = nil
		for _, kv := range allKVs {
			if kv.Spec.Offline {
				offliningKVs = append(offliningKVs, kv)
			}
		}
		return len(offliningKVs) == expectedCount
	}, 5*time.Minute, 5*time.Second).Should(gomega.BeTrue(), fmt.Sprintf("timed out waiting for %d offlining TiKV instances", expectedCount))
	return offliningKVs
}

// waitForTiKVOfflineCondition waits for a TiKV instance's offline condition to reach a specific reason.
func waitForTiKVOfflineCondition(ctx context.Context, f *framework.Framework, kv *v1alpha1.TiKV, reason string, timeout time.Duration, status bool) {
	gomega.Eventually(func(g gomega.Gomega) {
		instance := &v1alpha1.TiKV{}
		g.Expect(f.Client.Get(ctx, client.ObjectKeyFromObject(kv), instance)).To(gomega.Succeed())

		cond := meta.FindStatusCondition(instance.Status.Conditions, v1alpha1.StoreOfflineConditionType)
		g.Expect(cond).NotTo(gomega.BeNil(), "StoreOffline condition should exist")
		if status {
			g.Expect(cond.Status).To(gomega.Equal(metav1.ConditionTrue))
		} else {
			g.Expect(cond.Status).To(gomega.Equal(metav1.ConditionFalse))
		}
		g.Expect(cond.Reason).To(gomega.Equal(reason), fmt.Sprintf("Expected offline condition reason to be %s, but got %s", reason, cond.Reason))
	}, timeout, 10*time.Second).Should(gomega.Succeed(), fmt.Sprintf("TiKV instance %s did not reach offline condition %s in time", kv.Name, reason))
}

// waitForTiKVInstancesDeleted waits for a list of TiKV instances to be completely removed.
func waitForTiKVInstancesDeleted(ctx context.Context, f *framework.Framework, kvs []*v1alpha1.TiKV, timeout time.Duration) {
	for _, kv := range kvs {
		waitForTiKVInstanceCleanup(ctx, f, kv, timeout)
	}
}
