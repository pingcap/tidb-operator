package features

import (
	"slices"

	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// This variable records whether a component will be restarted when enabling/disabling a feature
// TODO: maybe moved to meta pkg
var unreloadable = map[meta.Feature][]meta.Component{
	meta.FeatureModification:  {},
	meta.VolumeAttributeClass: {},
	meta.DisablePDDefaultReadinessProbe: {
		meta.ComponentPD,
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
