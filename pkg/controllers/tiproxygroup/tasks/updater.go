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
	"github.com/pingcap/tidb-operator/v2/pkg/action"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/reloadable"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/updater"
	"github.com/pingcap/tidb-operator/v2/pkg/updater/policy"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

const (
	defaultUpdateWaitTime = time.Second * 30
)

// TaskUpdater is a task to scale or update TiProxy when spec of TiProxyGroup is changed.
func TaskUpdater(state *ReconcileContext, c client.Client, af tracker.AllocateFactory) task.Task {
	return task.NameTaskFunc("Updater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		proxyg := state.TiProxyGroup()

		checker := action.NewUpgradeChecker[scope.TiProxyGroup](c, state.Cluster(), logger)

		if needVersionUpgrade(proxyg) && !checker.CanUpgrade(ctx, proxyg) {
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
		}

		retryAfter := coreutil.RetryIfInstancesReadyButNotAvailable[scope.TiProxy](
			state.InstanceSlice(),
			coreutil.MinReadySeconds[scope.TiProxyGroup](proxyg),
		)
		if retryAfter != 0 {
			return task.Retry(retryAfter).With("wait until no instances is ready but not available")
		}

		var topos []v1alpha1.ScheduleTopology
		for _, p := range proxyg.Spec.SchedulePolicies {
			//nolint:gocritic // may add more types in the future
			switch p.Type {
			case v1alpha1.SchedulePolicyTypeEvenlySpread:
				topos = p.EvenlySpread.Topologies
			}
		}

		updateRevision, _, _ := state.Revision()

		proxies := state.Slice()
		topoPolicy, err := policy.NewTopologyPolicy(topos, updateRevision, proxies...)
		if err != nil {
			return task.Fail().With("invalid topo policy, it should be validated: %w", err)
		}

		needUpdate, needRestart := precheckInstances(proxyg, runtime.ToTiProxySlice(proxies), updateRevision)
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
		for _, in := range proxies {
			instances = append(instances, in.Name)
		}

		allocator := af.New(proxyg.Namespace, proxyg.Name, instances...)

		wait, err := updater.New[runtime.TiProxyTuple]().
			WithInstances(proxies...).
			WithDesired(int(state.Group().Replicas())).
			WithClient(c).
			WithMaxSurge(maxSurge).
			WithMaxUnavailable(maxUnavailable).
			WithRevision(updateRevision).
			WithNewFactory(TiProxyNewer(proxyg, updateRevision, state.FeatureGates())).
			WithAddHooks(
				updater.AllocateName[*runtime.TiProxy](allocator),
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
			WithMinReadySeconds(coreutil.MinReadySeconds[scope.TiProxyGroup](proxyg)).
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

func needVersionUpgrade(proxyg *v1alpha1.TiProxyGroup) bool {
	return proxyg.Spec.Template.Spec.Version != proxyg.Status.Version && proxyg.Status.Version != ""
}

func precheckInstances(proxyg *v1alpha1.TiProxyGroup, proxies []*v1alpha1.TiProxy, updateRevision string) (needUpdate, needRestart bool) {
	if len(proxies) != int(coreutil.Replicas[scope.TiProxyGroup](proxyg)) {
		needUpdate = true
	}
	for _, proxy := range proxies {
		if coreutil.UpdateRevision[scope.TiProxy](proxy) == updateRevision {
			continue
		}

		needUpdate = true
		if !reloadable.CheckTiProxy(proxyg, proxy) {
			needRestart = true
		}
	}

	return needUpdate, needRestart
}

func TiProxyNewer(proxyg *v1alpha1.TiProxyGroup, rev string, fg features.Gates) updater.NewFactory[*runtime.TiProxy] {
	return updater.NewFunc[*runtime.TiProxy](func() *runtime.TiProxy {
		spec := proxyg.Spec.Template.Spec.DeepCopy()

		proxy := &v1alpha1.TiProxy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: proxyg.Namespace,
				// Name will be allocated by updater.AllocateName
				Labels:      coreutil.InstanceLabels[scope.TiProxyGroup](proxyg, rev),
				Annotations: coreutil.InstanceAnnotations[scope.TiProxyGroup](proxyg),
				Finalizers:  []string{metav1alpha1.Finalizer},
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(proxyg, v1alpha1.SchemeGroupVersion.WithKind("TiProxyGroup")),
				},
			},
			Spec: v1alpha1.TiProxySpec{
				Cluster:             proxyg.Spec.Cluster,
				Features:            proxyg.Spec.Features,
				Subdomain:           coreutil.HeadlessServiceName[scope.TiProxyGroup](proxyg), // same as headless service
				TiProxyTemplateSpec: *spec,
			},
		}

		if fg.Enabled(metav1alpha1.ClusterSubdomain) {
			proxy.Spec.Subdomain = coreutil.ClusterSubdomain(proxyg.Spec.Cluster.Name)
		}

		return runtime.FromTiProxy(proxy)
	})
}
