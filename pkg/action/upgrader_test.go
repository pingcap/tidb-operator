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
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func Test_areGroupsUpgraded(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		groups    []v1alpha1.Group
		want      bool
		wantError bool
	}{
		{
			name:      "invalid version",
			version:   "foo",
			groups:    []v1alpha1.Group{},
			want:      false,
			wantError: true,
		},
		{
			name:      "no groups",
			version:   "v8.1.0",
			groups:    []v1alpha1.Group{},
			want:      true,
			wantError: false,
		},
		{
			name:    "all groups upgraded to the same version",
			version: "v8.1.0",
			groups: []v1alpha1.Group{
				&v1alpha1.FakeGroup{Healthy: true, ActualVersion: "v8.1.0"},
				&v1alpha1.FakeGroup{Healthy: true, ActualVersion: "v8.1.0"},
			},
			want:      true,
			wantError: false,
		},
		{
			name:    "all groups upgraded to the newer version",
			version: "v8.1.0",
			groups: []v1alpha1.Group{
				&v1alpha1.FakeGroup{Healthy: true, ActualVersion: "v8.1.1"},
				&v1alpha1.FakeGroup{Healthy: true, ActualVersion: "v8.1.1"},
			},
			want:      true,
			wantError: false,
		},
		{
			name:    "one group not upgraded",
			version: "v6.5.1",
			groups: []v1alpha1.Group{
				&v1alpha1.FakeGroup{Healthy: true, ActualVersion: "v6.5.1"},
				&v1alpha1.FakeGroup{Healthy: true, ActualVersion: "v6.5.0"},
			},
			want:      false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := areGroupsUpgraded(tt.version, tt.groups)
			if (err != nil) != tt.wantError {
				t.Errorf("areGroupsUpgraded() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("areGroupsUpgraded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getDependentGroups(t *testing.T) {
	tests := []struct {
		name         string
		existingObjs []client.Object
		group        v1alpha1.Group
		wantGroups   []v1alpha1.Group
		wantErr      bool
	}{
		{
			name:  "pd: no dependent groups",
			group: &v1alpha1.FakeGroup{ComponentKindVal: v1alpha1.ComponentKindPD},
		},
		{
			name:       "tidb depends on tikv but not found tikv groups",
			group:      &v1alpha1.FakeGroup{ComponentKindVal: v1alpha1.ComponentKindTiDB},
			wantGroups: []v1alpha1.Group{},
		},
		{
			name: "tikv depends on tiflash when has tiflash",
			existingObjs: []client.Object{
				fake.FakeObj[v1alpha1.TiFlashGroup]("tiflash",
					fake.SetNamespace[v1alpha1.TiFlashGroup]("test"),
					fake.Label[v1alpha1.TiFlashGroup](v1alpha1.LabelKeyCluster, "tc"),
					fake.Label[v1alpha1.TiFlashGroup](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentTiFlash),
				),
				fake.FakeObj[v1alpha1.PDGroup]("pd",
					fake.SetNamespace[v1alpha1.PDGroup]("test"),
					fake.Label[v1alpha1.PDGroup](v1alpha1.LabelKeyCluster, "tc"),
					fake.Label[v1alpha1.PDGroup](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
				),
			},
			group: &v1alpha1.FakeGroup{ComponentKindVal: v1alpha1.ComponentKindTiKV, Namespace: "test", ClusterName: "tc"},
			wantGroups: []v1alpha1.Group{
				fake.FakeObj[v1alpha1.TiFlashGroup]("tiflash",
					fake.SetNamespace[v1alpha1.TiFlashGroup]("test"),
					fake.Label[v1alpha1.TiFlashGroup](v1alpha1.LabelKeyCluster, "tc"),
					fake.Label[v1alpha1.TiFlashGroup](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentTiFlash),
				),
			},
		},
		{
			name: "tikv depends on pd when has no tiflash",
			existingObjs: []client.Object{
				fake.FakeObj[v1alpha1.PDGroup]("pd",
					fake.SetNamespace[v1alpha1.PDGroup]("test"),
					fake.Label[v1alpha1.PDGroup](v1alpha1.LabelKeyCluster, "tc"),
					fake.Label[v1alpha1.PDGroup](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
				),
			},
			group: &v1alpha1.FakeGroup{ComponentKindVal: v1alpha1.ComponentKindTiKV, Namespace: "test", ClusterName: "tc"},
			wantGroups: []v1alpha1.Group{
				fake.FakeObj[v1alpha1.PDGroup]("pd",
					fake.SetNamespace[v1alpha1.PDGroup]("test"),
					fake.Label[v1alpha1.PDGroup](v1alpha1.LabelKeyCluster, "tc"),
					fake.Label[v1alpha1.PDGroup](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
				),
			},
		},
		{
			name: "tiflash depends on pd",
			existingObjs: []client.Object{
				fake.FakeObj[v1alpha1.PDGroup]("pd",
					fake.SetNamespace[v1alpha1.PDGroup]("test"),
					fake.Label[v1alpha1.PDGroup](v1alpha1.LabelKeyCluster, "tc"),
					fake.Label[v1alpha1.PDGroup](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
				),
			},
			group: &v1alpha1.FakeGroup{ComponentKindVal: v1alpha1.ComponentKindTiFlash, Namespace: "test", ClusterName: "tc"},
			wantGroups: []v1alpha1.Group{
				fake.FakeObj[v1alpha1.PDGroup]("pd",
					fake.SetNamespace[v1alpha1.PDGroup]("test"),
					fake.Label[v1alpha1.PDGroup](v1alpha1.LabelKeyCluster, "tc"),
					fake.Label[v1alpha1.PDGroup](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := client.NewFakeClient(tt.existingObjs...)
			gotGroups, err := getDependentGroups(context.TODO(), cli, tt.group)
			if (err != nil) != tt.wantErr {
				t.Errorf("getDependentGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotGroups, tt.wantGroups) {
				t.Errorf("getDependentGroups() gotGroups = %v, want %v", gotGroups, tt.wantGroups)
			}
		})
	}
}

func TestUpgradePolicy(t *testing.T) {
	pdg := &v1alpha1.PDGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "pd",
			Labels: map[string]string{
				v1alpha1.LabelKeyCluster:   "tc",
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
			},
		},
		Spec: v1alpha1.PDGroupSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: "tc",
			},
			Template: v1alpha1.PDTemplate{
				Spec: v1alpha1.PDTemplateSpec{
					Version: "v8.5.0",
				},
			},
		},
		Status: v1alpha1.PDGroupStatus{
			GroupStatus: v1alpha1.GroupStatus{
				Version: "v8.1.0", // not upgraded
			},
		},
	}
	kvg := &v1alpha1.TiKVGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "tikv",
			Labels: map[string]string{
				v1alpha1.LabelKeyCluster:   "tc",
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
			},
		},
		Spec: v1alpha1.TiKVGroupSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: "tc",
			},
			Template: v1alpha1.TiKVTemplate{
				Spec: v1alpha1.TiKVTemplateSpec{
					Version: "v8.5.0",
				},
			},
		},
		Status: v1alpha1.TiKVGroupStatus{
			GroupStatus: v1alpha1.GroupStatus{
				Version: "v8.1.0", // not upgraded
			},
		},
	}

	tests := []struct {
		name       string
		policy     v1alpha1.UpgradePolicy
		canUpgrade bool
	}{
		{
			name:       "no constraints",
			policy:     v1alpha1.UpgradePolicyNoConstraints,
			canUpgrade: true,
		},
		{
			name:       "default policy",
			policy:     v1alpha1.UpgradePolicyDefault,
			canUpgrade: false,
		},
		{
			name:       "unknown policy",
			policy:     "unknown",
			canUpgrade: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := v1alpha1.Cluster{
				Spec: v1alpha1.ClusterSpec{
					UpgradePolicy: tt.policy,
				},
			}
			cli := client.NewFakeClient(pdg, kvg)
			checker := NewUpgradeChecker(cli, &cluster, logr.Discard())
			got := checker.CanUpgrade(context.TODO(), kvg)
			if got != tt.canUpgrade {
				t.Errorf("CanUpgrade() got = %v, want %v", got, tt.canUpgrade)
			}
		})
	}
}
