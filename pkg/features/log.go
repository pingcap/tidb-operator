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
	"fmt"
	"slices"

	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

// FeatureLog describes a feature at a particular revision.
type FeatureLog struct {
	Name    meta.Feature      `json:"name"`
	Stage   meta.FeatureStage `json:"stage"`
	Default bool              `json:"default"`
}

// DefinitionsAt returns a copy of the definitions referenced by hash.
// Unknown hashes are never interpreted as the current definitions.
func DefinitionsAt(hash string) ([]FeatureLog, error) {
	rev, ok := hashToRev[hash]
	if !ok {
		return nil, fmt.Errorf("unknown feature gate definition hash %q", hash)
	}
	return slices.Clone(logs[rev]), nil
}

// FeatureLogs returns a copy of the complete history, indexed by revision.
func FeatureLogs() [][]FeatureLog {
	entries := slices.Clone(logs)
	for i := range entries {
		entries[i] = slices.Clone(entries[i])
	}
	return entries
}
