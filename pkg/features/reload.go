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
	"slices"

	"k8s.io/apimachinery/pkg/util/sets"

	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

// This variable records whether a component will be restarted when enabling/disabling a feature
// IMPORTANT: Every Feature constant defined in api/meta/v1alpha1/feature.go MUST have
// a corresponding entry in this map. Use an empty slice {} if the feature doesn't require
// any component restarts, or specify the affected components if it does.
// The CI enforces this consistency with 'make verify/feature-gates'.
// TODO: maybe moved to meta pkg
var unreloadable = map[meta.Feature][]meta.Component{
	meta.FeatureModification:   {},
	meta.VolumeAttributesClass: {},
	meta.DisablePDDefaultReadinessProbe: {
		meta.ComponentPD,
	},
	meta.UsePDReadyAPI: {
		meta.ComponentPD,
	},
	meta.SessionTokenSigning: {
		meta.ComponentTiDB,
	},
}

func Reloadable(c meta.Component, update, current []meta.Feature) bool {
	updateSet, currentSet := sets.New(update...), sets.New(current...)

	// If FeatureModification is not enabled, we assume that features cannot be changed.
	// And we only allow to enable FeatureModification if it's not enabled.
	// It means that changes are always reloadable.
	if !currentSet.Has(meta.FeatureModification) {
		return true
	}

	diff := updateSet.SymmetricDifference(currentSet).UnsortedList()

	for _, f := range diff {
		cs, ok := unreloadable[f]
		if !ok {
			continue
		}
		if slices.Contains(cs, c) {
			return false
		}
	}

	return true
}
