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
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

type Gates interface {
	Enabled(feat meta.Feature) bool
}

// gates is a complete configuration: absent features are disabled.
type gates map[meta.Feature]bool

func (g gates) Enabled(feat meta.Feature) bool {
	return g[feat]
}

// Current defaults are constructed once and shared as a read-only fallback.
var defaultGates = NewFromLogs(logs[len(logs)-1])

// NewFromLogs constructs default gates from a complete feature log snapshot.
func NewFromLogs(definitions []FeatureLog) Gates {
	defaults := make(gates, len(definitions))
	for _, definition := range definitions {
		defaults[definition.Name] = definition.Default
	}
	return defaults
}

// NewFromHash reconstructs the default gates at hash.
// Unknown hashes, including an empty hash, return an error.
func NewFromHash(hash string) (Gates, error) {
	definitions, err := DefinitionsAt(hash)
	if err != nil {
		return nil, err
	}
	if hash == CurrentFeatureGateDefinitionHash {
		return defaultGates, nil
	}
	return NewFromLogs(definitions), nil
}

// NewFromObject constructs gates using only the Group/Instance features field.
func NewFromObject[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](obj F) Gates {
	return NewFromFeatures(scope.From[S](obj).Features())
}

// NewFromFeatures builds Group/Instance gates from their complete feature list.
// It never consults default gates; an absent feature is disabled.
func NewFromFeatures(fs []meta.Feature) Gates {
	enabled := make(gates, len(fs))
	for _, feature := range fs {
		enabled[feature] = true
	}
	return enabled
}

// clusterGates contains spec overrides and a separate default configuration.
type clusterGates struct {
	overrides gates
	defaults  Gates
}

func (g *clusterGates) Enabled(feat meta.Feature) bool {
	if enabled, ok := g.overrides[feat]; ok {
		return enabled
	}
	return g.defaults.Enabled(feat)
}

// NewFromCluster applies Cluster spec overrides over the current default gates.
// An omitted Enabled means true; an omitted feature uses its default.
func NewFromCluster(fs []meta.FeatureGate) Gates {
	return NewFromClusterWithDefaults(fs, defaultGates)
}

// NewFromClusterWithHash applies Cluster spec overrides over historical defaults.
func NewFromClusterWithHash(fs []meta.FeatureGate, hash string) (Gates, error) {
	defaults, err := NewFromHash(hash)
	if err != nil {
		return nil, err
	}
	return NewFromClusterWithDefaults(fs, defaults), nil
}

// NewFromClusterWithDefaults applies Cluster spec overrides over the supplied default gates.
func NewFromClusterWithDefaults(fs []meta.FeatureGate, defaults Gates) Gates {
	overrides := make(gates, len(fs))
	for _, feature := range fs {
		overrides[feature.Name] = feature.Enabled == nil || *feature.Enabled
	}
	return &clusterGates{overrides: overrides, defaults: defaults}
}
