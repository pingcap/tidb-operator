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
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/go-logr/logr"

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

// TaskUpdater is a task to scale or update PD when spec of PDGroup is changed.
func TaskUpdater(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Updater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		historyCli := history.NewClient(c)
		pdg := state.PDGroup()

		selector := labels.SelectorFromSet(labels.Set{
			// TODO(liubo02): add label of managed by operator ?
			v1alpha1.LabelKeyCluster:   pdg.Spec.Cluster.Name,
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
			v1alpha1.LabelKeyGroup:     pdg.Name,
		})

		revisions, err := historyCli.ListControllerRevisions(pdg, selector)
		if err != nil {
			return task.Fail().With("cannot list controller revisions: %w", err)
		}
		history.SortControllerRevisions(revisions)

		// Get the current(old) and update(new) ControllerRevisions
		currentRevision, updateRevision, collisionCount, err := func() (*appsv1.ControllerRevision, *appsv1.ControllerRevision, int32, error) {
			// always ignore bootstrapped field in spec
			bootstrapped := pdg.Spec.Bootstrapped
			pdg.Spec.Bootstrapped = false
			defer func() {
				pdg.Spec.Bootstrapped = bootstrapped
			}()
			return revision.GetCurrentAndUpdate(pdg, revisions, historyCli, pdg)
		}()
		if err != nil {
			return task.Fail().With("cannot get revisions: %w", err)
		}
		state.CurrentRevision = currentRevision.Name
		state.UpdateRevision = updateRevision.Name
		state.CollisionCount = collisionCount

		// TODO(liubo02): add a controller to do it
		if err = revision.TruncateHistory(historyCli, state.PDSlice(), revisions,
			currentRevision, updateRevision, state.Cluster().Spec.RevisionHistoryLimit); err != nil {
			logger.Error(err, "failed to truncate history")
		}

		checker := action.NewUpgradeChecker(c, state.Cluster(), logger)

		if needVersionUpgrade(pdg) && !checker.CanUpgrade(ctx, pdg) {
			// TODO(liubo02): change to Wait
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
		}

		desired := 1
		if pdg.Spec.Replicas != nil {
			desired = int(*pdg.Spec.Replicas)
		}

		var topos []v1alpha1.ScheduleTopology
		for _, p := range pdg.Spec.SchedulePolicies {
			switch p.Type {
			case v1alpha1.SchedulePolicyTypeEvenlySpread:
				topos = p.EvenlySpread.Topologies
			default:
				// do nothing
			}
		}

		topoPolicy, err := policy.NewTopologyPolicy[*runtime.PD](topos)
		if err != nil {
			return task.Fail().With("invalid topo policy, it should be validated: %w", err)
		}

		for _, pd := range state.PDSlice() {
			topoPolicy.Add(runtime.FromPD(pd))
		}

		wait, err := updater.New[*runtime.PD]().
			WithInstances(runtime.FromPDSlice(state.PDSlice())...).
			WithDesired(desired).
			WithClient(c).
			WithMaxSurge(0).
			WithMaxUnavailable(1).
			WithRevision(state.UpdateRevision).
			WithNewFactory(PDNewer(pdg, state.UpdateRevision)).
			WithAddHooks(topoPolicy).
			WithUpdateHooks(
				policy.KeepName[*runtime.PD](),
				policy.KeepTopology[*runtime.PD](),
			).
			WithDelHooks(topoPolicy).
			WithScaleInPreferPolicy(
				NotLeaderPolicy(),
				topoPolicy,
			).
			WithUpdatePreferPolicy(
				NotLeaderPolicy(),
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

func needVersionUpgrade(pdg *v1alpha1.PDGroup) bool {
	return pdg.Spec.Version != pdg.Status.Version && pdg.Status.Version != ""
}

const (
	suffixLen = 6
)

func PDNewer(pdg *v1alpha1.PDGroup, rev string) updater.NewFactory[*runtime.PD] {
	return updater.NewFunc[*runtime.PD](func() *runtime.PD {
		name := fmt.Sprintf("%s-%s", pdg.Name, random.Random(suffixLen))
		spec := pdg.Spec.Template.Spec.DeepCopy()

		var bootAnno map[string]string
		if !pdg.Spec.Bootstrapped {
			replicas := int64(1)
			if pdg.Spec.Replicas != nil {
				replicas = int64(*pdg.Spec.Replicas)
			}
			initialNum := strconv.FormatInt(replicas, 10)
			bootAnno = map[string]string{
				v1alpha1.AnnoKeyInitialClusterNum: initialNum,
			}
		}

		peer := &v1alpha1.PD{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: pdg.Namespace,
				Name:      name,
				Labels: maputil.Merge(pdg.Spec.Template.Labels, map[string]string{
					v1alpha1.LabelKeyManagedBy:            v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyComponent:            v1alpha1.LabelValComponentPD,
					v1alpha1.LabelKeyCluster:              pdg.Spec.Cluster.Name,
					v1alpha1.LabelKeyGroup:                pdg.Name,
					v1alpha1.LabelKeyInstanceRevisionHash: rev,
				}),
				Annotations: maputil.Merge(pdg.Spec.Template.Annotations, bootAnno),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(pdg, v1alpha1.SchemeGroupVersion.WithKind("PDGroup")),
				},
			},
			Spec: v1alpha1.PDSpec{
				Cluster:        pdg.Spec.Cluster,
				Version:        pdg.Spec.Version,
				Subdomain:      HeadlessServiceName(pdg.Name),
				PDTemplateSpec: *spec,
			},
		}

		return runtime.FromPD(peer)
	})
}
