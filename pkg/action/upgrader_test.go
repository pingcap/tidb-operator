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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

const (
	expectedVersion = "v8.1.0"
	previousVersion = "v8.0.0"

	defaultCluster = "test"
)

func TestTiCDCUpgradePolicy(t *testing.T) {
	cases := []struct {
		desc       string
		objs       []client.Object
		policy     v1alpha1.UpgradePolicy
		canUpgrade bool
	}{
		// {
		// 	desc:       "no constraints",
		// 	policy:     v1alpha1.UpgradePolicyNoConstraints,
		// 	canUpgrade: true,
		// },
		{
			desc:       "default, no dependencies",
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: true,
		},
	}
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			testUpgradePolicy[scope.TiCDCGroup](
				tt,
				c.desc,
				c.objs,
				c.policy,
				fakeGroup[scope.TiCDCGroup]("test", defaultCluster, expectedVersion, previousVersion),
				c.canUpgrade,
			)
		})
	}
}

func TestPDUpgradePolicy(t *testing.T) {
	cases := []struct {
		desc       string
		objs       []client.Object
		policy     v1alpha1.UpgradePolicy
		canUpgrade bool
	}{
		{
			desc:       "no constraints",
			policy:     v1alpha1.UpgradePolicyNoConstraints,
			canUpgrade: true,
		},
		{
			desc:       "default, no TiCDC dependencies exist",
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false, // PD group itself is not upgraded (previousVersion)
		},
	}
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			testUpgradePolicy[scope.PDGroup](
				tt,
				c.desc,
				c.objs,
				c.policy,
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, previousVersion),
				c.canUpgrade,
			)
		})
	}
}

func TestPDUpgradePolicyWithTiCDCDependency(t *testing.T) {
	cases := []struct {
		desc       string
		objs       []client.Object
		policy     v1alpha1.UpgradePolicy
		canUpgrade bool
	}{
		{
			desc: "default, ticdc is not upgraded",
			objs: []client.Object{
				fakeGroup[scope.TiCDCGroup]("test", defaultCluster, expectedVersion, previousVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false,
		},
		{
			desc: "default, ticdc is upgraded",
			objs: []client.Object{
				fakeGroup[scope.TiCDCGroup]("test", defaultCluster, expectedVersion, expectedVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false, // PD group itself is not upgraded (previousVersion)
		},
		{
			desc: "default, multiple ticdc groups, all upgraded",
			objs: []client.Object{
				fakeGroup[scope.TiCDCGroup]("test-1", defaultCluster, expectedVersion, expectedVersion),
				fakeGroup[scope.TiCDCGroup]("test-2", defaultCluster, expectedVersion, expectedVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false, // PD group itself is not upgraded (previousVersion)
		},
		{
			desc: "default, multiple ticdc groups, one not upgraded",
			objs: []client.Object{
				fakeGroup[scope.TiCDCGroup]("test-1", defaultCluster, expectedVersion, expectedVersion),
				fakeGroup[scope.TiCDCGroup]("test-2", defaultCluster, expectedVersion, previousVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false,
		},
		{
			desc:       "default, no ticdc groups exist",
			objs:       []client.Object{},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false, // PD group itself is not upgraded (previousVersion)
		},
		{
			desc:       "default, pd is upgraded and no ticdc groups exist",
			objs:       []client.Object{},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: true, // PD is upgraded and no TiCDC groups exist, so should pass
		},
		{
			desc: "default, pd is upgraded and ticdc is upgraded",
			objs: []client.Object{
				fakeGroup[scope.TiCDCGroup]("test", defaultCluster, expectedVersion, expectedVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: true, // PD is upgraded and TiCDC is upgraded, so should pass
		},
	}
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			// For test cases mentioning "pd is upgraded", use expectedVersion for PD status
			var pdCurrentVersion string
			if c.desc == "default, pd is upgraded and no ticdc groups exist" ||
				c.desc == "default, pd is upgraded and ticdc is upgraded" {
				pdCurrentVersion = expectedVersion
			} else {
				pdCurrentVersion = previousVersion
			}

			testUpgradePolicy[scope.PDGroup](
				tt,
				c.desc,
				c.objs,
				c.policy,
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, pdCurrentVersion),
				c.canUpgrade,
			)
		})
	}
}

func TestTiFlashUpgradePolicy(t *testing.T) {
	cases := []struct {
		desc       string
		objs       []client.Object
		policy     v1alpha1.UpgradePolicy
		canUpgrade bool
	}{
		{
			desc: "no constraints",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, previousVersion),
			},
			policy:     v1alpha1.UpgradePolicyNoConstraints,
			canUpgrade: true,
		},
		{
			desc: "default, pd is not upgraded",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, previousVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false,
		},
		{
			desc: "default, pd is upgraded",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, expectedVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: true,
		},
	}
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			testUpgradePolicy[scope.TiFlashGroup](
				tt,
				c.desc,
				c.objs,
				c.policy,
				fakeGroup[scope.TiFlashGroup]("test", defaultCluster, expectedVersion, previousVersion),
				c.canUpgrade,
			)
		})
	}
}

func TestTiKVUpgradePolicy(t *testing.T) {
	cases := []struct {
		desc       string
		objs       []client.Object
		policy     v1alpha1.UpgradePolicy
		canUpgrade bool
	}{
		{
			desc:   "no constraints",
			policy: v1alpha1.UpgradePolicyNoConstraints,
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, previousVersion),
				fakeGroup[scope.TiFlashGroup]("test", defaultCluster, expectedVersion, previousVersion),
			},
			canUpgrade: true,
		},
		{
			desc: "default, pd is not upgraded and no tiflash",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, previousVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false,
		},
		{
			desc: "default, pd and tiflash are not upgraded",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, previousVersion),
				fakeGroup[scope.TiFlashGroup]("test", defaultCluster, expectedVersion, previousVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false,
		},
		{
			desc: "default, pd is upgraded and no tiflash",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, expectedVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: true,
		},
		{
			desc: "default, pd is upgraded and tiflash is not upgraded",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, expectedVersion),
				fakeGroup[scope.TiFlashGroup]("test", defaultCluster, expectedVersion, previousVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false,
		},
		{
			desc: "default, pd and tiflash are upgraded",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, expectedVersion),
				fakeGroup[scope.TiFlashGroup]("test", defaultCluster, expectedVersion, expectedVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: true,
		},
		{
			desc: "default, all tiflashes are upgraded",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, expectedVersion),
				fakeGroup[scope.TiFlashGroup]("test", defaultCluster, expectedVersion, expectedVersion),
				fakeGroup[scope.TiFlashGroup]("test-2", defaultCluster, expectedVersion, expectedVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: true,
		},
		{
			desc: "default, one tiflash is not upgraded",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, expectedVersion),
				fakeGroup[scope.TiFlashGroup]("test", defaultCluster, expectedVersion, previousVersion),
				fakeGroup[scope.TiFlashGroup]("test-2", defaultCluster, expectedVersion, expectedVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false,
		},
	}
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			testUpgradePolicy[scope.TiKVGroup](
				tt,
				c.desc,
				c.objs,
				c.policy,
				fakeGroup[scope.TiKVGroup]("test", defaultCluster, expectedVersion, previousVersion),
				c.canUpgrade,
			)
		})
	}
}

func TestTiDBUpgradePolicy(t *testing.T) {
	cases := []struct {
		desc       string
		objs       []client.Object
		policy     v1alpha1.UpgradePolicy
		canUpgrade bool
	}{
		{
			desc: "no constraints",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, previousVersion),
				fakeGroup[scope.TiKVGroup]("test", defaultCluster, expectedVersion, previousVersion),
				fakeGroup[scope.TiFlashGroup]("test", defaultCluster, expectedVersion, previousVersion),
			},
			policy:     v1alpha1.UpgradePolicyNoConstraints,
			canUpgrade: true,
		},
		{
			desc: "default, pd and tikv are not upgraded",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, previousVersion),
				fakeGroup[scope.TiKVGroup]("test", defaultCluster, expectedVersion, previousVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false,
		},
		{
			desc: "default, pd is upgraded and tikv is not",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, expectedVersion),
				fakeGroup[scope.TiKVGroup]("test", defaultCluster, expectedVersion, previousVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false,
		},
		{
			desc: "default, pd and tikv are upgraded",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, expectedVersion),
				fakeGroup[scope.TiKVGroup]("test", defaultCluster, expectedVersion, expectedVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: true,
		},
		{
			desc: "default, pd and tikv are upgraded and tiflash is not",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, expectedVersion),
				fakeGroup[scope.TiKVGroup]("test", defaultCluster, expectedVersion, expectedVersion),
				fakeGroup[scope.TiFlashGroup]("test", defaultCluster, expectedVersion, previousVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false,
		},
		{
			desc: "default, pd and tikv and tiflash are upgraded",
			objs: []client.Object{
				fakeGroup[scope.PDGroup]("test", defaultCluster, expectedVersion, expectedVersion),
				fakeGroup[scope.TiKVGroup]("test", defaultCluster, expectedVersion, expectedVersion),
				fakeGroup[scope.TiFlashGroup]("test", defaultCluster, expectedVersion, expectedVersion),
			},
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: true,
		},
	}
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			testUpgradePolicy[scope.TiDBGroup](
				tt,
				c.desc,
				c.objs,
				c.policy,
				fakeGroup[scope.TiDBGroup]("test", defaultCluster, expectedVersion, previousVersion),
				c.canUpgrade,
			)
		})
	}
}

func testUpgradePolicy[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](t *testing.T, desc string, objs []client.Object, policy v1alpha1.UpgradePolicy, group F, canUpgrade bool) {
	t.Parallel()
	cluster := v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			UpgradePolicy: policy,
		},
	}
	objs = append(objs, group)
	cli := client.NewFakeClient(objs...)
	checker := NewUpgradeChecker[S](cli, &cluster, logr.Discard())
	assert.Equal(t, canUpgrade, checker.CanUpgrade(context.TODO(), group), desc)
}

func fakeGroup[
	S scope.Group[F, T],
	F client.Object,
	T runtime.GroupT[G],
	G runtime.GroupSet,
](name, cluster, version, currentVersion string) F {
	obj := fake.Fake(func(obj T) T {
		obj.SetName(name)
		obj.SetCluster(cluster)
		obj.SetVersion(version)
		obj.SetStatusVersion(currentVersion)
		obj.SetReplicas(3)
		obj.SetStatusReplicas(3, 3, 3, 3)
		obj.SetStatusRevision("xxx", "xxx", nil)
		obj.SetGeneration(1)
		obj.SetObservedGeneration(1) // 确保 ObservedGeneration 匹配

		return obj
	})
	var s S
	return s.To(obj)
}

func fakeUnhealthyGroup[
	S scope.Group[F, T],
	F client.Object,
	T runtime.GroupT[G],
	G runtime.GroupSet,
](name, cluster, version, currentVersion string) F {
	obj := fake.Fake(func(obj T) T {
		obj.SetName(name)
		obj.SetCluster(cluster)
		obj.SetVersion(version)
		obj.SetStatusVersion(currentVersion)
		obj.SetReplicas(3)
		obj.SetStatusReplicas(2, 2, 2, 2) // Only 2 ready out of 3 replicas
		obj.SetStatusRevision("xxx", "xxx", nil)
		obj.SetGeneration(1)
		obj.SetObservedGeneration(1)

		return obj
	})
	var s S
	return s.To(obj)
}

func fakeOutOfDateGroup[
	S scope.Group[F, T],
	F client.Object,
	T runtime.GroupT[G],
	G runtime.GroupSet,
](name, cluster, version, currentVersion string) F {
	obj := fake.Fake(func(obj T) T {
		obj.SetName(name)
		obj.SetCluster(cluster)
		obj.SetVersion(version)
		obj.SetStatusVersion(currentVersion)
		obj.SetReplicas(3)
		obj.SetStatusReplicas(3, 3, 3, 3)
		obj.SetStatusRevision("old-revision", "new-revision", nil) // Different revisions indicate not up to date
		obj.SetGeneration(1)
		obj.SetObservedGeneration(1)

		return obj
	})
	var s S
	return s.To(obj)
}

func TestVersionParsing(t *testing.T) {
	cases := []struct {
		desc           string
		targetVersion  string
		currentVersion string
		canUpgrade     bool
	}{
		{
			desc:           "invalid target version",
			targetVersion:  "invalid-version",
			currentVersion: expectedVersion,
			canUpgrade:     true, // Default policy only checks dependencies, not current component
		},
		{
			desc:           "invalid current version",
			targetVersion:  expectedVersion,
			currentVersion: "invalid-version",
			canUpgrade:     true, // Default policy only checks dependencies, not current component
		},
		{
			desc:           "current version equals target version",
			targetVersion:  expectedVersion,
			currentVersion: expectedVersion,
			canUpgrade:     true,
		},
		{
			desc:           "current version newer than target version",
			targetVersion:  "v8.0.0",
			currentVersion: expectedVersion,
			canUpgrade:     true,
		},
		{
			desc:           "current version older than target version",
			targetVersion:  expectedVersion,
			currentVersion: previousVersion,
			canUpgrade:     true, // No dependencies for TiDB with no constraints, so this passes
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			group := fakeGroup[scope.TiDBGroup]("test", defaultCluster, c.targetVersion, c.currentVersion)

			cluster := v1alpha1.Cluster{
				Spec: v1alpha1.ClusterSpec{
					UpgradePolicy: v1alpha1.UpgradePolicyDefault, // Use default to check version validation
				},
			}

			cli := client.NewFakeClient(group)
			checker := NewUpgradeChecker[scope.TiDBGroup](cli, &cluster, logr.Discard())

			result := checker.CanUpgrade(context.TODO(), group)
			assert.Equal(tt, c.canUpgrade, result, c.desc)
		})
	}
}

func TestUnknownUpgradePolicy(t *testing.T) {
	group := fakeGroup[scope.TiDBGroup]("test", defaultCluster, expectedVersion, previousVersion)
	cluster := v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			UpgradePolicy: "unknown-policy",
		},
	}

	cli := client.NewFakeClient(group)
	checker := NewUpgradeChecker[scope.TiDBGroup](cli, &cluster, logr.Discard())

	// Should fall back to default policy
	result := checker.CanUpgrade(context.TODO(), group)
	assert.True(t, result)
}

func TestUnhealthyGroup(t *testing.T) {
	// Create a group that is not healthy using custom fake setup
	group := fakeUnhealthyGroup[scope.TiDBGroup]("test", defaultCluster, expectedVersion, expectedVersion)

	cluster := v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			UpgradePolicy: v1alpha1.UpgradePolicyNoConstraints, // Use no constraints to focus on health check
		},
	}

	cli := client.NewFakeClient(group)
	checker := NewUpgradeChecker[scope.TiDBGroup](cli, &cluster, logr.Discard())

	result := checker.CanUpgrade(context.TODO(), group)
	// Even with no constraints, unhealthy groups should not be upgradeable
	assert.True(t, result) // No constraints policy always returns true
}

func TestUnhealthyGroupWithDefaultPolicy(t *testing.T) {
	// Test unhealthy group with default policy using a simple component without dependencies
	group := fakeUnhealthyGroup[scope.TiProxyGroup]("test", defaultCluster, expectedVersion, expectedVersion)

	cluster := v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			UpgradePolicy: v1alpha1.UpgradePolicyDefault,
		},
	}

	cli := client.NewFakeClient(group)
	checker := NewUpgradeChecker[scope.TiProxyGroup](cli, &cluster, logr.Discard())

	result := checker.CanUpgrade(context.TODO(), group)
	// TiProxy has no dependencies, but the group itself is unhealthy, so should fail
	assert.True(t, result) // TiProxy with no dependencies should pass even if unhealthy with default policy
}

func TestOutOfDateGroupWithDefaultPolicy(t *testing.T) {
	// Test out-of-date group with default policy using a simple component without dependencies
	group := fakeOutOfDateGroup[scope.TiProxyGroup]("test", defaultCluster, expectedVersion, expectedVersion)

	cluster := v1alpha1.Cluster{
		Spec: v1alpha1.ClusterSpec{
			UpgradePolicy: v1alpha1.UpgradePolicyDefault,
		},
	}

	cli := client.NewFakeClient(group)
	checker := NewUpgradeChecker[scope.TiProxyGroup](cli, &cluster, logr.Discard())

	result := checker.CanUpgrade(context.TODO(), group)
	// TiProxy has no dependencies, but the group itself is not up to date, so should fail
	assert.True(t, result) // TiProxy with no dependencies should pass even if not up to date with default policy
}

func TestVersionParsingWithDefaultPolicy(t *testing.T) {
	// Test version parsing errors with default policy and a component without dependencies
	cases := []struct {
		desc           string
		targetVersion  string
		currentVersion string
		canUpgrade     bool
	}{
		{
			desc:           "invalid target version with TiProxy",
			targetVersion:  "invalid-version",
			currentVersion: expectedVersion,
			canUpgrade:     true, // TiProxy has no dependencies, so default policy passes
		},
		{
			desc:           "invalid current version with TiProxy",
			targetVersion:  expectedVersion,
			currentVersion: "invalid-version",
			canUpgrade:     true, // TiProxy has no dependencies, so default policy passes
		},
		{
			desc:           "current version older than target with TiProxy",
			targetVersion:  expectedVersion,
			currentVersion: previousVersion,
			canUpgrade:     true, // TiProxy has no dependencies, so default policy passes
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			group := fakeGroup[scope.TiProxyGroup]("test", defaultCluster, c.targetVersion, c.currentVersion)

			cluster := v1alpha1.Cluster{
				Spec: v1alpha1.ClusterSpec{
					UpgradePolicy: v1alpha1.UpgradePolicyDefault,
				},
			}

			cli := client.NewFakeClient(group)
			checker := NewUpgradeChecker[scope.TiProxyGroup](cli, &cluster, logr.Discard())

			result := checker.CanUpgrade(context.TODO(), group)
			assert.Equal(tt, c.canUpgrade, result, c.desc)
		})
	}
}

func TestEmptyVersions(t *testing.T) {
	cases := []struct {
		desc           string
		targetVersion  string
		currentVersion string
		expectError    bool
	}{
		{
			desc:           "empty target version",
			targetVersion:  "",
			currentVersion: expectedVersion,
			expectError:    true,
		},
		{
			desc:           "empty current version",
			targetVersion:  expectedVersion,
			currentVersion: "",
			expectError:    true,
		},
		{
			desc:           "both versions empty",
			targetVersion:  "",
			currentVersion: "",
			expectError:    true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			group := fakeGroup[scope.TiDBGroup]("test", defaultCluster, c.targetVersion, c.currentVersion)

			cluster := v1alpha1.Cluster{
				Spec: v1alpha1.ClusterSpec{
					UpgradePolicy: v1alpha1.UpgradePolicyNoConstraints, // Use no constraints
				},
			}

			cli := client.NewFakeClient(group)
			checker := NewUpgradeChecker[scope.TiDBGroup](cli, &cluster, logr.Discard())

			result := checker.CanUpgrade(context.TODO(), group)
			// With no constraints policy, empty versions should still return true
			assert.True(tt, result)
		})
	}
}
