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
	"strconv"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/action"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/updater"
	"github.com/pingcap/tidb-operator/v2/pkg/updater/policy"
	maputil "github.com/pingcap/tidb-operator/v2/pkg/utils/map"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

const (
	defaultUpdateWaitTime = time.Second * 30
)

func TaskUpdater(state *ReconcileContext, c client.Client, af tracker.AllocateFactory) task.Task {
	return task.NameTaskFunc("Updater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		dmg := state.DMGroup()

		checker := action.NewUpgradeChecker[scope.DMGroup](c, state.Cluster(), logger)

		if needVersionUpgrade(dmg) && !checker.CanUpgrade(ctx, dmg) {
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
		}

		retryAfter := coreutil.RetryIfInstancesReadyButNotAvailable[scope.DM](state.InstanceSlice(), coreutil.MinReadySeconds[scope.DMGroup](dmg))
		if retryAfter != 0 {
			return task.Retry(retryAfter).With("wait until no instances is ready but not available")
		}

		var topos []v1alpha1.ScheduleTopology
		for _, p := range dmg.Spec.SchedulePolicies {
			switch p.Type {
			case v1alpha1.SchedulePolicyTypeEvenlySpread:
				topos = p.EvenlySpread.Topologies
			default:
			}
		}

		updateRevision, _, _ := state.Revision()

		dms := state.Slice()
		topoPolicy, err := policy.NewTopologyPolicy(topos, updateRevision, dms...)
		if err != nil {
			return task.Fail().With("invalid topo policy, it should be validated: %w", err)
		}

		var instances []string
		for _, in := range dms {
			instances = append(instances, in.Name)
		}

		allocator := af.New(dmg.Namespace, dmg.Name, instances...)

		wait, err := updater.New[runtime.DMTuple]().
			WithInstances(dms...).
			WithDesired(int(state.Group().Replicas())).
			WithClient(c).
			WithMaxSurge(0).
			WithMaxUnavailable(1).
			WithRevision(updateRevision).
			WithNewFactory(DMNewer(dmg, updateRevision, state.InstanceSlice())).
			WithAddHooks(
				updater.AllocateName[*runtime.DM](allocator),
				topoPolicy,
			).
			WithDelHooks(topoPolicy).
			WithUpdateHooks(topoPolicy).
			WithScaleInPreferPolicy(topoPolicy.PolicyScaleIn()).
			WithMinReadySeconds(coreutil.MinReadySeconds[scope.DMGroup](dmg)).
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

func needVersionUpgrade(dmg *v1alpha1.DMGroup) bool {
	return dmg.Spec.Template.Spec.Version != dmg.Status.Version && dmg.Status.Version != ""
}

func DMNewer(dmg *v1alpha1.DMGroup, rev string, currentDMs []*v1alpha1.DM) updater.NewFactory[*runtime.DM] {
	// Derive bootstrap state: if any existing instance lacks the boot annotation,
	// the cluster has already bootstrapped and new instances should use join.
	alreadyBootstrapped := false
	for _, dm := range currentDMs {
		if _, ok := dm.Annotations[v1alpha1.AnnoKeyInitialClusterNum]; !ok {
			alreadyBootstrapped = true
			break
		}
	}

	return updater.NewFunc[*runtime.DM](func() *runtime.DM {
		spec := dmg.Spec.Template.Spec.DeepCopy()

		var bootAnno map[string]string
		if !alreadyBootstrapped {
			replicas := int64(1)
			if dmg.Spec.Replicas != nil {
				replicas = int64(*dmg.Spec.Replicas)
			}
			initialNum := strconv.FormatInt(replicas, 10)
			bootAnno = map[string]string{
				v1alpha1.AnnoKeyInitialClusterNum: initialNum,
			}
		}

		dm := &v1alpha1.DM{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: dmg.Namespace,
				// Name will be allocated by updater.AllocateName
				Labels:      coreutil.InstanceLabels[scope.DMGroup](dmg, rev),
				Annotations: coreutil.InstanceAnnotations[scope.DMGroup](dmg),
				Finalizers:  []string{metav1alpha1.Finalizer},
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(dmg, v1alpha1.SchemeGroupVersion.WithKind("DMGroup")),
				},
			},
			Spec: v1alpha1.DMSpec{
				Cluster:        dmg.Spec.Cluster,
				Features:       dmg.Spec.Features,
				Subdomain:      coreutil.HeadlessServiceName[scope.DMGroup](dmg),
				DMTemplateSpec: *spec,
			},
		}

		dm.Annotations = maputil.MergeTo(dm.Annotations, bootAnno)

		return runtime.FromDM(dm)
	})
}
