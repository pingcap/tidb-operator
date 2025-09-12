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
	"github.com/pingcap/tidb-operator/tests/e2e/utils/cert"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("TiKV", label.TiKV, func() {
	f := framework.New()
	f.Setup()

	ginkgo.DescribeTableSubtree("Leader Eviction", label.P1,
		func(tls bool) {
			if tls {
				f.SetupCluster(data.WithClusterTLSEnabled())
			}

			// NOTE(liubo02): this case is failed in e2e env because of the cgroup v2.
			// Enable it if env is fixed.
			ginkgo.PIt("leader evicted when delete tikv pod directly", func(ctx context.Context) {
				if tls {
					ns := f.Cluster.Namespace
					cn := f.Cluster.Name
					f.Must(cert.InstallTiDBIssuer(ctx, f.Client, ns, cn))
					f.Must(cert.InstallTiDBCertificates(ctx, f.Client, ns, cn, "dbg"))
					f.Must(cert.InstallTiDBComponentsCertificates(ctx, f.Client, ns, cn, "pdg", "kvg", "dbg", "flashg", "cdcg", "pg"))
				}
				pdg := f.MustCreatePD(ctx)
				kvg := f.MustCreateTiKV(ctx,
					data.WithReplicas[*runtime.TiKVGroup](3),
				)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTiKVGroupReady(ctx, kvg)

				kvs, err := apicall.ListInstances[scope.TiKVGroup](ctx, f.Client, kvg)
				f.Must(err)

				kv := kvs[0]

				nctx, cancel := context.WithCancel(ctx)
				ch := make(chan struct{})
				go func() {
					defer close(ch)
					defer ginkgo.GinkgoRecover()
					f.WaitTiKVPreStopHookSuccess(nctx, kv)
				}()

				f.RestartTiKVPod(ctx, kv)

				cancel()
				<-ch
			})
		},
		func(tls bool) string {
			if tls {
				return "TLS"
			}
			return "NO TLS"
		},
		ginkgo.Entry(nil, false),
		ginkgo.Entry(nil, label.FeatureTLS, true),
	)

	ginkgo.Context("Race Condition Scenarios", label.P1, func() {
		workload := f.SetupWorkload()

		ginkgo.It("should recreate pod when deleted during graceful store removal", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithSlowDataMigration())
			kvg := f.MustCreateTiKV(ctx, data.WithReplicas[*runtime.TiKVGroup](4))
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			// Make sure each TiKV store has enough leaders and regions,
			// otherwise the scale-in operation will be too fast.
			workload.MustImportData(ctx, data.DefaultTiDBServiceName)

			ginkgo.By("Initiating scale in from 4 to 3 replicas")
			patch := client.MergeFrom(kvg.DeepCopy())
			kvg.Spec.Replicas = ptr.To[int32](3)
			f.Must(f.Client.Patch(ctx, kvg, patch))

			ginkgo.By("Finding the TiKV instance that is being scaled in")
			offliningKVs := findOffliningTiKVs(ctx, f, kvg, 1)
			gomega.Expect(offliningKVs).To(gomega.HaveLen(1), "Expected 1 TiKV instances to be marked for offline")
			targetTiKV := offliningKVs[0]

			ginkgo.By("Recording original pod information")
			originalPod, err := apicall.GetPod[scope.TiKV](ctx, f.Client, targetTiKV)
			f.Must(err)
			originalPodUID := originalPod.UID

			ginkgo.By("Simulating manual pod deletion during graceful shutdown")
			// This simulates the race condition where user manually deletes the pod
			// while the operator is trying to gracefully remove the store from PD
			f.Must(f.Client.Delete(ctx, originalPod, client.GracePeriodSeconds(0)))

			ginkgo.By("Verifying operator recreates the pod during store removal")

			gomega.Eventually(func(g gomega.Gomega) {
				// The operator should recreate the pod to ensure graceful store removal can complete
				newPod, err := apicall.GetPod[scope.TiKV](ctx, f.Client, targetTiKV)
				g.Expect(err).To(gomega.BeNil())
				// Verify this is a new pod (different UID)
				g.Expect(newPod.UID).ShouldNot(gomega.Equal(originalPodUID))

				// Verify leaders are evicted
				var tikv v1alpha1.TiKV
				g.Expect(f.Client.Get(ctx, client.ObjectKeyFromObject(targetTiKV), &tikv)).To(gomega.Succeed())
				g.Expect(meta.IsStatusConditionTrue(tikv.Status.Conditions, v1alpha1.TiKVCondLeadersEvicted)).To(gomega.BeTrue())
			}, 3*time.Minute, 5*time.Second)

			ginkgo.By("Verifying TiKV instance eventually completes removal")
			f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, targetTiKV, waiter.LongTaskTimeout))

			ginkgo.By("Verifying TiKVGroup reaches desired state")
			f.WaitForTiKVGroupReady(ctx, kvg)

			ginkgo.By("Verifying final replica count")
			finalTiKVs, err := apicall.ListInstances[scope.TiKVGroup](ctx, f.Client, kvg)
			f.Must(err)
			gomega.Expect(finalTiKVs).To(gomega.HaveLen(3),
				"Should have 3 TiKV instances after scale in from 4 to 3")
		})

		ginkgo.It("Evict leaders before deleting tikv", label.P1, label.Delete, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithSlowDataMigration())
			kvg := f.MustCreateTiKV(ctx, data.WithReplicas[*runtime.TiKVGroup](4))
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			// Make sure each TiKV store has enough leaders and regions,
			// otherwise the scale-in operation will be too fast.
			workload.MustImportData(ctx, data.DefaultTiDBServiceName)

			nctx, cancel := context.WithCancel(ctx)
			ch := make(chan struct{})
			synced := make(chan struct{})
			go func() {
				defer close(ch)
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WatchUntilInstanceList[scope.TiKVGroup](
					nctx,
					f.Client,
					kvg.DeepCopy(),
					waiter.EvictLeaderBeforeStoreIsRemoving(1),
					waiter.LongTaskTimeout,
					synced,
				))
			}()
			// wait until cache is synced
			<-synced

			ginkgo.By("Initiating scale in from 4 to 3 replicas")
			patch := client.MergeFrom(kvg.DeepCopy())
			kvg.Spec.Replicas = ptr.To[int32](3)
			f.Must(f.Client.Patch(ctx, kvg, patch))

			f.WaitForTiKVGroupReady(ctx, kvg)
			cancel()
			<-ch
		})
	})

	ginkgo.Context("when scaling in TiKV with two-step deletion", func() {
		workload := f.SetupWorkload()

		ginkgo.It("should complete the full scale-in flow", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx, data.WithReplicas[*runtime.TiKVGroup](4))
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			workload.MustImportData(ctx, data.DefaultTiDBServiceName)

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
			waitForTiKVOfflineCondition(ctx, f, offlineTiKV, v1alpha1.ReasonOfflineCompleted, 5*time.Minute, true)

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
region-schedule-limit = 16
replica-schedule-limit = 8
`
			})
			kvg := f.MustCreateTiKV(ctx, data.WithReplicas[*runtime.TiKVGroup](4))
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			workload.MustImportData(ctx, data.DefaultTiDBServiceName)

			ginkgo.By("Scaling in TiKV from 4 to 3 replica")
			patch := client.MergeFrom(kvg.DeepCopy())
			kvg.Spec.Replicas = ptr.To[int32](3)
			f.Must(f.Client.Patch(ctx, kvg, patch))
			offliningKVs := findOffliningTiKVs(ctx, f, kvg, 1)
			gomega.Expect(offliningKVs).To(gomega.HaveLen(1))

			ginkgo.By("Cancelling scale-in by scaling back to 4 replicas")
			patch = client.MergeFrom(kvg.DeepCopy())
			kvg.Spec.Replicas = ptr.To[int32](4)
			f.Must(f.Client.Patch(ctx, kvg, patch))

			ginkgo.By("Verifying offline operations are cancelled")
			// When offline operation is cancelled, the TiKV instance should no longer be marked for offline
			gomega.Eventually(func(g gomega.Gomega) {
				instance := &v1alpha1.TiKV{}
				g.Expect(f.Client.Get(ctx, client.ObjectKeyFromObject(offliningKVs[0]), instance)).To(gomega.Succeed())
				g.Expect(instance.Spec.Offline).To(gomega.BeFalse(), "TiKV instance should not be marked for offline after cancellation")
				g.Expect(meta.FindStatusCondition(instance.Status.Conditions, v1alpha1.StoreOfflinedConditionType)).To(gomega.BeNil(), "StoreOffline condition should not exist")
			}, 3*time.Minute, 10*time.Second).Should(gomega.Succeed())

			ginkgo.By("check if it's evicting leaders")
			tikvGet := &v1alpha1.TiKV{}
			gomega.Expect(f.Client.Get(ctx, client.ObjectKeyFromObject(offliningKVs[0]), tikvGet)).To(gomega.Succeed())
			cond := meta.FindStatusCondition(tikvGet.Status.Conditions, v1alpha1.TiKVCondLeadersEvicted)
			gomega.Expect(cond).NotTo(gomega.BeNil(), "LeadersEvicted condition should exist")
			gomega.Expect(cond.Status).Should(gomega.Equal(metav1.ConditionFalse))
			time.Sleep(10 * time.Second)

			ginkgo.By("Verifying TiKVGroup stabilizes at 4 replicas")
			f.WaitForTiKVGroupReady(ctx, kvg)
			finalTiKVs, err := apicall.ListInstances[scope.TiKVGroup](ctx, f.Client, kvg)
			f.Must(err)
			gomega.Expect(finalTiKVs).To(gomega.HaveLen(4))
			for _, kv := range finalTiKVs {
				gomega.Expect(kv.Spec.Offline).To(gomega.BeFalse())
			}
		})

		ginkgo.It("should handle partial cancellation of scale-in", func(ctx context.Context) {
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
			workload.MustImportData(ctx, data.DefaultTiDBServiceName)

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
			// When offline operation is cancelled, the TiKV instance should no longer be marked for offline
			gomega.Eventually(func(g gomega.Gomega) {
				instance := &v1alpha1.TiKV{}
				g.Expect(f.Client.Get(ctx, client.ObjectKeyFromObject(cancelledKVs[0]), instance)).To(gomega.Succeed())
				g.Expect(instance.Spec.Offline).To(gomega.BeFalse(), "Cancelled TiKV instance should not be marked for offline")
				g.Expect(meta.FindStatusCondition(instance.Status.Conditions, v1alpha1.StoreOfflinedConditionType)).To(gomega.BeNil(), "StoreOffline condition should not exist")
			}, 3*time.Minute, 10*time.Second).Should(gomega.Succeed())

			ginkgo.By("Waiting for the remaining offline operations to complete")
			waitForTiKVOfflineCondition(ctx, f, remainingOffliningKVs[0], v1alpha1.ReasonOfflineCompleted, 5*time.Minute, true)

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

		cond := meta.FindStatusCondition(instance.Status.Conditions, v1alpha1.StoreOfflinedConditionType)
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
		f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, kv, timeout))
	}
}
