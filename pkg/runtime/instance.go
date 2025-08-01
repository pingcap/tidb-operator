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
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

// StoreInstance represents an instance that is both a Store and Instance, like TiKV and TiFlash.
type StoreInstance interface {
	Instance

	IsOffline() bool
	SetOffline(bool)
}

func SetOfflineCondition(s StoreInstance, condition *metav1.Condition) {
	if condition == nil {
		return
	}

	conditions := s.Conditions()

	// Find existing condition
	updated := false
	for i := range conditions {
		if conditions[i].Type == v1alpha1.StoreOfflineConditionType {
			conditions[i] = *condition
			updated = true
		}
	}

	if !updated {
		conditions = append(conditions, *condition)
	}
	s.SetConditions(conditions)
}

func GetOfflineCondition(s StoreInstance) *metav1.Condition {
	return meta.FindStatusCondition(s.Conditions(), v1alpha1.StoreOfflineConditionType)
}

func IsOfflineCompleted(s StoreInstance) bool {
	cond := GetOfflineCondition(s)
	return s.IsOffline() && cond != nil && cond.Reason == v1alpha1.OfflineReasonCompleted
}

// Instance2Store converts an Instance to StoreInstance if it implements the StoreInstance interface.
func Instance2Store(instance Instance) (StoreInstance, bool) {
	storeInstance, ok := instance.(StoreInstance)
	return storeInstance, ok
}

// toClientObject converts a runtime StoreInstance to a client.Object for Kubernetes operations.
// This handles the type conversion from runtime types (*runtime.TiKV, *runtime.TiFlash)
// to their corresponding v1alpha1 types (*v1alpha1.TiKV, *v1alpha1.TiFlash).
func toClientObject(store StoreInstance) client.Object {
	switch v := any(store).(type) {
	case *TiKV:
		return v.To()
	case *TiFlash:
		return v.To()
	default:
		panic("this should not happen")
	}
}

func SetOffline(ctx context.Context, cli client.Client, store StoreInstance, offline bool) error {
	if offline == store.IsOffline() {
		return nil
	}

	obj := toClientObject(store)
	patchData := []byte(fmt.Sprintf(`[{"op": "replace", "path": "/spec/offline", "value": %v}]`, offline))
	return cli.Patch(ctx, obj, client.RawPatch(types.JSONPatchType, patchData))
}
