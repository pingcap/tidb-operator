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

package k8s

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
)

// EnsureFinalizer ensures the finalizer is added to the object and updates the object if necessary.
func EnsureFinalizer(ctx context.Context, cli client.Client, obj client.Object) error {
	if controllerutil.AddFinalizer(obj, v1alpha1.Finalizer) {
		if err := cli.Update(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}

// RemoveFinalizer removes the finalizer from the object and updates the object if necessary.
func RemoveFinalizer(ctx context.Context, cli client.Client, obj client.Object) error {
	if controllerutil.RemoveFinalizer(obj, v1alpha1.Finalizer) {
		if err := cli.Update(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}
