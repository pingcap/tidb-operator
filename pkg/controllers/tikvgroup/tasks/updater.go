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
	"k8s.io/apimachinery/pkg/labels"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/action"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/updater"
	"github.com/pingcap/tidb-operator/pkg/updater/policy"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s/revision"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/random"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/history"
)

const (
	defaultUpdateWaitTime = time.Second * 30
)

// TaskUpdater is a task to scale or update PD when spec of TiKVGroup is changed.
func TaskUpdater(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Updater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		historyCli := history.NewClient(c)
		kvg := state.TiKVGroup()

		selector := labels.SelectorFromSet(labels.Set{
			// TODO(liubo02): add label of managed by operator ?
			v1alpha1.LabelKeyCluster:   kvg.Spec.Cluster.Name,
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
			v1alpha1.LabelKeyGroup:     kvg.Name,
		})

		revisions, err := historyCli.ListControllerRevisions(kvg, selector)
		if err != nil {
			return task.Fail().With("cannot list controller revisions: %w", err)
		}
		history.SortControllerRevisions(revisions)

		// Get the current(old) and update(new) ControllerRevisions.
		currentRevision, updateRevision, collisionCount, err := revision.GetCurrentAndUpdate(kvg, revisions, historyCli, kvg)
		if err != nil {
			return task.Fail().With("cannot get revisions: %w", err)
		}
		state.CurrentRevision = currentRevision.Name
		state.UpdateRevision = updateRevision.Name
		state.CollisionCount = collisionCount

		// TODO(liubo02): add a controller to do it
		if err = revision.TruncateHistory(historyCli, state.TiKVSlice(), revisions,
			currentRevision, updateRevision, state.Cluster().Spec.RevisionHistoryLimit); err != nil {
			logger.Error(err, "failed to truncate history")
		}

		checker := action.NewUpgradeChecker(c, state.Cluster(), logger)

		if needVersionUpgrade(kvg) && !checker.CanUpgrade(ctx, kvg) {
			// TODO(liubo02): change to Wait
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
		}

		desired := 1
		if kvg.Spec.Replicas != nil {
			desired = int(*kvg.Spec.Replicas)
		}

		var topos []v1alpha1.ScheduleTopology
		for _, p := range kvg.Spec.SchedulePolicies {
			switch p.Type {
			case v1alpha1.SchedulePolicyTypeEvenlySpread:
				topos = p.EvenlySpread.Topologies
			default:
				// do nothing
			}
		}

		topoPolicy, err := policy.NewTopologyPolicy[*runtime.TiKV](topos)
		if err != nil {
			return task.Fail().With("invalid topo policy, it should be validated: %w", err)
		}

		for _, tikv := range state.TiKVSlice() {
			topoPolicy.Add(runtime.FromTiKV(tikv))
		}

		wait, err := updater.New[runtime.TiKVTuple]().
			WithInstances(runtime.FromTiKVSlice(state.TiKVSlice())...).
			WithDesired(desired).
			WithClient(c).
			WithMaxSurge(0).
			WithMaxUnavailable(1).
			WithRevision(state.UpdateRevision).
			WithNewFactory(TiKVNewer(kvg, state.UpdateRevision)).
			WithAddHooks(topoPolicy).
			WithUpdateHooks(
				policy.KeepName[*runtime.TiKV](),
				policy.KeepTopology[*runtime.TiKV](),
			).
			WithDelHooks(topoPolicy).
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

func needVersionUpgrade(kvg *v1alpha1.TiKVGroup) bool {
	return kvg.Spec.Version != kvg.Status.Version && kvg.Status.Version != ""
}

const (
	suffixLen = 6
)

func TiKVNewer(kvg *v1alpha1.TiKVGroup, rev string) updater.NewFactory[*runtime.TiKV] {
	return updater.NewFunc[*runtime.TiKV](func() *runtime.TiKV {
		name := fmt.Sprintf("%s-%s", kvg.Name, random.Random(suffixLen))

		spec := kvg.Spec.Template.Spec.DeepCopy()

		tikv := &v1alpha1.TiKV{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: kvg.Namespace,
				Name:      name,
				Labels: maputil.Merge(kvg.Spec.Template.Labels, map[string]string{
					v1alpha1.LabelKeyManagedBy:            v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyComponent:            v1alpha1.LabelValComponentTiKV,
					v1alpha1.LabelKeyCluster:              kvg.Spec.Cluster.Name,
					v1alpha1.LabelKeyGroup:                kvg.Name,
					v1alpha1.LabelKeyInstanceRevisionHash: rev,
				}),
				Annotations: maputil.Copy(kvg.Spec.Template.Annotations),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(kvg, v1alpha1.SchemeGroupVersion.WithKind("TiKVGroup")),
				},
			},
			Spec: v1alpha1.TiKVSpec{
				Cluster:          kvg.Spec.Cluster,
				Version:          kvg.Spec.Version,
				Subdomain:        HeadlessServiceName(kvg.Name),
				TiKVTemplateSpec: *spec,
			},
		}

		return runtime.FromTiKV(tikv)
	})
}
