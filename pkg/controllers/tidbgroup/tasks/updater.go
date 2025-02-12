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

// TaskUpdater is a task to scale or update TiDB when spec of TiDBGroup is changed.
func TaskUpdater(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Updater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		dbg := state.TiDBGroup()

		checker := action.NewUpgradeChecker[scope.TiDBGroup](c, state.Cluster(), logger)

		if needVersionUpgrade(dbg) && !checker.CanUpgrade(ctx, dbg) {
			// TODO(liubo02): change to Wait
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
		}

		var topos []v1alpha1.ScheduleTopology
		for _, p := range dbg.Spec.SchedulePolicies {
			switch p.Type {
			case v1alpha1.SchedulePolicyTypeEvenlySpread:
				topos = p.EvenlySpread.Topologies
			default:
				// do nothing
			}
		}

		dbs := state.Slice()
		topoPolicy, err := policy.NewTopologyPolicy(topos, dbs...)
		if err != nil {
			return task.Fail().With("invalid topo policy, it should be validated: %w", err)
		}

		updateRevision, _, _ := state.Revision()

		wait, err := updater.New[runtime.TiDBTuple]().
			WithInstances(dbs...).
			WithDesired(int(state.Group().Replicas())).
			WithClient(c).
			WithMaxSurge(0).
			WithMaxUnavailable(1).
			WithRevision(updateRevision).
			WithNewFactory(TiDBNewer(dbg, updateRevision)).
			WithAddHooks(topoPolicy).
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

func needVersionUpgrade(dbg *v1alpha1.TiDBGroup) bool {
	return dbg.Spec.Template.Spec.Version != dbg.Status.Version && dbg.Status.Version != ""
}

const (
	suffixLen = 6
)

func TiDBNewer(dbg *v1alpha1.TiDBGroup, rev string) updater.NewFactory[*runtime.TiDB] {
	return updater.NewFunc[*runtime.TiDB](func() *runtime.TiDB {
		name := fmt.Sprintf("%s-%s", dbg.Name, random.Random(suffixLen))
		spec := dbg.Spec.Template.Spec.DeepCopy()

		tidb := &v1alpha1.TiDB{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: dbg.Namespace,
				Name:      name,
				Labels: maputil.Merge(dbg.Spec.Template.Labels, map[string]string{
					v1alpha1.LabelKeyManagedBy:            v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyComponent:            v1alpha1.LabelValComponentTiDB,
					v1alpha1.LabelKeyCluster:              dbg.Spec.Cluster.Name,
					v1alpha1.LabelKeyGroup:                dbg.Name,
					v1alpha1.LabelKeyInstanceRevisionHash: rev,
				}),
				Annotations: maputil.Copy(dbg.Spec.Template.Annotations),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(dbg, v1alpha1.SchemeGroupVersion.WithKind("TiDBGroup")),
				},
			},
			Spec: v1alpha1.TiDBSpec{
				Cluster:          dbg.Spec.Cluster,
				Subdomain:        HeadlessServiceName(dbg.Name), // same as headless service
				TiDBTemplateSpec: *spec,
			},
		}

		return runtime.FromTiDB(tidb)
	})
}
