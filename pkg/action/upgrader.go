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

package action

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

type UpgradePolicy[
	G client.Object,
] interface {
	ArePreconditionsMet(ctx context.Context, cli client.Client, group G) (bool, error)
}

type noConstraints[G client.Object] struct{}

func (noConstraints[G]) ArePreconditionsMet(_ context.Context, _ client.Client, _ G) (bool, error) {
	return true, nil
}

func NewNoConstraints[G client.Object]() UpgradePolicy[G] {
	return noConstraints[G]{}
}

type defaultPolicy[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
] struct{}

func NewDefault[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
]() UpgradePolicy[F] {
	return defaultPolicy[S, F, T]{}
}

func (defaultPolicy[S, F, T]) ArePreconditionsMet(ctx context.Context, cli client.Client, group F) (bool, error) {
	ns := group.GetNamespace()
	cluster := coreutil.Cluster[S](group)
	version := coreutil.Version[S](group)

	var comps []string
	switch scope.Component[S]() {
	case v1alpha1.LabelValComponentPD:
	case v1alpha1.LabelValComponentTSO, v1alpha1.LabelValComponentScheduler:
		comps = append(comps,
			v1alpha1.LabelValComponentPD,
		)
	case v1alpha1.LabelValComponentTiKV:
		comps = append(comps,
			v1alpha1.LabelValComponentTiFlash,
			v1alpha1.LabelValComponentPD,
			v1alpha1.LabelValComponentTSO,
			v1alpha1.LabelValComponentScheduler,
		)
	case v1alpha1.LabelValComponentTiDB:
		comps = append(comps,
			v1alpha1.LabelValComponentTiKV,
			v1alpha1.LabelValComponentTiFlash,
			v1alpha1.LabelValComponentPD,
			v1alpha1.LabelValComponentTSO,
			v1alpha1.LabelValComponentScheduler,
		)
	case v1alpha1.LabelValComponentTiFlash:
		comps = append(comps,
			v1alpha1.LabelValComponentPD,
			v1alpha1.LabelValComponentTSO,
			v1alpha1.LabelValComponentScheduler,
		)
	default:
		return false, fmt.Errorf("unknown component: %s", scope.Component[S]())
	}

	return checkComponentsUpgraded(ctx, cli, ns, cluster, version, comps...)
}

func checkComponentsUpgraded(ctx context.Context, c client.Client, ns, cluster, version string, comps ...string) (bool, error) {
	for _, comp := range comps {
		var upgraded bool
		var err error
		switch comp {
		case v1alpha1.LabelValComponentPD:
			upgraded, err = checkOneComponentUpgraded[scope.PDGroup](ctx, c, ns, cluster, version)
		case v1alpha1.LabelValComponentTSO:
			upgraded, err = checkOneComponentUpgraded[scope.TSOGroup](ctx, c, ns, cluster, version)
		case v1alpha1.LabelValComponentScheduler:
			upgraded, err = checkOneComponentUpgraded[scope.SchedulerGroup](ctx, c, ns, cluster, version)
		case v1alpha1.LabelValComponentTiKV:
			upgraded, err = checkOneComponentUpgraded[scope.TiKVGroup](ctx, c, ns, cluster, version)
		case v1alpha1.LabelValComponentTiDB:
			upgraded, err = checkOneComponentUpgraded[scope.TiDBGroup](ctx, c, ns, cluster, version)
		case v1alpha1.LabelValComponentTiFlash:
			upgraded, err = checkOneComponentUpgraded[scope.TiFlashGroup](ctx, c, ns, cluster, version)
		default:
			return false, fmt.Errorf("unknown component: %s", comp)
		}
		if err != nil {
			return false, err
		}
		if !upgraded {
			return false, nil
		}
	}

	return true, nil
}

func checkOneComponentUpgraded[
	S scope.GroupList[F, T, L],
	F client.Object,
	T runtime.Group,
	L client.ObjectList,
](ctx context.Context, c client.Client, ns, cluster, version string) (bool, error) {
	comp := scope.Component[S]()
	groups, err := apicall.ListGroups[S](ctx, c, ns, cluster)
	if err != nil {
		return false, fmt.Errorf("cannot list %s groups: %w", comp, err)
	}

	upgraded, err := isUpgraded[S](groups, version)
	if err != nil {
		return false, fmt.Errorf("cannot check whether %s groups are upgraded: %w", comp, err)
	}

	return upgraded, nil
}

func isUpgraded[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](groups []F, version string) (bool, error) {
	// fast path, if no groups, no need to parse version
	if len(groups) == 0 {
		return true, nil
	}

	ver, err := semver.NewVersion(version)
	if err != nil {
		return false, fmt.Errorf("cannot parse the desired version: %s", version)
	}

	for _, group := range groups {
		statusVer := coreutil.StatusVersion[S](group)
		v, e := semver.NewVersion(statusVer)
		if e != nil {
			return false, fmt.Errorf("cannot parse the group status version: %s", statusVer)
		}
		if !v.GreaterThanEqual(ver) || !coreutil.IsGroupHealthyAndUpToDate[S](group) {
			return false, nil
		}
	}
	return true, nil
}

type UpgradeChecker[G client.Object] interface {
	CanUpgrade(context.Context, G) bool
}

type upgradeChecker[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
] struct {
	cli    client.Client
	logger logr.Logger
	policy UpgradePolicy[F]
}

func NewUpgradeChecker[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](cli client.Client, cluster *v1alpha1.Cluster, logger logr.Logger) UpgradeChecker[F] {
	var p UpgradePolicy[F]
	switch cluster.Spec.UpgradePolicy {
	case v1alpha1.UpgradePolicyNoConstraints:
		p = NewNoConstraints[F]()
	case v1alpha1.UpgradePolicyDefault:
		p = NewDefault[S]()
	default:
		logger.Info("unknown upgrade policy, use the default one", "policy", cluster.Spec.UpgradePolicy)
		p = NewDefault[S]()
	}
	return &upgradeChecker[S, F, T]{cli: cli, logger: logger, policy: p}
}

func (c *upgradeChecker[S, F, T]) CanUpgrade(ctx context.Context, group F) bool {
	yes, err := c.policy.ArePreconditionsMet(ctx, c.cli, group)
	if err != nil {
		c.logger.Error(err, "failed to check preconditions for upgrading", "group_ns",
			group.GetNamespace(), "group_name", group.GetName(), "component", scope.Component[S]())
	}
	c.logger.Info("check preconditions for upgrading", "group_ns",
		group.GetNamespace(), "group_name", group.GetName(), "component", scope.Component[S](), "result", yes)
	return yes
}
