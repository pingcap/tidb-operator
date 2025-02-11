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
			desc:       "default",
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: true,
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
		obj.SetLabels(map[string]string{
			v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
			v1alpha1.LabelKeyCluster:   cluster,
			v1alpha1.LabelKeyComponent: scope.Component[S](),
		})
		obj.SetCluster(cluster)
		obj.SetVersion(version)
		obj.SetStatusVersion(currentVersion)
		obj.SetReplicas(3)
		obj.SetStatusReplicas(3, 3, 3, 3)
		obj.SetStatusRevision("xxx", "xxx", nil)

		return obj
	})
	var s S
	return s.To(obj)
}
