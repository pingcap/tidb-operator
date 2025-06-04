package state

import (
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

type IFeatureGates interface {
	FeatureGates() features.Gates
}

type IObjectAndCluster[T client.Object] interface {
	IObject[T]
	ICluster
}

type featureGates[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
] struct {
	obj IObjectAndCluster[F]

	gates features.Gates
}

func (s *featureGates[S, F, T]) FeatureGates() features.Gates {
	if s.gates == nil {
		gates := features.New[S](s.obj.Object())
		// if feature modification is not enabled, use features defined in cluster directly
		if !gates.Enabled(metav1alpha1.FeatureModification) {
			c := s.obj.Cluster()
			fs := coreutil.EnabledFeatures(c)
			s.gates = features.NewFromFeatures(fs)
		} else {
			s.gates = gates
		}
	}

	return s.gates
}

func NewFeatureGates[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](obj IObjectAndCluster[F]) IFeatureGates {
	return &featureGates[S, F, T]{
		obj: obj,
	}
}
