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

func TaskUpdater(state *ReconcileContext, c client.Client, af tracker.AllocateFactory) task.Task {
	return task.NameTaskFunc("Updater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		dwg := state.DMWorkerGroup()

		checker := action.NewUpgradeChecker[scope.DMWorkerGroup](c, state.Cluster(), logger)

		if needVersionUpgrade(dwg) && !checker.CanUpgrade(ctx, dwg) {
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
		}

		retryAfter := coreutil.RetryIfInstancesReadyButNotAvailable[scope.DMWorker](state.InstanceSlice(), coreutil.MinReadySeconds[scope.DMWorkerGroup](dwg))
		if retryAfter != 0 {
			return task.Retry(retryAfter).With("wait until no instances is ready but not available")
		}

		var topos []v1alpha1.ScheduleTopology
		for _, p := range dwg.Spec.SchedulePolicies {
			switch p.Type {
			case v1alpha1.SchedulePolicyTypeEvenlySpread:
				topos = p.EvenlySpread.Topologies
			default:
			}
		}

		updateRevision, _, _ := state.Revision()

		dws := state.Slice()
		topoPolicy, err := policy.NewTopologyPolicy(topos, updateRevision, dws...)
		if err != nil {
			return task.Fail().With("invalid topo policy, it should be validated: %w", err)
		}

		var instances []string
		for _, in := range dws {
			instances = append(instances, in.Name)
		}

		allocator := af.New(dwg.Namespace, dwg.Name, instances...)

		wait, err := updater.New[runtime.DMWorkerTuple]().
			WithInstances(dws...).
			WithDesired(int(state.Group().Replicas())).
			WithClient(c).
			WithMaxSurge(0).
			WithMaxUnavailable(1).
			WithRevision(updateRevision).
			WithNewFactory(DMWorkerNewer(dwg, updateRevision)).
			WithAddHooks(
				updater.AllocateName[*runtime.DMWorker](allocator),
				topoPolicy,
			).
			WithDelHooks(topoPolicy).
			WithUpdateHooks(topoPolicy).
			WithScaleInPreferPolicy(topoPolicy.PolicyScaleIn()).
			WithMinReadySeconds(coreutil.MinReadySeconds[scope.DMWorkerGroup](dwg)).
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

func needVersionUpgrade(dwg *v1alpha1.DMWorkerGroup) bool {
	return dwg.Spec.Template.Spec.Version != dwg.Status.Version && dwg.Status.Version != ""
}

func DMWorkerNewer(dwg *v1alpha1.DMWorkerGroup, rev string) updater.NewFactory[*runtime.DMWorker] {
	return updater.NewFunc[*runtime.DMWorker](func() *runtime.DMWorker {
		spec := dwg.Spec.Template.Spec.DeepCopy()

		dw := &v1alpha1.DMWorker{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: dwg.Namespace,
				// Name will be allocated by updater.AllocateName
				Labels:      coreutil.InstanceLabels[scope.DMWorkerGroup](dwg, rev),
				Annotations: coreutil.InstanceAnnotations[scope.DMWorkerGroup](dwg),
				Finalizers:  []string{metav1alpha1.Finalizer},
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(dwg, v1alpha1.SchemeGroupVersion.WithKind("DMWorkerGroup")),
				},
			},
			Spec: v1alpha1.DMWorkerSpec{
				Cluster:              dwg.Spec.Cluster,
				DMGroupRef:           dwg.Spec.DMGroupRef,
				Features:             dwg.Spec.Features,
				Subdomain:            coreutil.HeadlessServiceName[scope.DMWorkerGroup](dwg),
				DMWorkerTemplateSpec: *spec,
			},
		}

		return runtime.FromDMWorker(dw)
	})
}
