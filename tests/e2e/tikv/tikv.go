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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/cert"
)

var _ = ginkgo.Describe("TiKV", label.TiKV, func() {
	f := framework.New()
	f.Setup()

	ginkgo.DescribeTableSubtree("Leader Eviction", label.P1,
		func(tls bool) {
			if tls {
				f.SetupCluster(data.WithClusterTLS())
			}

			// NOTE(liubo02): this case is failed in e2e env because of the cgroup v2.
			// Enable it if env is fixed.
			ginkgo.PIt("leader evicted when delete tikv pod directly", func(ctx context.Context) {
				if tls {
					ns := f.Cluster.Namespace
					cn := f.Cluster.Name
					f.Must(cert.InstallTiDBIssuer(ctx, f.Client, ns, cn))
					f.Must(cert.InstallTiDBCertificates(ctx, f.Client, ns, cn, "dbg"))
					f.Must(cert.InstallTiDBComponentsCertificates(ctx, f.Client, ns, cn, "pdg", "kvg", "dbg", "flashg", "cdcg"))
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
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx, data.WithReplicas[*runtime.TiKVGroup](4))
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			// Make sure each TiKV store has enough leaders and regions,
			// otherwise the scale-in operation will be too fast.
			workload.MustImportData(ctx, data.DefaultTiDBServiceName, "root", "", "", 500)

			ginkgo.By("Initiating scale in from 4 to 3 replicas")
			patch := client.MergeFrom(kvg.DeepCopy())
			kvg.Spec.Replicas = ptr.To[int32](3)
			f.Must(f.Client.Patch(ctx, kvg, patch))

			ginkgo.By("Finding the TiKV instance that is being scaled in")
			var targetTiKV *v1alpha1.TiKV
			gomega.Eventually(func() bool {
				updatedTiKVs, err := apicall.ListInstances[scope.TiKVGroup](ctx, f.Client, kvg)
				if err != nil {
					return false
				}

				for _, tikv := range updatedTiKVs {
					if !tikv.DeletionTimestamp.IsZero() && tikv.Status.State == "Removing" {
						targetTiKV = tikv
						ginkgo.By(fmt.Sprintf("Found TiKV instance being scaled in: %s", tikv.Name))
						return true
					}
				}
				return false
			}, 2*time.Minute, 3*time.Second).Should(gomega.BeTrue(),
				"Should find a TiKV instance with deletion timestamp during scale in")

			ginkgo.By("Recording original pod information")
			originalPod := getPodForTiKV(ctx, f, targetTiKV)
			originalPodUID := originalPod.UID

			ginkgo.By("Simulating manual pod deletion during graceful shutdown")
			// This simulates the race condition where user manually deletes the pod
			// while the operator is trying to gracefully remove the store from PD
			f.Must(f.Client.Delete(ctx, originalPod, client.GracePeriodSeconds(0)))

			ginkgo.By("Verifying operator recreates the pod during store removal")
			// The operator should recreate the pod to ensure graceful store removal can complete
			gomega.Eventually(func() bool {
				newPod := &corev1.Pod{}
				err := f.Client.Get(ctx, client.ObjectKey{
					Namespace: targetTiKV.Namespace,
					Name:      originalPod.Name,
				}, newPod)
				if err != nil {
					return false
				}
				// Verify this is a new pod (different UID)
				return newPod.UID != originalPodUID
			}, 3*time.Minute, 5*time.Second).Should(gomega.BeTrue(),
				"Operator should recreate pod with different UID during store removal")

			ginkgo.By("Verifying TiKV instance eventually completes removal")
			waitForTiKVInstanceCleanup(ctx, f, targetTiKV, 8*time.Minute)

			ginkgo.By("Verifying TiKVGroup reaches desired state")
			f.WaitForTiKVGroupReady(ctx, kvg)

			ginkgo.By("Verifying final replica count")
			finalTiKVs, err := apicall.ListInstances[scope.TiKVGroup](ctx, f.Client, kvg)
			f.Must(err)
			gomega.Expect(finalTiKVs).To(gomega.HaveLen(3),
				"Should have 3 TiKV instances after scale in from 4 to 3")
		})
	})
})

// getPodForTiKV gets the pod for a TiKV instance.
func getPodForTiKV(ctx context.Context, f *framework.Framework, tikv *v1alpha1.TiKV) *corev1.Pod {
	pod := &corev1.Pod{}
	f.Must(f.Client.Get(ctx, client.ObjectKey{
		Namespace: tikv.Namespace,
		Name:      coreutil.PodName[scope.TiKV](tikv),
	}, pod))
	return pod
}

// waitForTiKVInstanceCleanup waits for a TiKV instance to be completely removed
func waitForTiKVInstanceCleanup(ctx context.Context, f *framework.Framework, tikv *v1alpha1.TiKV, timeout time.Duration) {
	gomega.Eventually(func() error {
		instance := &v1alpha1.TiKV{}
		err := f.Client.Get(ctx, client.ObjectKey{
			Namespace: tikv.Namespace,
			Name:      tikv.Name,
		}, instance)
		if err != nil {
			return nil // Successfully deleted
		}
		if !instance.DeletionTimestamp.IsZero() {
			return fmt.Errorf("TiKV instance %s is stuck in Terminating state", tikv.Name)
		}
		return fmt.Errorf("TiKV instance %s still exists", tikv.Name)
	}, timeout, 15*time.Second).Should(gomega.Succeed(),
		fmt.Sprintf("TiKV instance %s should be cleaned up properly", tikv.Name))
}
