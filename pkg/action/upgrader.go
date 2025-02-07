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
	"github.com/pingcap/tidb-operator/pkg/client"
)

type UpgradePolicy interface {
	ArePreconditionsMet(ctx context.Context, cli client.Client, group v1alpha1.Group) (bool, error)
}

var (
	_ UpgradePolicy = &defaultPolicy{}
	_ UpgradePolicy = &noConstraint{}
)

type defaultPolicy struct{}

func (defaultPolicy) ArePreconditionsMet(ctx context.Context, cli client.Client, group v1alpha1.Group) (bool, error) {
	groups, err := getDependentGroups(ctx, cli, group)
	if err != nil {
		return false, fmt.Errorf("cannot get dependent groups for %s/%s: %w", group.GetNamespace(), group.GetName(), err)
	}
	return areGroupsUpgraded(group.GetDesiredVersion(), groups)
}

// getDependentGroups returns the groups that depend on the given group when upgrade.
func getDependentGroups(ctx context.Context, cli client.Client, group v1alpha1.Group) (groups []v1alpha1.Group, err error) {
	switch group.ComponentKind() {
	case v1alpha1.ComponentKindPD:

	case v1alpha1.ComponentKindTiDB:
		var kvgList v1alpha1.TiKVGroupList
		if err = cli.List(ctx, &kvgList, client.InNamespace(group.GetNamespace()), client.MatchingLabels{
			v1alpha1.LabelKeyCluster:   group.GetClusterName(),
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
		}); err != nil {
			return nil, fmt.Errorf("cannot list TiKVGroups: %w", err)
		}
		groups = kvgList.ToSlice()

	case v1alpha1.ComponentKindTiKV:
		var tiflashGroupList v1alpha1.TiFlashGroupList
		if err = cli.List(ctx, &tiflashGroupList, client.InNamespace(group.GetNamespace()), client.MatchingLabels{
			v1alpha1.LabelKeyCluster:   group.GetClusterName(),
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
		}); err != nil {
			return nil, fmt.Errorf("cannot list TiFlashGroups: %w", err)
		}
		groups = tiflashGroupList.ToSlice()
		// If there is no TiFlashGroup, should check PDGroups.
		if len(groups) == 0 {
			groups, err = listPDGroups(ctx, cli, group.GetNamespace(), group.GetClusterName())
			if err != nil {
				return nil, fmt.Errorf("cannot list PDGroups: %w", err)
			}
		}

	case v1alpha1.ComponentKindTiFlash:
		groups, err = listPDGroups(ctx, cli, group.GetNamespace(), group.GetClusterName())
		if err != nil {
			return nil, fmt.Errorf("cannot list PDGroups: %w", err)
		}
	}

	return groups, nil
}

func listPDGroups(ctx context.Context, cli client.Client, ns, clusterName string) ([]v1alpha1.Group, error) {
	var pdgList v1alpha1.PDGroupList
	if err := cli.List(ctx, &pdgList, client.InNamespace(ns), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   clusterName,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
	}); err != nil {
		return nil, fmt.Errorf("cannot list PDGroups: %w", err)
	}
	return pdgList.ToSlice(), nil
}

// areGroupsUpgraded checks if all groups' version are greater or equal to the desired version.
func areGroupsUpgraded(version string, groups []v1alpha1.Group) (bool, error) {
	desiredVer, err := semver.NewVersion(version)
	if err != nil {
		return false, fmt.Errorf("cannot parse the desired version: %s", version)
	}

	for _, group := range groups {
		v, e := semver.NewVersion(group.GetActualVersion())
		if e != nil {
			return false, fmt.Errorf("cannot parse the group status version: %s", group.GetActualVersion())
		}
		if !v.GreaterThanEqual(desiredVer) || !v1alpha1.IsGroupHealthyAndUpToDate(group) {
			return false, nil
		}
	}
	return true, nil
}

type noConstraint struct{}

func (noConstraint) ArePreconditionsMet(_ context.Context, _ client.Client, _ v1alpha1.Group) (bool, error) {
	return true, nil
}

type UpgradeChecker interface {
	CanUpgrade(context.Context, v1alpha1.Group) bool
}

type upgradeChecker struct {
	cli    client.Client
	logger logr.Logger
	policy UpgradePolicy
}

func NewUpgradeChecker(cli client.Client, cluster *v1alpha1.Cluster, logger logr.Logger) UpgradeChecker {
	var policy UpgradePolicy
	switch cluster.Spec.UpgradePolicy {
	case v1alpha1.UpgradePolicyNoConstraints:
		policy = &noConstraint{}
	case v1alpha1.UpgradePolicyDefault:
		policy = &defaultPolicy{}
	default:
		logger.Info("unknown upgrade policy, use the default one", "policy", cluster.Spec.UpgradePolicy)
		policy = &defaultPolicy{}
	}
	return &upgradeChecker{cli: cli, logger: logger, policy: policy}
}

func (c *upgradeChecker) CanUpgrade(ctx context.Context, group v1alpha1.Group) bool {
	yes, err := c.policy.ArePreconditionsMet(ctx, c.cli, group)
	if err != nil {
		c.logger.Error(err, "failed to check preconditions for upgrading", "group_ns",
			group.GetNamespace(), "group_name", group.GetName(), "component", group.ComponentKind())
	}
	return yes
}
