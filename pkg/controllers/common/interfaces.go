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
	PDSliceInitializer      = ResourceSliceInitializer[v1alpha1.PD]
	TiKVSliceInitializer    = ResourceSliceInitializer[v1alpha1.TiKV]
	TiDBSliceInitializer    = ResourceSliceInitializer[v1alpha1.TiDB]
	TiFlashSliceInitializer = ResourceSliceInitializer[v1alpha1.TiFlash]
	TiCDCSliceInitializer   = ResourceSliceInitializer[v1alpha1.TiCDC]
	TiProxySliceInitializer = ResourceSliceInitializer[v1alpha1.TiProxy]
)

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
	PDGroupState interface {
		PDGroup() *v1alpha1.PDGroup
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
	TiKVGroupState interface {
		TiKVGroup() *v1alpha1.TiKVGroup
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
	TiDBGroupState interface {
		TiDBGroup() *v1alpha1.TiDBGroup
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
	TiFlashGroupState interface {
		TiFlashGroup() *v1alpha1.TiFlashGroup
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
	TiCDCGroupState interface {
		TiCDCGroup() *v1alpha1.TiCDCGroup
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
	TiProxyGroupState interface {
		TiProxyGroup() *v1alpha1.TiProxyGroup
	}
	TiProxyState interface {
		TiProxy() *v1alpha1.TiProxy
	}
	TiProxySliceStateInitializer interface {
		TiProxySliceInitializer() TiProxySliceInitializer
	}
	TiProxySliceState interface {
		TiProxySlice() []*v1alpha1.TiProxy
	}
)

type ObjectState[
	F client.Object,
] interface {
	Object() F
}

type ClusterState interface {
	Cluster() *v1alpha1.Cluster
}

type SliceState[
	F client.Object,
] interface {
	InstanceSlice() []F
}

type (
	PodState interface {
		Pod() *corev1.Pod
		IsPodTerminating() bool
	}
	PodStateUpdater interface {
		// It will be called after get or update api calls to k8s
		SetPod(pod *corev1.Pod)
		// Pod cannot be updated when call DELETE API, so we have to mark pod is deleting after calling DELETE api
		DeletePod(pod *corev1.Pod)
	}
)

type StoreState interface {
	GetStoreState() string
	// IsStoreUp returns true if the store state is `Preparing` or `Serving`,
	// which means the store is in the state of providing services.
	IsStoreUp() bool
}

type StoreStateUpdater interface {
	SetStoreState(string)
}

type HealthyState interface {
	// It means the instance is healthy to serve.
	// Normally, it's from PD or api exposed by the instance.
	// Now the operator checks it just like a liveness/readiness prober.
	// It may be removed if all components support /ready api in the future.
	// And then we can use pod's ready condition directly to check whether the instance
	// is ready.
	// But now, we still probe health from the operator.
	IsHealthy() bool
}

type HealthyStateUpdater interface {
	SetHealthy()
}
