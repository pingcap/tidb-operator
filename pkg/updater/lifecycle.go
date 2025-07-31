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

package updater

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

// ScaleInLifecycle defines an optional, extended lifecycle for scale-in operations.
// If a component provides an implementation of this interface, the updater will use it.
// Otherwise, it falls back to the default behavior (immediate deletion).
type ScaleInLifecycle[R runtime.Instance] interface {
	// IsOffline returns `spec.offline`.
	IsOffline(instance R) bool

	// IsOfflineCompleted should return true if the instance has finished the offline process
	// and is ready for final deletion.
	IsOfflineCompleted(instance R) bool

	// BeginScaleIn initiates the first step of the scale-in (e.g., sets `spec.offline = true`).
	// The actor will call this instead of immediate deletion.
	BeginScaleIn(ctx context.Context, instance R) error

	// CancelScaleIn aborts the scale-in process for an instance (e.g., sets `spec.offline = false`).
	CancelScaleIn(ctx context.Context, instance R) error
}

// storeLifecycle implements the updater.ScaleInLifecycle interface for store components (TiKV/TiFlash).
type storeLifecycle[SI runtime.StoreInstance] struct {
	cli client.Client
}

// NewStoreLifecycle creates a new storeLifecycle instance for store components (TiKV/TiFlash).
func NewStoreLifecycle[SI runtime.StoreInstance](cli client.Client) ScaleInLifecycle[SI] {
	return &storeLifecycle[SI]{
		cli: cli,
	}
}

func (s *storeLifecycle[SI]) IsOffline(instance SI) bool {
	return instance.IsOffline()
}

func (s *storeLifecycle[SI]) IsOfflineCompleted(instance SI) bool {
	cond := runtime.GetOfflineCondition(instance)
	return instance.IsOffline() && cond != nil && cond.Reason == v1alpha1.OfflineReasonCompleted
}

// toClientObject converts a runtime StoreInstance to a client.Object for Kubernetes operations.
// This handles the type conversion from runtime types (*runtime.TiKV, *runtime.TiFlash)
// to their corresponding v1alpha1 types (*v1alpha1.TiKV, *v1alpha1.TiFlash).
func toClientObject[SI runtime.StoreInstance](instance SI) client.Object {
	// Use type assertion to access the To() method that exists on concrete types
	switch v := any(instance).(type) {
	case *runtime.TiKV:
		return v.To()
	case *runtime.TiFlash:
		return v.To()
	default:
		// This should not happen with properly generated runtime types
		panic("should not happen")
	}
}

func (s *storeLifecycle[SI]) BeginScaleIn(ctx context.Context, instance SI) error {
	if instance.IsOffline() {
		return nil
	}
	obj := toClientObject(instance)
	patchData := []byte(`[{"op": "replace", "path": "/spec/offline", "value": true}]`)
	return s.cli.Patch(ctx, obj, client.RawPatch(types.JSONPatchType, patchData))
}

func (s *storeLifecycle[SI]) CancelScaleIn(ctx context.Context, instance SI) error {
	if !instance.IsOffline() {
		return nil
	}
	obj := toClientObject(instance)
	patchData := []byte(`[{"op": "replace", "path": "/spec/offline", "value": false}]`)
	return s.cli.Patch(ctx, obj, client.RawPatch(types.JSONPatchType, patchData))
}
