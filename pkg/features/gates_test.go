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

package features

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"

	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

func installTestDefaults(t *testing.T) {
	t.Helper()
	// Tests using this fixture must remain non-parallel.
	originalLogs, originalHashes, originalDefaults := logs, hashToRev, defaultGates
	t.Cleanup(func() { logs, hashToRev, defaultGates = originalLogs, originalHashes, originalDefaults })
	logs = [][]FeatureLog{
		{
			{Name: "Alpha", Stage: meta.FeatureStageAlpha, Default: false},
			{Name: "Beta", Stage: meta.FeatureStageBeta, Default: true},
		},
		{
			{Name: "Alpha", Stage: meta.FeatureStageAlpha, Default: false},
			{Name: "Beta", Stage: meta.FeatureStageBeta, Default: true},
			{Name: "NewBeta", Stage: meta.FeatureStageBeta, Default: true},
		},
	}
	hashToRev = map[string]int{"old": 0, "new": 1}
	defaultGates = NewFromLogs(logs[1])
}

func TestGroupInstanceGatesIgnoreDefaults(t *testing.T) {
	installTestDefaults(t)
	fs := []meta.Feature{"Alpha", "Alpha"}
	group := NewFromFeatures(fs)
	if !group.Enabled("Alpha") || group.Enabled("Beta") || group.Enabled("NewBeta") {
		t.Fatal("Group/Instance gates must use only their features field")
	}
	if NewFromFeatures(nil).Enabled("Beta") {
		t.Fatal("empty features must disable everything")
	}
	fs[0] = "Beta"
	if group.Enabled("Beta") {
		t.Fatal("input mutation changed gates")
	}
}

func TestHistoricalDefaultGates(t *testing.T) {
	installTestDefaults(t)
	old, err := NewFromHash("old")
	if err != nil {
		t.Fatal(err)
	}
	latest, err := NewFromHash("new")
	if err != nil {
		t.Fatal(err)
	}
	if old.Enabled("Alpha") || !old.Enabled("Beta") || old.Enabled("NewBeta") || !latest.Enabled("NewBeta") {
		t.Fatal("defaults were not reconstructed at the requested revision")
	}
	for _, hash := range []string{"", "unknown"} {
		if got, err := NewFromHash(hash); err == nil || got != nil {
			t.Fatal("invalid hash accepted")
		}
		if got, err := NewFromClusterWithHash(nil, hash); err == nil || got != nil {
			t.Fatal("invalid cluster hash accepted")
		}
	}
}

func TestClusterOverrides(t *testing.T) {
	installTestDefaults(t)
	disabled := false
	enabled := true
	spec := []meta.FeatureGate{
		{Name: "Alpha"},
		{Name: "Beta", Enabled: &disabled},
		{Name: "Explicit", Enabled: &enabled},
	}
	configured := NewFromCluster(spec)
	if !configured.Enabled("Alpha") || configured.Enabled("Beta") || !configured.Enabled("NewBeta") || !configured.Enabled("Explicit") {
		t.Fatal("nil means true, explicit false overrides default, absent features use defaults")
	}
	disabled = true
	spec[0].Name = "Changed"
	if configured.Enabled("Beta") || !configured.Enabled("Alpha") || configured.Enabled("Changed") {
		t.Fatal("spec mutation changed constructed gates")
	}
	if !defaultGates.Enabled("Beta") || defaultGates.Enabled("Alpha") {
		t.Fatal("cluster overrides mutated defaults")
	}
}

func TestHistoricalClusterGates(t *testing.T) {
	installTestDefaults(t)
	disabled := false
	historical, err := NewFromClusterWithHash([]meta.FeatureGate{{Name: "Beta", Enabled: &disabled}}, "old")
	if err != nil {
		t.Fatal(err)
	}
	if historical.Enabled("Beta") || historical.Enabled("NewBeta") {
		t.Fatal("historical overrides or defaults are incorrect")
	}
	if !NewFromCluster(nil).Enabled("NewBeta") {
		t.Fatal("empty cluster spec must use defaults")
	}
}

func TestCurrentDefaultFeatureGates(t *testing.T) {
	defaults, err := NewFromHash(CurrentFeatureGateDefinitionHash)
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range logs[len(logs)-1] {
		if defaults.Enabled(definition.Name) != definition.Default {
			t.Fatalf("wrong default for %s", definition.Name)
		}
	}
}

func TestFeatureGates(t *testing.T) {
	cases := []struct {
		desc string

		obj *v1alpha1.PD

		feat    meta.Feature
		enabled bool
	}{
		{
			desc: "aaa is enabled",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Spec.Features = []meta.Feature{"aaa"}
				return obj
			}),
			feat:    meta.Feature("aaa"),
			enabled: true,
		},
		{
			desc: "bbb is not enabled",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Spec.Features = []meta.Feature{"aaa"}
				return obj
			}),
			feat:    meta.Feature("bbb"),
			enabled: false,
		},
		{
			desc: "no feature",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			}),
			feat:    meta.Feature("aaa"),
			enabled: false,
		},
	}

	for i := range cases {
		c := &cases[i]

		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()
			fg := NewFromObject[scope.PD](c.obj)
			assert.Equal(tt, c.enabled, fg.Enabled(c.feat), c.desc)
		})
	}
}

func TestObjectGatesIgnoreDefaults(t *testing.T) {
	installTestDefaults(t)
	obj := &v1alpha1.PD{}
	if NewFromObject[scope.PD](obj).Enabled("Beta") {
		t.Fatal("instance constructor must not consult default gates")
	}
	obj.Spec.Features = []meta.Feature{"Beta"}
	if !NewFromObject[scope.PD](obj).Enabled("Beta") {
		t.Fatal("instance constructor must honor its own features")
	}
}
