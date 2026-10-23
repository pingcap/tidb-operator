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
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	watchtools "k8s.io/client-go/tools/watch"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/action"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("Sequential scale in TiKV", label.TiKV, label.MultipleAZ, label.P0, label.Scale, ginkgo.Serial, func() {
	f := framework.New()
	f.Setup()
	f.SetupCluster(data.WithFeatureGates(meta.TiKVSequentialScaleIn))

	ginkgo.DescribeTable("scale in from 6 to 3",
		func(ctx context.Context, rollingUpdate bool) {
			// transfer-leader-in store limits require a PD build newer than the default v8.5.2.
			pdg := f.MustCreatePD(ctx, data.WithImage[scope.PDGroup]("pingcap/pd:master"))
			kvg := f.MustCreateTiKV(ctx,
				data.WithReplicas[scope.TiKVGroup](6),
				data.WithTiKVEvenlySpreadPolicy(),
			)
			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)

			nctx, cancel := context.WithTimeout(ctx, waiter.LongTaskTimeout)
			defer cancel()
			// Watch from the initial list's resource version before changing replicas.
			// Use the ordered watch stream: relisting could hide overlapping removals.
			lw := apicall.NewInstanceListerWatcher[scope.TiKVGroup](nctx, f.Client, kvg)
			obj, err := lw.List(metav1.ListOptions{})
			f.Must(err)
			initial := obj.(*v1alpha1.TiKVList)
			gomega.Expect(initial.Items).To(gomega.HaveLen(6))
			stream, err := lw.Watch(metav1.ListOptions{ResourceVersion: initial.ResourceVersion})
			f.Must(err)
			defer stream.Stop()

			if rollingUpdate {
				// Offlining stores may also restart while waiting for their removal turn.
				// Verify the retained Pods were recreated without requiring a fixed total Pod count.
				changeTime, err := waiter.MaxPodsCreateTimestamp[scope.TiKVGroup](ctx, f.Client, kvg)
				f.Must(err)
				action.MustScaleAndRollingRestart[scope.TiKVGroup](ctx, f, kvg, 3)
				f.Must(waitForSequentialTiKVRemoval(nctx, stream, 3))
				f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiKVGroup(kvg), *changeTime, waiter.LongTaskTimeout))
			} else {
				action.MustScale[scope.TiKVGroup](ctx, f, kvg, 3)
				f.Must(waitForSequentialTiKVRemoval(nctx, stream, 3))
			}

			f.WaitForTiKVGroupReady(ctx, kvg)
			framework.MustEvenlySpread[scope.TiKVGroup](ctx, f, kvg)
		},
		ginkgo.Entry("one store at a time", false),
		ginkgo.Entry("with simultaneous rolling update", label.Update, true),
	)
})

func waitForSequentialTiKVRemoval(ctx context.Context, stream watch.Interface, count int) error {
	removing := ""
	removed := make(map[string]bool)
	_, err := watchtools.UntilWithoutRetry(ctx, stream, func(event watch.Event) (bool, error) {
		kv, ok := event.Object.(*v1alpha1.TiKV)
		if !ok {
			return false, fmt.Errorf("unexpected TiKV watch event: %s, %v", event.Type, event.Object)
		}
		if removed[kv.Name] {
			return false, nil
		}
		switch kv.Status.State {
		case v1alpha1.StoreStateRemoving, v1alpha1.StoreStateRemoved:
			if removing != "" && removing != kv.Name {
				return false, fmt.Errorf("TiKV %s entered %s while %s was still Removing", kv.Name, kv.Status.State, removing)
			}
			removing = kv.Name
		}
		// A deleting instance can be finalized without persisting Removed status.
		if kv.Status.State == v1alpha1.StoreStateRemoved || event.Type == watch.Deleted {
			if removing != kv.Name {
				return false, fmt.Errorf("TiKV %s disappeared without an observed store removal", kv.Name)
			}
			removed[kv.Name] = true
			removing = ""
		}
		return len(removed) == count, nil
	})
	return err
}
