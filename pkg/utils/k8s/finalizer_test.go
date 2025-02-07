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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func TestFinalizer(t *testing.T) {
	obj := fake.FakeObj[v1alpha1.PD]("pd-0")
	cli := client.NewFakeClient(obj)

	// Add finalizer
	err := EnsureFinalizer(context.Background(), cli, obj)
	require.NoError(t, err)

	// check if finalizer is added
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}, obj)
	require.NoError(t, err)
	require.Contains(t, obj.GetFinalizers(), v1alpha1.Finalizer)

	// call EnsureFinalizer again, should not update the object
	err = EnsureFinalizer(context.Background(), cli, obj)
	require.NoError(t, err)

	// Remove finalizer
	err = RemoveFinalizer(context.Background(), cli, obj)
	require.NoError(t, err)

	// check if finalizer is removed
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}, obj)
	require.NoError(t, err)
	require.NotContains(t, obj.GetFinalizers(), v1alpha1.Finalizer)
}
