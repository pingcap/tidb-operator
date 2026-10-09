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
	"reflect"
	"testing"

	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

func TestClusterFeatures(t *testing.T) {
	installTestDefaults(t)
	disabled := false
	enabled := true
	spec := []meta.FeatureGate{
		{Name: "Alpha"},
		{Name: "Beta", Enabled: &disabled},
		{Name: "Explicit", Enabled: &enabled},
	}
	configured := NewFromCluster(spec)
	want := []meta.Feature{"Alpha", "Explicit", "NewBeta"}
	if got := ClusterFeatures(spec); !reflect.DeepEqual(got, want) {
		t.Fatalf("wrong propagated features: got %v, want %v", got, want)
	}
	group := NewFromFeatures(ClusterFeatures(spec))
	for _, name := range []meta.Feature{"Alpha", "Beta", "NewBeta", "Explicit", "Unknown"} {
		if group.Enabled(name) != configured.Enabled(name) {
			t.Fatalf("propagated gate mismatch for %s", name)
		}
	}
}
