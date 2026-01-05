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

// TaskUpdater is a task to scale or update ResourceManager when spec of ResourceManagerGroup is changed.
func TaskUpdater(state *ReconcileContext, c client.Client, af tracker.AllocateFactory) task.Task {
	return task.NameTaskFunc("Updater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		obj := state.Object()

		checker := action.NewUpgradeChecker[scope.ResourceManagerGroup](c, state.Cluster(), logger)

		if needVersionUpgrade(obj) && !checker.CanUpgrade(ctx, obj) {
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
		}

		retryAfter := coreutil.RetryIfInstancesReadyButNotAvailable[scope.ResourceManager](
			state.InstanceSlice(),
			coreutil.MinReadySeconds[scope.ResourceManagerGroup](obj),
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
		wait, err := updater.New[runtime.ResourceManagerTuple]().
			WithInstances(is...).
			WithDesired(int(state.Group().Replicas())).
			WithClient(c).
			WithMaxSurge(0).
			WithMaxUnavailable(1).
			WithRevision(updateRevision).
			WithNewFactory(ResourceManagerNewer(obj, updateRevision, state.FeatureGates())).
			WithAddHooks(
				updater.AllocateName[*runtime.ResourceManager](allocator),
				topoPolicy,
			).
			WithDelHooks(topoPolicy).
			WithUpdateHooks(topoPolicy).
			WithScaleInPreferPolicy(topoPolicy.PolicyScaleIn()).
			WithMinReadySeconds(coreutil.MinReadySeconds[scope.ResourceManagerGroup](obj)).
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

func needVersionUpgrade(rmg *v1alpha1.ResourceManagerGroup) bool {
	return rmg.Spec.Template.Spec.Version != rmg.Status.Version && rmg.Status.Version != ""
}

func ResourceManagerNewer(rmg *v1alpha1.ResourceManagerGroup, rev string, fg features.Gates) updater.NewFactory[*runtime.ResourceManager] {
	return updater.NewFunc[*runtime.ResourceManager](func() *runtime.ResourceManager {
		spec := rmg.Spec.Template.Spec.DeepCopy()

		rm := &v1alpha1.ResourceManager{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: rmg.Namespace,
				// Name will be allocated by updater.AllocateName
				Labels:      coreutil.InstanceLabels[scope.ResourceManagerGroup](rmg, rev),
				Annotations: coreutil.InstanceAnnotations[scope.ResourceManagerGroup](rmg),
				Finalizers:  []string{metav1alpha1.Finalizer},
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(rmg, v1alpha1.SchemeGroupVersion.WithKind("ResourceManagerGroup")),
				},
			},
			Spec: v1alpha1.ResourceManagerSpec{
				Cluster:                     rmg.Spec.Cluster,
				Features:                    rmg.Spec.Features,
				Subdomain:                   coreutil.HeadlessServiceName[scope.ResourceManagerGroup](rmg),
				ResourceManagerTemplateSpec: *spec,
			},
		}

		if fg.Enabled(metav1alpha1.ClusterSubdomain) {
			rm.Spec.Subdomain = coreutil.ClusterSubdomain(rmg.Spec.Cluster.Name)
		}

		return runtime.FromResourceManager(rm)
	})
}
