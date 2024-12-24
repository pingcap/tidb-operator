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
