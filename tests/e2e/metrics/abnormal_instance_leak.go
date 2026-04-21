// Copyright 2026 PingCAP, Inc.
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

package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/k8s"
	metricutil "github.com/pingcap/tidb-operator/v2/tests/e2e/utils/metrics"
)

const (
	operatorNamespace      = "tidb-admin"
	operatorDeploymentName = "tidb-operator"
	abnormalInstanceMetric = "tidb_operator_abnormal_instance"
)

var _ = ginkgo.Describe("Abnormal Instance Gauge Leak", label.TiKV, func() {
	f := framework.New()
	f.Setup()

	ginkgo.It("clears gauge series when the instance CR is deleted gracefully", label.P1, label.Delete, func(ctx context.Context) {
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx, data.WithReplicas[scope.TiKVGroup](1))
		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)

		restConfig, err := k8s.LoadConfig()
		f.Must(err)

		kvs, err := apicall.ListInstances[scope.TiKVGroup](ctx, f.Client, kvg)
		f.Must(err)
		gomega.Expect(kvs).To(gomega.HaveLen(1))
		target := kvs[0]

		ginkgo.By("waiting for the gauge series to be populated for the tikv instance")
		gomega.Eventually(func() error {
			return expectInstanceSample(ctx, restConfig, target.Namespace, target.Name, true)
		}, 3*time.Minute, 5*time.Second).Should(gomega.Succeed())

		ginkgo.By("deleting the tikv group to trigger graceful deletion of the instance")
		f.Must(f.Client.Delete(ctx, kvg))

		ginkgo.By("waiting for the gauge series to disappear")
		gomega.Eventually(func() error {
			return expectInstanceSample(ctx, restConfig, target.Namespace, target.Name, false)
		}, 3*time.Minute, 5*time.Second).Should(gomega.Succeed())
	})

	ginkgo.It("clears gauge series on force-delete that bypasses the finalizer", label.P1, label.Delete, func(ctx context.Context) {
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx, data.WithReplicas[scope.TiKVGroup](1))
		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)

		restConfig, err := k8s.LoadConfig()
		f.Must(err)

		kvs, err := apicall.ListInstances[scope.TiKVGroup](ctx, f.Client, kvg)
		f.Must(err)
		gomega.Expect(kvs).To(gomega.HaveLen(1))
		target := kvs[0]

		ginkgo.By("waiting for the gauge series to be populated for the tikv instance")
		gomega.Eventually(func() error {
			return expectInstanceSample(ctx, restConfig, target.Namespace, target.Name, true)
		}, 3*time.Minute, 5*time.Second).Should(gomega.Succeed())

		ginkgo.By("deleting the tikv CR then stripping its finalizers to force GC without waiting on the finalize task")
		f.Must(f.Client.Delete(ctx, target))

		// After Delete sets DeletionTimestamp, clearing finalizers causes the API
		// server to GC the object immediately, skipping TaskInstanceFinalizerDel.
		gomega.Eventually(func() error {
			latest := &v1alpha1.TiKV{}
			if err := f.Client.Get(ctx, client.ObjectKeyFromObject(target), latest); err != nil {
				// Already gone; nothing more to do.
				return nil
			}
			if len(latest.Finalizers) == 0 {
				return nil
			}
			patch := client.MergeFrom(latest.DeepCopy())
			latest.Finalizers = nil
			return f.Client.Patch(ctx, latest, patch)
		}, 30*time.Second, time.Second).Should(gomega.Succeed())

		ginkgo.By("waiting for the gauge series to be cleared by the informer DELETE handler")
		gomega.Eventually(func() error {
			return expectInstanceSample(ctx, restConfig, target.Namespace, target.Name, false)
		}, 2*time.Minute, 5*time.Second).Should(gomega.Succeed())

		// Stop the owner group so cleanup completes before the spec exits.
		f.Must(f.Client.Delete(ctx, kvg))
	})
})

// expectInstanceSample scrapes the operator pod and returns nil when the
// presence of a sample matching (namespace, instance) equals want.
func expectInstanceSample(ctx context.Context, restConfig *rest.Config, namespace, instance string, want bool) error {
	samples, err := metricutil.FetchOperatorMetric(ctx, restConfig,
		operatorNamespace, operatorDeploymentName, abnormalInstanceMetric)
	if err != nil {
		return err
	}
	if hasInstanceSample(samples, namespace, instance) != want {
		return fmt.Errorf("sample presence for %s/%s: got %v, want %v",
			namespace, instance, !want, want)
	}
	return nil
}

func hasInstanceSample(samples []metricutil.MetricSample, namespace, instance string) bool {
	for _, s := range samples {
		if s.Labels["namespace"] == namespace && s.Labels["instance"] == instance {
			return true
		}
	}
	return false
}
