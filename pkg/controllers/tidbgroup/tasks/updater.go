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

// TaskUpdater is a task to scale or update TiDB when spec of TiDBGroup is changed.
func TaskUpdater(state *ReconcileContext, c client.Client, af tracker.AllocateFactory) task.Task {
	return task.NameTaskFunc("Updater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		dbg := state.TiDBGroup()

		checker := action.NewUpgradeChecker[scope.TiDBGroup](c, state.Cluster(), logger)

		if needVersionUpgrade(dbg) && !checker.CanUpgrade(ctx, dbg) {
			// TODO(liubo02): change to Wait
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
		}

		retryAfter := coreutil.RetryIfInstancesReadyButNotAvailable[scope.TiDB](
			state.InstanceSlice(),
			coreutil.MinReadySeconds[scope.TiDBGroup](dbg),
		)
		if retryAfter != 0 {
			return task.Retry(retryAfter).With("wait until no instances is ready but not available")
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

		updateRevision, _, _ := state.Revision()

		dbs := state.Slice()
		topoPolicy, err := policy.NewTopologyPolicy(topos, updateRevision, dbs...)
		if err != nil {
			return task.Fail().With("invalid topo policy, it should be validated: %w", err)
		}

		needUpdate, needRestart := precheckInstances(dbg, runtime.ToTiDBSlice(dbs), updateRevision)
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
		for _, in := range dbs {
			instances = append(instances, in.Name)
		}

		allocator := af.New(dbg.Namespace, dbg.Name, instances...)

		wait, err := updater.New[runtime.TiDBTuple]().
			WithInstances(dbs...).
			WithDesired(int(state.Group().Replicas())).
			WithClient(c).
			WithMaxSurge(maxSurge).
			WithMaxUnavailable(maxUnavailable).
			WithRevision(updateRevision).
			WithNewFactory(newFactory(dbg, updateRevision, state.FeatureGates())).
			WithAddHooks(
				updater.AllocateName[*runtime.TiDB](allocator),
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
			WithMinReadySeconds(coreutil.MinReadySeconds[scope.TiDBGroup](dbg)).
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

func precheckInstances(dbg *v1alpha1.TiDBGroup, dbs []*v1alpha1.TiDB, updateRevision string) (needUpdate, needRestart bool) {
	if len(dbs) != int(coreutil.Replicas[scope.TiDBGroup](dbg)) {
		needUpdate = true
	}
	for _, db := range dbs {
		if coreutil.UpdateRevision[scope.TiDB](db) == updateRevision {
			continue
		}

		needUpdate = true
		if !reloadable.CheckTiDB(dbg, db) {
			needRestart = true
		}
	}

	return needUpdate, needRestart
}

type factory struct {
	dbg *v1alpha1.TiDBGroup
	rev string
	fg  features.Gates
}

func (f *factory) New() *runtime.TiDB {
	dbg := f.dbg

	tidb := &v1alpha1.TiDB{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dbg.Namespace,
			// Name will be allocated by updater.AllocateName
		},
	}

	tidb = f.updateTiDB(tidb)

	return runtime.FromTiDB(tidb)
}

func (f *factory) Adopt() (*runtime.TiDB, updater.UnlockFunc, bool) {
	return nil, nil, false
}

func (f *factory) updateTiDB(tidb *v1alpha1.TiDB) *v1alpha1.TiDB {
	spec := f.dbg.Spec.Template.Spec.DeepCopy()

	tidb.Labels = coreutil.InstanceLabels[scope.TiDBGroup](f.dbg, f.rev)
	tidb.Annotations = coreutil.InstanceAnnotations[scope.TiDBGroup](f.dbg)
	tidb.Finalizers = []string{metav1alpha1.Finalizer}
	tidb.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(f.dbg, v1alpha1.SchemeGroupVersion.WithKind("TiDBGroup")),
	}

	tidb.Spec.Cluster = f.dbg.Spec.Cluster
	tidb.Spec.Features = f.dbg.Spec.Features
	tidb.Spec.Subdomain = HeadlessServiceName(f.dbg.Name) // same as headless service
	tidb.Spec.TiDBTemplateSpec = *spec

	return tidb
}

func newFactory(dbg *v1alpha1.TiDBGroup, rev string, fg features.Gates) updater.NewFactory[*runtime.TiDB] {
	return &factory{
		dbg: dbg,
		rev: rev,
		fg:  fg,
	}
}
