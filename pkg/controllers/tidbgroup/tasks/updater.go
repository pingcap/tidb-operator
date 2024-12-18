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
	"fmt"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/updater"
	"github.com/pingcap/tidb-operator/pkg/updater/policy"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s/revision"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/random"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/history"
)

// TaskUpdater is a task for updating TiDBGroup when its spec is changed.
type TaskUpdater struct {
	Logger logr.Logger
	Client client.Client
	CRCli  history.Interface
}

func NewTaskUpdater(logger logr.Logger, c client.Client) task.Task[ReconcileContext] {
	return &TaskUpdater{
		Logger: logger,
		Client: c,
		CRCli:  history.NewClient(c),
	}
}

func (*TaskUpdater) Name() string {
	return "Updater"
}

func (t *TaskUpdater) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	// TODO: move to task v2
	if !rtx.TiDBGroup.GetDeletionTimestamp().IsZero() {
		return task.Complete().With("tidb group has been deleted")
	}

	if rtx.Cluster.ShouldSuspendCompute() {
		return task.Complete().With("skip updating TiDBGroup for suspension")
	}

	// List all controller revisions for the TiDBGroup
	selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			v1alpha1.LabelKeyCluster:   rtx.Cluster.Name,
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
			v1alpha1.LabelKeyGroup:     rtx.TiDBGroup.Name,
		},
	})
	revisions, err := t.CRCli.ListControllerRevisions(rtx.TiDBGroup, selector)
	if err != nil {
		return task.Fail().With("cannot list controller revisions: %w", err)
	}
	history.SortControllerRevisions(revisions)

	// Get the current(old) and update(new) ControllerRevisions for TiDBGroup
	currentRevision, updateRevision, collisionCount, err := revision.GetCurrentAndUpdate(rtx.TiDBGroup, revisions, t.CRCli, rtx.TiDBGroup)
	if err != nil {
		return task.Fail().With("cannot get revisions: %w", err)
	}
	rtx.CurrentRevision = currentRevision.Name
	rtx.UpdateRevision = updateRevision.Name
	rtx.CollisionCount = &collisionCount

	if err = revision.TruncateHistory(t.CRCli, rtx.TiDBs, revisions,
		currentRevision, updateRevision, rtx.Cluster.Spec.RevisionHistoryLimit); err != nil {
		t.Logger.Error(err, "failed to truncate history")
	}

	if needVersionUpgrade(rtx.TiDBGroup) && !rtx.UpgradeChecker.CanUpgrade(ctx, rtx.TiDBGroup) {
		return task.Fail().Continue().With(
			"preconditions of upgrading the tidb group %s/%s are not met",
			rtx.TiDBGroup.Namespace, rtx.TiDBGroup.Name)
	}

	desired := 1
	if rtx.TiDBGroup.Spec.Replicas != nil {
		desired = int(*rtx.TiDBGroup.Spec.Replicas)
	}

	var topos []v1alpha1.ScheduleTopology
	for _, p := range rtx.TiDBGroup.Spec.SchedulePolicies {
		switch p.Type {
		case v1alpha1.SchedulePolicyTypeEvenlySpread:
			topos = p.EvenlySpread.Topologies
		default:
			// do nothing
		}
	}

	topoPolicy, err := policy.NewTopologyPolicy[*runtime.TiDB](topos)
	if err != nil {
		return task.Fail().With("invalid topo policy, it should be validated: %w", err)
	}

	for _, tidb := range rtx.TiDBs {
		topoPolicy.Add(runtime.FromTiDB(tidb))
	}

	wait, err := updater.New[*runtime.TiDB]().
		WithInstances(runtime.FromTiDBSlice(rtx.TiDBs)...).
		WithDesired(desired).
		WithClient(t.Client).
		WithMaxSurge(0).
		WithMaxUnavailable(1).
		WithRevision(rtx.UpdateRevision).
		WithNewFactory(TiDBNewer(rtx.TiDBGroup, rtx.UpdateRevision)).
		WithAddHooks(topoPolicy).
		WithUpdateHooks(
			policy.KeepName[*runtime.TiDB](),
			policy.KeepTopology[*runtime.TiDB](),
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
		return task.Complete().With("wait for all instances ready")
	}
	return task.Complete().With("all instances are synced")
}

func needVersionUpgrade(dbg *v1alpha1.TiDBGroup) bool {
	return dbg.Spec.Version != dbg.Status.Version && dbg.Status.Version != ""
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
				Version:          dbg.Spec.Version,
				Subdomain:        HeadlessServiceName(dbg.Name), // same as headless service
				TiDBTemplateSpec: *spec,
			},
		}

		return runtime.FromTiDB(tidb)
	})
}
