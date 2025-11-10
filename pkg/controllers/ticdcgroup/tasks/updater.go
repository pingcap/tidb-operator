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
	"github.com/pingcap/tidb-operator/pkg/action"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/reloadable"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/updater"
	"github.com/pingcap/tidb-operator/pkg/updater/policy"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/pkg/utils/tracker"
)

const (
	defaultUpdateWaitTime = time.Second * 30
)

// TaskUpdater is a task to scale or update TiCDC when spec of TiCDCGroup is changed.
func TaskUpdater(state *ReconcileContext, c client.Client, af tracker.AllocateFactory) task.Task {
	return task.NameTaskFunc("Updater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		cdcg := state.TiCDCGroup()

		checker := action.NewUpgradeChecker[scope.TiCDCGroup](c, state.Cluster(), logger)

		if needVersionUpgrade(cdcg) && !checker.CanUpgrade(ctx, cdcg) {
			// TODO(liubo02): change to Wait
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
		}

		retryAfter := coreutil.RetryIfInstancesReadyButNotAvailable[scope.TiCDC](
			state.InstanceSlice(),
			coreutil.MinReadySeconds[scope.TiCDCGroup](cdcg),
		)
		if retryAfter != 0 {
			return task.Retry(retryAfter).With("wait until no instances is ready but not available")
		}

		var topos []v1alpha1.ScheduleTopology
		for _, p := range cdcg.Spec.SchedulePolicies {
			switch p.Type {
			case v1alpha1.SchedulePolicyTypeEvenlySpread:
				topos = p.EvenlySpread.Topologies
			default:
				// do nothing
			}
		}

		updateRevision, _, _ := state.Revision()

		cdcs := state.Slice()
		topoPolicy, err := policy.NewTopologyPolicy(topos, updateRevision, cdcs...)
		if err != nil {
			return task.Fail().With("invalid topo policy, it should be validated: %w", err)
		}

		needUpdate, needRestart := precheckInstances(cdcg, runtime.ToTiCDCSlice(cdcs), updateRevision)
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
		for _, in := range cdcs {
			instances = append(instances, in.Name)
		}

		allocator := af.New(cdcg.Namespace, cdcg.Name, instances...)
		// TODO: get the real time owner info from TiCDC and prefer non-owner when scaling in or updating
		wait, err := updater.New[runtime.TiCDCTuple]().
			WithInstances(cdcs...).
			WithDesired(int(state.Group().Replicas())).
			WithClient(c).
			WithMaxSurge(maxSurge).
			WithMaxUnavailable(maxUnavailable).
			WithRevision(updateRevision).
			WithNewFactory(TiCDCNewer(cdcg, updateRevision, state.FeatureGates())).
			WithAddHooks(
				updater.AllocateName[*runtime.TiCDC](allocator),
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
			WithMinReadySeconds(coreutil.MinReadySeconds[scope.TiCDCGroup](cdcg)).
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

func needVersionUpgrade(cdcg *v1alpha1.TiCDCGroup) bool {
	return cdcg.Spec.Template.Spec.Version != cdcg.Status.Version && cdcg.Status.Version != ""
}

func TiCDCNewer(cdcg *v1alpha1.TiCDCGroup, rev string, fg features.Gates) updater.NewFactory[*runtime.TiCDC] {
	return updater.NewFunc[*runtime.TiCDC](func() *runtime.TiCDC {
		spec := cdcg.Spec.Template.Spec.DeepCopy()

		ticdc := &v1alpha1.TiCDC{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cdcg.Namespace,
				// Name will be allocated by updater.AllocateName
				Labels:      coreutil.InstanceLabels[scope.TiCDCGroup](cdcg, rev),
				Annotations: coreutil.InstanceAnnotations[scope.TiCDCGroup](cdcg),
				Finalizers:  []string{metav1alpha1.Finalizer},
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(cdcg, v1alpha1.SchemeGroupVersion.WithKind("TiCDCGroup")),
				},
			},
			Spec: v1alpha1.TiCDCSpec{
				Cluster:           cdcg.Spec.Cluster,
				Features:          cdcg.Spec.Features,
				Subdomain:         HeadlessServiceName(cdcg.Name), // same as headless service
				TiCDCTemplateSpec: *spec,
			},
		}
		if fg.Enabled(metav1alpha1.ClusterSubdomain) {
			ticdc.Spec.Subdomain = coreutil.ClusterSubdomain(cdcg.Spec.Cluster.Name)
		}

		return runtime.FromTiCDC(ticdc)
	})
}

func precheckInstances(cdcg *v1alpha1.TiCDCGroup, cdcs []*v1alpha1.TiCDC, updateRevision string) (needUpdate, needRestart bool) {
	if len(cdcs) != int(coreutil.Replicas[scope.TiCDCGroup](cdcg)) {
		needUpdate = true
	}
	for _, cdc := range cdcs {
		if coreutil.UpdateRevision[scope.TiCDC](cdc) == updateRevision {
			continue
		}

		needUpdate = true
		if !reloadable.CheckTiCDC(cdcg, cdc) {
			needRestart = true
		}
	}

	return needUpdate, needRestart
}
