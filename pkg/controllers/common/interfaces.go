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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
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
	ClusterInitializer = ResourceInitializer[v1alpha1.Cluster]

	PDGroupInitializer = ResourceInitializer[v1alpha1.PDGroup]
	PDInitializer      = ResourceInitializer[v1alpha1.PD]
	PDSliceInitializer = ResourceSliceInitializer[v1alpha1.PD]

	TiKVGroupInitializer = ResourceInitializer[v1alpha1.TiKVGroup]
	TiKVInitializer      = ResourceInitializer[v1alpha1.TiKV]
	TiKVSliceInitializer = ResourceSliceInitializer[v1alpha1.TiKV]

	TiDBGroupInitializer = ResourceInitializer[v1alpha1.TiDBGroup]
	TiDBInitializer      = ResourceInitializer[v1alpha1.TiDB]
	TiDBSliceInitializer = ResourceSliceInitializer[v1alpha1.TiDB]

	TiFlashGroupInitializer = ResourceInitializer[v1alpha1.TiFlashGroup]
	TiFlashInitializer      = ResourceInitializer[v1alpha1.TiFlash]
	TiFlashSliceInitializer = ResourceSliceInitializer[v1alpha1.TiFlash]

	TiCDCGroupInitializer = ResourceInitializer[v1alpha1.TiCDCGroup]
	TiCDCInitializer      = ResourceInitializer[v1alpha1.TiCDC]
	TiCDCSliceInitializer = ResourceSliceInitializer[v1alpha1.TiCDC]

	PodInitializer = ResourceInitializer[corev1.Pod]
)

type (
	ClusterStateInitializer interface {
		ClusterInitializer() ClusterInitializer
	}
	ClusterState interface {
		Cluster() *v1alpha1.Cluster
	}
)

type GroupStateInitializer[G runtime.GroupSet] interface {
	GroupInitializer() ResourceInitializer[G]
}

type GroupState[G runtime.Group] interface {
	Group() G
}

type InstanceState[I runtime.Instance] interface {
	Instance() I
}

type JobState[J runtime.Job] interface {
	Job() J
}

type InstanceAndPodState[I runtime.Instance] interface {
	InstanceState[I]
	PodState
}

type InstanceSliceState[I runtime.Instance] interface {
	Slice() []I
}

type ObjectState[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
] interface {
	Object() F
}

type GroupAndInstanceSliceState[
	G runtime.Group,
	I runtime.Instance,
] interface {
	GroupState[G]
	InstanceSliceState[I]
}

type RevisionStateInitializer[G runtime.Group] interface {
	RevisionInitializer() RevisionInitializer[G]
}

type RevisionState interface {
	Revision() (update, current string, collisionCount int32)
}

type GroupAndInstanceSliceAndRevisionState[
	G runtime.Group,
	I runtime.Instance,
] interface {
	GroupState[G]
	InstanceSliceState[I]
	RevisionState
}

type ClusterAndGroupAndInstanceSliceState[
	G runtime.Group,
	I runtime.Instance,
] interface {
	ClusterState
	GroupState[G]
	InstanceSliceState[I]
}

type (
	PDGroupStateInitializer interface {
		PDGroupInitializer() PDGroupInitializer
	}
	PDGroupState interface {
		PDGroup() *v1alpha1.PDGroup
	}
	PDStateInitializer interface {
		PDInitializer() PDInitializer
	}
	PDState interface {
		PD() *v1alpha1.PD
	}
	PDSliceStateInitializer interface {
		PDSliceInitializer() PDSliceInitializer
	}
	PDSliceState interface {
		PDSlice() []*v1alpha1.PD
	}
)

type (
	TiKVGroupStateInitializer interface {
		TiKVGroupInitializer() TiKVGroupInitializer
	}
	TiKVGroupState interface {
		TiKVGroup() *v1alpha1.TiKVGroup
	}
	TiKVStateInitializer interface {
		TiKVInitializer() TiKVInitializer
	}
	TiKVState interface {
		TiKV() *v1alpha1.TiKV
	}
	TiKVSliceStateInitializer interface {
		TiKVSliceInitializer() TiKVSliceInitializer
	}
	TiKVSliceState interface {
		TiKVSlice() []*v1alpha1.TiKV
	}
)

type (
	TiDBGroupStateInitializer interface {
		TiDBGroupInitializer() TiDBGroupInitializer
	}
	TiDBGroupState interface {
		TiDBGroup() *v1alpha1.TiDBGroup
	}
	TiDBStateInitializer interface {
		TiDBInitializer() TiDBInitializer
	}
	TiDBState interface {
		TiDB() *v1alpha1.TiDB
	}
	TiDBSliceStateInitializer interface {
		TiDBSliceInitializer() TiDBSliceInitializer
	}
	TiDBSliceState interface {
		TiDBSlice() []*v1alpha1.TiDB
	}
)

type (
	TiFlashGroupStateInitializer interface {
		TiFlashGroupInitializer() TiFlashGroupInitializer
	}
	TiFlashGroupState interface {
		TiFlashGroup() *v1alpha1.TiFlashGroup
	}
	TiFlashStateInitializer interface {
		TiFlashInitializer() TiFlashInitializer
	}
	TiFlashState interface {
		TiFlash() *v1alpha1.TiFlash
	}
	TiFlashSliceStateInitializer interface {
		TiFlashSliceInitializer() TiFlashSliceInitializer
	}
	TiFlashSliceState interface {
		TiFlashSlice() []*v1alpha1.TiFlash
	}
)

type (
	TiCDCGroupStateInitializer interface {
		TiCDCGroupInitializer() TiCDCGroupInitializer
	}
	TiCDCGroupState interface {
		TiCDCGroup() *v1alpha1.TiCDCGroup
	}
	TiCDCStateInitializer interface {
		TiCDCInitializer() TiCDCInitializer
	}
	TiCDCState interface {
		TiCDC() *v1alpha1.TiCDC
	}
	TiCDCSliceStateInitializer interface {
		TiCDCSliceInitializer() TiCDCSliceInitializer
	}
	TiCDCSliceState interface {
		TiCDCSlice() []*v1alpha1.TiCDC
	}
)

type (
	PodStateInitializer interface {
		PodInitializer() PodInitializer
	}
	PodState interface {
		Pod() *corev1.Pod
	}
)
