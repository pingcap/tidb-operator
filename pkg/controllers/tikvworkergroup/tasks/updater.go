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

package tasks

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/action"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/reloadable"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/updater"
	"github.com/pingcap/tidb-operator/v2/pkg/updater/policy"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

const (
	defaultUpdateWaitTime = time.Second * 30
)

// TaskUpdater is a task to scale or update TiKVWorker when spec of TiKVWorkerGroup is changed.
func TaskUpdater(state *ReconcileContext, c client.Client, af tracker.AllocateFactory) task.Task {
	return task.NameTaskFunc("Updater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		obj := state.Object()

		checker := action.NewUpgradeChecker[scope.TiKVWorkerGroup](c, state.Cluster(), logger)

		if needVersionUpgrade(obj) && !checker.CanUpgrade(ctx, obj) {
			// TODO(liubo02): change to Wait
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
		}

		retryAfter := coreutil.RetryIfInstancesReadyButNotAvailable[scope.TiKVWorker](
			state.InstanceSlice(),
			coreutil.MinReadySeconds[scope.TiKVWorkerGroup](obj),
		)
		if retryAfter != 0 {
			return task.Retry(retryAfter).With("wait until no instances is ready but not available")
		}

		var topos []v1alpha1.ScheduleTopology
		for _, p := range obj.Spec.SchedulePolicies {
			switch p.Type {
			case v1alpha1.SchedulePolicyTypeEvenlySpread:
				topos = p.EvenlySpread.Topologies
			default:
				// do nothing
			}
		}

		updateRevision, _, _ := state.Revision()

		ws := state.Slice()
		topoPolicy, err := policy.NewTopologyPolicy(topos, updateRevision, ws...)
		if err != nil {
			return task.Fail().With("invalid topo policy, it should be validated: %w", err)
		}

		needUpdate, needRestart := precheckInstances(obj, runtime.ToTiKVWorkerSlice(ws), updateRevision)
		if !needUpdate {
			return task.Complete().With("all instances are synced")
		}

		maxSurge, maxUnavailable := 0, 1
		noUpdate := false
		if needRestart {
			maxSurge, maxUnavailable = 1, 0
			noUpdate = true
		}

		var instances []string
		for _, in := range ws {
			instances = append(instances, in.Name)
		}

		allocator := af.New(obj.Namespace, obj.Name, instances...)
		wait, err := updater.New[runtime.TiKVWorkerTuple]().
			WithInstances(ws...).
			WithDesired(int(state.Group().Replicas())).
			WithClient(c).
			// TODO(liubo02): recheck it
			WithMaxSurge(maxSurge).
			WithMaxUnavailable(maxUnavailable).
			WithRevision(updateRevision).
			WithNewFactory(TiKVWorkerNewer(obj, updateRevision, state.FeatureGates())).
			WithAddHooks(
				updater.AllocateName[*runtime.TiKVWorker](allocator),
				topoPolicy,
			).
			WithDelHooks(topoPolicy).
			WithUpdateHooks(topoPolicy).
			WithScaleInPreferPolicy(
				topoPolicy.PolicyScaleIn(),
			).
			WithUpdatePreferPolicy(
				topoPolicy.PolicyUpdate(),
			).
			WithNoInPaceUpdate(noUpdate).
			WithMinReadySeconds(coreutil.MinReadySeconds[scope.TiKVWorkerGroup](obj)).
			Build().
			Do(ctx)
		if err != nil {
			return task.Fail().With("cannot update instances: %w", err)
		}
		if wait {
			return task.Wait().With("wait for all instances ready")
		}
		return task.Complete().With("all instances are synced")
	})
}

func needVersionUpgrade(wg *v1alpha1.TiKVWorkerGroup) bool {
	return wg.Spec.Template.Spec.Version != wg.Status.Version && wg.Status.Version != ""
}

func precheckInstances(wg *v1alpha1.TiKVWorkerGroup, ws []*v1alpha1.TiKVWorker, updateRevision string) (needUpdate, needRestart bool) {
	if len(ws) != int(coreutil.Replicas[scope.TiKVWorkerGroup](wg)) {
		needUpdate = true
	}
	for _, w := range ws {
		if coreutil.UpdateRevision[scope.TiKVWorker](w) == updateRevision {
			continue
		}

		needUpdate = true
		if !reloadable.CheckTiKVWorker(wg, w) {
			needRestart = true
		}
	}

	return needUpdate, needRestart
}

func TiKVWorkerNewer(wg *v1alpha1.TiKVWorkerGroup, rev string, fg features.Gates) updater.NewFactory[*runtime.TiKVWorker] {
	return updater.NewFunc[*runtime.TiKVWorker](func() *runtime.TiKVWorker {
		spec := wg.Spec.Template.Spec.DeepCopy()

		tso := &v1alpha1.TiKVWorker{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: wg.Namespace,
				// Name will be allocated by updater.AllocateName
				Labels:      coreutil.InstanceLabels[scope.TiKVWorkerGroup](wg, rev),
				Annotations: coreutil.InstanceAnnotations[scope.TiKVWorkerGroup](wg),
				Finalizers:  []string{metav1alpha1.Finalizer},
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(wg, v1alpha1.SchemeGroupVersion.WithKind("TiKVWorkerGroup")),
				},
			},
			Spec: v1alpha1.TiKVWorkerSpec{
				Cluster:                wg.Spec.Cluster,
				Features:               wg.Spec.Features,
				Subdomain:              coreutil.HeadlessServiceName[scope.TiKVWorkerGroup](wg), // same as headless service
				TiKVWorkerTemplateSpec: *spec,
			},
		}
		if fg.Enabled(metav1alpha1.ClusterSubdomain) {
			tso.Spec.Subdomain = coreutil.ClusterSubdomain(wg.Spec.Cluster.Name)
		}

		return runtime.FromTiKVWorker(tso)
	})
}
