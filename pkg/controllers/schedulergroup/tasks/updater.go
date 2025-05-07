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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/action"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/updater"
	"github.com/pingcap/tidb-operator/pkg/updater/policy"
	"github.com/pingcap/tidb-operator/pkg/utils/random"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	defaultUpdateWaitTime = time.Second * 30
)

// TaskUpdater is a task to scale or update Scheduler when spec of SchedulerGroup is changed.
func TaskUpdater(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Updater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		obj := state.Object()

		checker := action.NewUpgradeChecker[scope.SchedulerGroup](c, state.Cluster(), logger) // Changed TSOGroup to SchedulerGroup

		if needVersionUpgrade(obj) && !checker.CanUpgrade(ctx, obj) {
			// TODO(liubo02): change to Wait
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
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

		wait, err := updater.New[runtime.SchedulerTuple](). // Changed TSOTuple to SchedulerTuple
									WithInstances(is...).
									WithDesired(int(state.Group().Replicas())).
									WithClient(c).
			// TODO(liubo02): recheck it
			WithMaxSurge(0).
			WithMaxUnavailable(1).
			WithRevision(updateRevision).
			WithNewFactory(SchedulerNewer(obj, updateRevision)). // Changed TSONewer to SchedulerNewer
			WithAddHooks(topoPolicy).
			WithDelHooks(topoPolicy).
			WithUpdateHooks(topoPolicy).
			WithScaleInPreferPolicy(
				topoPolicy,
			).
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

func needVersionUpgrade(sg *v1alpha1.SchedulerGroup) bool { // Changed tg to sg (SchedulerGroup)
	return sg.Spec.Template.Spec.Version != sg.Status.Version && sg.Status.Version != ""
}

const (
	suffixLen = 6
)

func SchedulerNewer(sg *v1alpha1.SchedulerGroup, rev string) updater.NewFactory[*runtime.Scheduler] { // Changed TSONewer to SchedulerNewer, tg to sg, runtime.TSO to runtime.Scheduler
	return updater.NewFunc[*runtime.Scheduler](func() *runtime.Scheduler {
		name := fmt.Sprintf("%s-%s", sg.Name, random.Random(suffixLen))
		spec := sg.Spec.Template.Spec.DeepCopy()

		scheduler := &v1alpha1.Scheduler{ // Changed tso to scheduler
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   sg.Namespace,
				Name:        name,
				Labels:      coreutil.InstanceLabels[scope.SchedulerGroup](sg, rev), // Changed TSOGroup to SchedulerGroup
				Annotations: coreutil.InstanceAnnotations[scope.SchedulerGroup](sg), // Changed TSOGroup to SchedulerGroup
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(sg, v1alpha1.SchemeGroupVersion.WithKind("SchedulerGroup")),
				},
			},
			Spec: v1alpha1.SchedulerSpec{ // Changed TSOSpec to SchedulerSpec
				Cluster:               sg.Spec.Cluster,
				Subdomain:             HeadlessServiceName(sg.Name), // same as headless service
				SchedulerTemplateSpec: *spec,                        // Changed TSOTemplateSpec to SchedulerTemplateSpec
			},
		}

		return runtime.FromScheduler(scheduler) // Changed FromTSO to FromScheduler
	})
}
