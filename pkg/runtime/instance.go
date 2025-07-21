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

//go:generate ${GOBIN}/mockgen -write_command_comment=false -copyright_file ${BOILERPLATE_FILE} -destination instance_mock_generated.go -package=runtime ${GO_MODULE}/pkg/runtime Instance,StoreInstance
package runtime

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

type Instance interface {
	Object

	GetTopology() v1alpha1.Topology
	SetTopology(topo v1alpha1.Topology)

	GetUpdateRevision() string
	IsReady() bool
	// IsUpToDate means all resources managed by the instance is up to date
	// NOTE: It does not mean the instance is updated to the newest revision
	// TODO: may be change a more meaningful name?
	IsUpToDate() bool

	CurrentRevision() string
	SetCurrentRevision(rev string)

	PodOverlay() *v1alpha1.PodOverlay
}

type InstanceT[T InstanceSet] interface {
	Instance

	*T
}

type InstanceSet interface {
	PD | TiDB | TiKV | TiFlash | TiCDC | TiProxy
}

type InstanceTuple[PT client.Object, PU Instance] interface {
	Tuple[PT, PU]
}

// Store represents a TiKV or TiFlash store.
type Store interface {
	client.Object

	IsOffline() bool
	SetOffline(bool)
	GetOfflineCondition() *metav1.Condition
	SetOfflineCondition(metav1.Condition)
}

// StoreInstance represents an instance that is both a Store and Instance.
type StoreInstance interface {
	Instance
	Store
}
