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

// TaskUpdater is a task for updating TiFlashGroup when its spec is changed.
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
	if !rtx.TiFlashGroup.GetDeletionTimestamp().IsZero() {
		return task.Complete().With("tiflash group has been deleted")
	}

	// List all controller revisions for the TiFlashGroup.
	selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			v1alpha1.LabelKeyCluster:   rtx.Cluster.Name,
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
			v1alpha1.LabelKeyGroup:     rtx.TiFlashGroup.Name,
		},
	})
	revisions, err := t.CRCli.ListControllerRevisions(rtx.TiFlashGroup, selector)
	if err != nil {
		return task.Fail().With("cannot list controller revisions: %w", err)
	}
	history.SortControllerRevisions(revisions)

	// Get the current(old) and update(new) ControllerRevisions.
	currentRevision, updateRevision, collisionCount, err := revision.GetCurrentAndUpdate(
		rtx.TiFlashGroup, revisions, t.CRCli, rtx.TiFlashGroup)
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

	if needVersionUpgrade(rtx.TiFlashGroup) && !rtx.UpgradeChecker.CanUpgrade(ctx, rtx.TiFlashGroup) {
		return task.Fail().Continue().With(
			"preconditions of upgrading the tiflash group %s/%s are not met",
			rtx.TiFlashGroup.Namespace, rtx.TiFlashGroup.Name)
	}

	desired := 1
	if rtx.TiFlashGroup.Spec.Replicas != nil {
		desired = int(*rtx.TiFlashGroup.Spec.Replicas)
	}

	var topos []v1alpha1.ScheduleTopology
	for _, p := range rtx.TiFlashGroup.Spec.SchedulePolicies {
		switch p.Type {
		case v1alpha1.SchedulePolicyTypeEvenlySpread:
			topos = p.EvenlySpread.Topologies
		default:
			// do nothing
		}
	}

	topoPolicy, err := policy.NewTopologyPolicy[*runtime.TiFlash](topos)
	if err != nil {
		return task.Fail().With("invalid topo policy, it should be validated: %w", err)
	}

	for _, tiflash := range rtx.Peers {
		topoPolicy.Add(runtime.FromTiFlash(tiflash))
	}

	wait, err := updater.New[*runtime.TiFlash]().
		WithInstances(runtime.FromTiFlashSlice(rtx.Peers)...).
		WithDesired(desired).
		WithClient(t.Client).
		WithMaxSurge(0).
		WithMaxUnavailable(1).
		WithRevision(rtx.UpdateRevision).
		WithNewFactory(TiFlashNewer(rtx.TiFlashGroup, rtx.UpdateRevision)).
		WithAddHooks(topoPolicy).
		WithUpdateHooks(
			policy.KeepName[*runtime.TiFlash](),
			policy.KeepTopology[*runtime.TiFlash](),
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

func needVersionUpgrade(flashg *v1alpha1.TiFlashGroup) bool {
	return flashg.Spec.Version != flashg.Status.Version && flashg.Status.Version != ""
}

const (
	suffixLen = 6
)

func TiFlashNewer(fg *v1alpha1.TiFlashGroup, rev string) updater.NewFactory[*runtime.TiFlash] {
	return updater.NewFunc[*runtime.TiFlash](func() *runtime.TiFlash {
		name := fmt.Sprintf("%s-%s", fg.Name, random.Random(suffixLen))
		spec := fg.Spec.Template.Spec.DeepCopy()

		tiflash := &v1alpha1.TiFlash{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: fg.Namespace,
				Name:      name,
				Labels: maputil.Merge(fg.Spec.Template.Labels, map[string]string{
					v1alpha1.LabelKeyManagedBy:            v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyComponent:            v1alpha1.LabelValComponentTiFlash,
					v1alpha1.LabelKeyCluster:              fg.Spec.Cluster.Name,
					v1alpha1.LabelKeyGroup:                fg.Name,
					v1alpha1.LabelKeyInstanceRevisionHash: rev,
				}),
				Annotations: maputil.Copy(fg.Spec.Template.Annotations),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(fg, v1alpha1.SchemeGroupVersion.WithKind("TiFlashGroup")),
				},
			},
			Spec: v1alpha1.TiFlashSpec{
				Cluster:             fg.Spec.Cluster,
				Version:             fg.Spec.Version,
				Subdomain:           HeadlessServiceName(fg.Name),
				TiFlashTemplateSpec: *spec,
			},
		}

		return runtime.FromTiFlash(tiflash)
	})
}
