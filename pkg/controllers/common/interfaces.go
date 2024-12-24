package common

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
)

type Object[T any] interface {
	client.Object
	*T
}

type ObjectList[T any] interface {
	client.ObjectList
	*T
}

type (
	PDInitializer = ResourceInitializer[*v1alpha1.PD]

	ClusterInitializer = ResourceInitializer[*v1alpha1.Cluster]

	PodInitializer     = ResourceInitializer[*corev1.Pod]
	PDSliceInitializer = ResourceSliceInitializer[*v1alpha1.PD]
)

type PDStateInitializer interface {
	PDInitializer() PDInitializer
}

type PDState interface {
	PD() *v1alpha1.PD
}

type ClusterStateInitializer interface {
	ClusterInitializer() ClusterInitializer
}

type ClusterState interface {
	Cluster() *v1alpha1.Cluster
}

type PodStateInitializer interface {
	PodInitializer() PodInitializer
}

type PodState interface {
	Pod() *corev1.Pod
}

type PDSliceStateInitializer interface {
	PDSliceInitializer() PDSliceInitializer
}

type PDSliceState interface {
	PDSlice() []*v1alpha1.PD
}
