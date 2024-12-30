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

package common

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type fakeState[T any] struct {
	ns   string
	name string
	obj  *T
}

func (f *fakeState[T]) Object() *T {
	return f.obj
}

func (f *fakeState[T]) Initializer() ResourceInitializer[T] {
	return NewResource(func(obj *T) { f.obj = obj }).
		WithNamespace(Namespace(f.ns)).
		WithName(Name(f.name)).
		Initializer()
}

type fakeSliceState[T any] struct {
	ns     string
	labels map[string]string
	objs   []*T
}

func (f *fakeSliceState[T]) Slice() []*T {
	return f.objs
}

func (f *fakeSliceState[T]) Initializer() ResourceSliceInitializer[T] {
	return NewResourceSlice(func(objs []*T) { f.objs = objs }).
		WithNamespace(Namespace(f.ns)).
		WithLabels(Labels(f.labels)).
		Initializer()
}

type fakePDState struct {
	s *fakeState[v1alpha1.PD]
}

func (f *fakePDState) PD() *v1alpha1.PD {
	return f.s.Object()
}

func (f *fakePDState) PDInitializer() PDInitializer {
	return f.s.Initializer()
}

type fakeClusterState struct {
	s *fakeState[v1alpha1.Cluster]
}

func (f *fakeClusterState) Cluster() *v1alpha1.Cluster {
	return f.s.Object()
}

func (f *fakeClusterState) ClusterInitializer() ClusterInitializer {
	return f.s.Initializer()
}

type fakePodState struct {
	s *fakeState[corev1.Pod]
}

func (f *fakePodState) Pod() *corev1.Pod {
	return f.s.Object()
}

func (f *fakePodState) PodInitializer() PodInitializer {
	return f.s.Initializer()
}

type fakePDSliceState struct {
	s *fakeSliceState[v1alpha1.PD]
}

func (f *fakePDSliceState) PDSlice() []*v1alpha1.PD {
	return f.s.Slice()
}

func (f *fakePDSliceState) PDSliceInitializer() PDSliceInitializer {
	return f.s.Initializer()
}

type fakeGroupState[RG runtime.Group] struct {
	g RG
}

func (f *fakeGroupState[RG]) Group() RG {
	return f.g
}

func FakeGroupState[RG runtime.Group](g RG) GroupState[RG] {
	return &fakeGroupState[RG]{
		g: g,
	}
}

type fakeInstanceSliceState[RI runtime.Instance] struct {
	slice []RI
}

func (f *fakeInstanceSliceState[RI]) Slice() []RI {
	return f.slice
}

func FakeInstanceSliceState[RI runtime.Instance](in []RI) InstanceSliceState[RI] {
	return &fakeInstanceSliceState[RI]{
		slice: in,
	}
}

type fakeGroupAndInstanceSliceState[
	RG runtime.Group,
	RI runtime.Instance,
] struct {
	GroupState[RG]
	InstanceSliceState[RI]
}

func FakeGroupAndInstanceSliceState[
	RG runtime.Group,
	RI runtime.Instance,
](g RG, s ...RI) GroupAndInstanceSliceState[RG, RI] {
	return &fakeGroupAndInstanceSliceState[RG, RI]{
		GroupState:         FakeGroupState[RG](g),
		InstanceSliceState: FakeInstanceSliceState[RI](s),
	}
}
