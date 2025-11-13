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

// TaskUpdater is a task to scale or update TSO when spec of TSOGroup is changed.
func TaskUpdater(state *ReconcileContext, c client.Client, af tracker.AllocateFactory) task.Task {
	return task.NameTaskFunc("Updater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		obj := state.Object()

		checker := action.NewUpgradeChecker[scope.TSOGroup](c, state.Cluster(), logger)

		if needVersionUpgrade(obj) && !checker.CanUpgrade(ctx, obj) {
			// TODO(liubo02): change to Wait
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
		}

		retryAfter := coreutil.RetryIfInstancesReadyButNotAvailable[scope.TSO](
			state.InstanceSlice(),
			coreutil.MinReadySeconds[scope.TSOGroup](obj),
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

		is := state.Slice()
		topoPolicy, err := policy.NewTopologyPolicy(topos, updateRevision, is...)
		if err != nil {
			return task.Fail().With("invalid topo policy, it should be validated: %w", err)
		}

		var instances []string
		for _, in := range is {
			instances = append(instances, in.Name)
		}

		allocator := af.New(obj.Namespace, obj.Name, instances...)
		wait, err := updater.New[runtime.TSOTuple]().
			WithInstances(is...).
			WithDesired(int(state.Group().Replicas())).
			WithClient(c).
			// TODO(liubo02): recheck it
			WithMaxSurge(0).
			WithMaxUnavailable(1).
			WithRevision(updateRevision).
			WithNewFactory(TSONewer(obj, updateRevision, state.FeatureGates())).
			WithAddHooks(
				updater.AllocateName[*runtime.TSO](allocator),
				topoPolicy,
			).
			WithDelHooks(topoPolicy).
			WithUpdateHooks(topoPolicy).
			WithScaleInPreferPolicy(
				NotLeaderPolicy(),
				topoPolicy.PolicyScaleIn(),
			).
			WithUpdatePreferPolicy(
				NotLeaderPolicy(),
			).
			WithMinReadySeconds(coreutil.MinReadySeconds[scope.TSOGroup](obj)).
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

func needVersionUpgrade(tg *v1alpha1.TSOGroup) bool {
	return tg.Spec.Template.Spec.Version != tg.Status.Version && tg.Status.Version != ""
}

func TSONewer(tg *v1alpha1.TSOGroup, rev string, fg features.Gates) updater.NewFactory[*runtime.TSO] {
	return updater.NewFunc[*runtime.TSO](func() *runtime.TSO {
		spec := tg.Spec.Template.Spec.DeepCopy()

		tso := &v1alpha1.TSO{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: tg.Namespace,
				// Name will be allocated by updater.AllocateName
				Labels:      coreutil.InstanceLabels[scope.TSOGroup](tg, rev),
				Annotations: coreutil.InstanceAnnotations[scope.TSOGroup](tg),
				Finalizers:  []string{metav1alpha1.Finalizer},
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(tg, v1alpha1.SchemeGroupVersion.WithKind("TSOGroup")),
				},
			},
			Spec: v1alpha1.TSOSpec{
				Cluster:         tg.Spec.Cluster,
				Features:        tg.Spec.Features,
				Subdomain:       HeadlessServiceName(tg.Name), // same as headless service
				TSOTemplateSpec: *spec,
			},
		}
		if fg.Enabled(metav1alpha1.ClusterSubdomain) {
			tso.Spec.Subdomain = coreutil.ClusterSubdomain(tg.Spec.Cluster.Name)
		}

		return runtime.FromTSO(tso)
	})
}
