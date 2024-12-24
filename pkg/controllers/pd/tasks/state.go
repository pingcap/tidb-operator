package tasks

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
)

type state struct {
	key types.NamespacedName

	cluster *v1alpha1.Cluster
	pd      *v1alpha1.PD
	pod     *corev1.Pod
	pds     []*v1alpha1.PD
}

type State interface {
	common.PDStateInitializer
	common.ClusterStateInitializer
	common.PodStateInitializer
	common.PDSliceStateInitializer

	common.PDState
	common.ClusterState
	common.PodState
	common.PDSliceState

	SetPod(*corev1.Pod)
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	return s
}

func (s *state) PD() *v1alpha1.PD {
	return s.pd
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) Pod() *corev1.Pod {
	return s.pod
}

func (s *state) SetPod(pod *corev1.Pod) {
	s.pod = pod
}

func (s *state) PDSlice() []*v1alpha1.PD {
	return s.pds
}

func (s *state) PDInitializer() common.PDInitializer {
	return common.NewResource(func(pd *v1alpha1.PD) { s.pd = pd }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Name(s.key.Name)).
		Initializer()
}

func (s *state) ClusterInitializer() common.ClusterInitializer {
	return common.NewResource(func(cluster *v1alpha1.Cluster) { s.cluster = cluster }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.NameFunc(func() string {
			return s.pd.Spec.Cluster.Name
		})).
		Initializer()
}

func (s *state) PodInitializer() common.PodInitializer {
	return common.NewResource(s.SetPod).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.NameFunc(func() string {
			return s.pd.PodName()
		})).
		Initializer()
}

func (s *state) PDSliceInitializer() common.PDSliceInitializer {
	return common.NewResourceSlice(func(pds []*v1alpha1.PD) { s.pds = pds }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithLabels(common.LabelsFunc(func() map[string]string {
			return map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
				v1alpha1.LabelKeyCluster:   s.cluster.Name,
			}
		})).
		Initializer()
}
