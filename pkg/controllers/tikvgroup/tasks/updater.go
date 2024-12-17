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
	"k8s.io/apimachinery/pkg/labels"

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

// TaskUpdater is a task for updating TikVGroup when its spec is changed.
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
	if !rtx.TiKVGroup.GetDeletionTimestamp().IsZero() {
		return task.Complete().With("tikv group has been deleted")
	}

	// List all controller revisions for the TiKVGroup.
	selector := labels.SelectorFromSet(labels.Set{
		v1alpha1.LabelKeyCluster:   rtx.Cluster.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
		v1alpha1.LabelKeyGroup:     rtx.TiKVGroup.Name,
	})
	revisions, err := t.CRCli.ListControllerRevisions(rtx.TiKVGroup, selector)
	if err != nil {
		return task.Fail().With("cannot list controller revisions: %w", err)
	}
	history.SortControllerRevisions(revisions)

	// Get the current(old) and update(new) ControllerRevisions.
	currentRevision, updateRevision, collisionCount, err := revision.GetCurrentAndUpdate(rtx.TiKVGroup, revisions, t.CRCli, rtx.TiKVGroup)
	if err != nil {
		return task.Fail().With("cannot get revisions: %w", err)
	}
	rtx.CurrentRevision = currentRevision.Name
	rtx.UpdateRevision = updateRevision.Name
	rtx.CollisionCount = &collisionCount

	if err = revision.TruncateHistory(t.CRCli, rtx.Peers, revisions,
		currentRevision, updateRevision, rtx.Cluster.Spec.RevisionHistoryLimit); err != nil {
		t.Logger.Error(err, "failed to truncate history")
	}

	if needVersionUpgrade(rtx.TiKVGroup) && !rtx.UpgradeChecker.CanUpgrade(ctx, rtx.TiKVGroup) {
		return task.Fail().Continue().With(
			"preconditions of upgrading the tikv group %s/%s are not met",
			rtx.TiKVGroup.Namespace, rtx.TiKVGroup.Name)
	}

	desired := 1
	if rtx.TiKVGroup.Spec.Replicas != nil {
		desired = int(*rtx.TiKVGroup.Spec.Replicas)
	}

	var topos []v1alpha1.ScheduleTopology
	for _, p := range rtx.TiKVGroup.Spec.SchedulePolicies {
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

	for _, tikv := range rtx.Peers {
		topoPolicy.Add(runtime.FromTiKV(tikv))
	}

	wait, err := updater.New[*runtime.TiKV]().
		WithInstances(runtime.FromTiKVSlice(rtx.Peers)...).
		WithDesired(desired).
		WithClient(t.Client).
		WithMaxSurge(0).
		WithMaxUnavailable(1).
		WithRevision(rtx.UpdateRevision).
		WithNewFactory(TiKVNewer(rtx.TiKVGroup, rtx.UpdateRevision)).
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
		return task.Complete().With("wait for all instances ready")
	}
	return task.Complete().With("all instances are synced")
}

func needVersionUpgrade(kvg *v1alpha1.TiKVGroup) bool {
	return kvg.Spec.Version != kvg.Status.Version && kvg.Status.Version != ""
}

func TiKVNewer(kvg *v1alpha1.TiKVGroup, rev string) updater.NewFactory[*runtime.TiKV] {
	return updater.NewFunc[*runtime.TiKV](func() *runtime.TiKV {
		//nolint:mnd // refactor to use a constant
		name := fmt.Sprintf("%s-%s-%s", kvg.Spec.Cluster.Name, kvg.Name, random.Random(6))

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
				Subdomain:        HeadlessServiceName(kvg.Spec.Cluster.Name, kvg.Name),
				TiKVTemplateSpec: *spec,
			},
		}

		return runtime.FromTiKV(tikv)
	})
}
