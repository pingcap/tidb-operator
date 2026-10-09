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

	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

func TestClusterSubdomainIncludesDM(t *testing.T) {
	current := []meta.Feature{meta.FeatureModification}
	update := []meta.Feature{meta.FeatureModification, meta.ClusterSubdomain}
	for _, component := range []meta.Component{meta.ComponentDMMaster, meta.ComponentDMWorker} {
		if Reloadable(component, update, current) {
			t.Fatalf("ClusterSubdomain must restart %s", component)
		}
		if Reloadable(component, current, update) {
			t.Fatalf("disabling ClusterSubdomain must restart %s", component)
		}
	}
}
