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
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/random"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	defaultUpdateWaitTime = time.Second * 30
)

// TaskUpdater is a task to scale or update PD when spec of PDGroup is changed.
func TaskUpdater(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Updater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		pdg := state.PDGroup()

		checker := action.NewUpgradeChecker[scope.PDGroup](c, state.Cluster(), logger)

		if needVersionUpgrade(pdg) && !checker.CanUpgrade(ctx, pdg) {
			// TODO(liubo02): change to Wait
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
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

		pds := state.Slice()
		topoPolicy, err := policy.NewTopologyPolicy(topos, pds...)
		if err != nil {
			return task.Fail().With("invalid topo policy, it should be validated: %w", err)
		}

		updateRevision, _, _ := state.Revision()

		wait, err := updater.New[runtime.PDTuple]().
			WithInstances(pds...).
			WithDesired(int(state.Group().Replicas())).
			WithClient(c).
			WithMaxSurge(0).
			WithMaxUnavailable(1).
			WithRevision(updateRevision).
			WithNewFactory(PDNewer(pdg, updateRevision)).
			WithAddHooks(topoPolicy).
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
	return pdg.Spec.Template.Spec.Version != pdg.Status.Version && pdg.Status.Version != ""
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
				Namespace:   pdg.Namespace,
				Name:        name,
				Labels:      coreutil.InstanceLabels[scope.PDGroup](pdg, rev),
				Annotations: coreutil.InstanceAnnotations[scope.PDGroup](pdg),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(pdg, v1alpha1.SchemeGroupVersion.WithKind("PDGroup")),
				},
			},
			Spec: v1alpha1.PDSpec{
				Cluster:        pdg.Spec.Cluster,
				Subdomain:      HeadlessServiceName(pdg.Name),
				PDTemplateSpec: *spec,
			},
		}

		peer.Annotations = maputil.MergeTo(peer.Annotations, bootAnno)

		return runtime.FromPD(peer)
	})
}
