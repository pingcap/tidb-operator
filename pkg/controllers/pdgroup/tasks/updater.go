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
	"strconv"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
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

// TaskUpdater is a task to scale or update PD when spec of PDGroup is changed.
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
	if !rtx.PDGroup.GetDeletionTimestamp().IsZero() {
		return task.Complete().With("pd group has been deleted")
	}

	if rtx.Cluster.ShouldSuspendCompute() {
		return task.Complete().With("skip updating PDGroup for suspension")
	}

	// List all controller revisions for the PDGroup
	selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			v1alpha1.LabelKeyCluster:   rtx.Cluster.Name,
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
			v1alpha1.LabelKeyGroup:     rtx.PDGroup.Name,
		},
	})
	revisions, err := t.CRCli.ListControllerRevisions(rtx.PDGroup, selector)
	if err != nil {
		return task.Fail().With("cannot list controller revisions: %w", err)
	}
	history.SortControllerRevisions(revisions)

	// Get the current(old) and update(new) ControllerRevisions
	currentRevision, updateRevision, collisionCount, err := func() (*appsv1.ControllerRevision, *appsv1.ControllerRevision, int32, error) {
		// always ignore bootstrapped field in spec
		bootstrapped := rtx.PDGroup.Spec.Bootstrapped
		rtx.PDGroup.Spec.Bootstrapped = false
		defer func() {
			rtx.PDGroup.Spec.Bootstrapped = bootstrapped
		}()
		return revision.GetCurrentAndUpdate(rtx.PDGroup, revisions, t.CRCli, rtx.PDGroup)
	}()
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

	if needVersionUpgrade(rtx.PDGroup) && !rtx.UpgradeChecker.CanUpgrade(ctx, rtx.PDGroup) {
		return task.Fail().Continue().With("preconditions of upgrading the pd group %s/%s are not met", rtx.PDGroup.Namespace, rtx.PDGroup.Name)
	}

	desired := 1
	if rtx.PDGroup.Spec.Replicas != nil {
		desired = int(*rtx.PDGroup.Spec.Replicas)
	}

	var topos []v1alpha1.ScheduleTopology
	for _, p := range rtx.PDGroup.Spec.SchedulePolicies {
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

	for _, pd := range rtx.Peers {
		topoPolicy.Add(runtime.FromPD(pd))
	}

	wait, err := updater.New[*runtime.PD]().
		WithInstances(runtime.FromPDSlice(rtx.Peers)...).
		WithDesired(desired).
		WithClient(t.Client).
		WithMaxSurge(0).
		WithMaxUnavailable(1).
		WithRevision(rtx.UpdateRevision).
		WithNewFactory(PDNewer(rtx.PDGroup, rtx.UpdateRevision)).
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
		return task.Complete().With("wait for all instances ready")
	}
	return task.Complete().With("all instances are synced")
}

func needVersionUpgrade(pdg *v1alpha1.PDGroup) bool {
	return pdg.Spec.Version != pdg.Status.Version && pdg.Status.Version != ""
}

func PDNewer(pdg *v1alpha1.PDGroup, rev string) updater.NewFactory[*runtime.PD] {
	return updater.NewFunc[*runtime.PD](func() *runtime.PD {
		//nolint:mnd // refactor to use a constant
		name := fmt.Sprintf("%s-%s-%s", pdg.Spec.Cluster.Name, pdg.Name, random.Random(6))
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
				Subdomain:      HeadlessServiceName(pdg.Spec.Cluster.Name, pdg.Name),
				PDTemplateSpec: *spec,
			},
		}

		return runtime.FromPD(peer)
	})
}
