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

package volumes

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

// TestCapacityWithAPIServer checks real PVC validation and SSA conflict handling.
// Run with KUBEBUILDER_ASSETS pointing to envtest binaries.
func TestCapacityWithAPIServer(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("set KUBEBUILDER_ASSETS to run API server integration tests")
	}
	env := &envtest.Environment{}
	cfg, err := env.Start()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, env.Stop()) })
	cli, err := client.New(cfg, client.GroupVersions(corev1.SchemeGroupVersion))
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, cli.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "volume-test"}}))
	require.NoError(t, cli.Create(ctx, &storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "expandable"}, Provisioner: "example.test/storage", AllowVolumeExpansion: ptr.To(true)}))
	t.Run("vac", func(t *testing.T) {
		desired := capacityPVC("100Gi", "")
		desired.Name = "vac"
		desired.Namespace = "volume-test"
		desired.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		desired.Spec.StorageClassName = ptr.To("expandable")
		require.NoError(t, SyncPVCs(ctx, cli, []*corev1.PersistentVolumeClaim{desired}))
		current := &corev1.PersistentVolumeClaim{}
		require.NoError(t, cli.Get(ctx, client.ObjectKeyFromObject(desired), current))
		current.Status.Phase = corev1.ClaimBound
		current.Status.Capacity = corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("100Gi")}
		require.NoError(t, cli.Status().Update(ctx, current))
		desired.SetManagedFields(nil)
		desired.SetResourceVersion("")
		desired.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("50Gi")
		// Demonstrate that the API server rejects the unprotected decrease.
		require.Error(t, cli.Apply(ctx, desired.DeepCopy()))
		desired.Labels = map[string]string{"changed": "true"}
		require.NoError(t, SyncPVCs(ctx, cli, []*corev1.PersistentVolumeClaim{desired}))
		require.NoError(t, cli.Get(ctx, client.ObjectKeyFromObject(desired), current))
		require.Zero(t, current.Spec.Resources.Requests.Storage().Cmp(resource.MustParse("100Gi")))
		require.Equal(t, "true", current.Labels["changed"])
		// A newly created PVC still gets the reduced request.
		newPVC := capacityPVC("50Gi", "")
		newPVC.Name = "vac-new"
		newPVC.Namespace = desired.Namespace
		newPVC.Spec.AccessModes = desired.Spec.AccessModes
		newPVC.Spec.StorageClassName = desired.Spec.StorageClassName
		require.NoError(t, SyncPVCs(ctx, cli, []*corev1.PersistentVolumeClaim{newPVC}))
		require.NoError(t, cli.Get(ctx, client.ObjectKeyFromObject(newPVC), newPVC))
		require.Zero(t, newPVC.Spec.Resources.Requests.Storage().Cmp(resource.MustParse("50Gi")))
		// Expand after the normalizer's read but before its apply: the write must conflict.
		options := client.NewApplyOptions(desired)
		retainPVCRequestIfSufficient().With(options)
		desired.SetManagedFields(nil)
		desired.SetResourceVersion("")
		desired.Labels["concurrent"] = "true"
		err = cli.Apply(ctx, desired.DeepCopy(), client.Transformers(
			options.Transformers...,
		), client.Transformers(client.TransformerFunc(func(_, expected client.Object) client.Object {
			current.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("200Gi")
			require.NoError(t, cli.Update(ctx, current))
			return expected
		})))
		require.True(t, errors.IsConflict(err), "expected resource version conflict, got %v", err)
		require.NoError(t, SyncPVCs(ctx, cli, []*corev1.PersistentVolumeClaim{desired}))
		require.NoError(t, cli.Get(ctx, client.ObjectKeyFromObject(desired), current))
		require.Zero(t, current.Spec.Resources.Requests.Storage().Cmp(resource.MustParse("200Gi")))
		require.Equal(t, "true", current.Labels["concurrent"])
	})
	t.Run("excess capacity without expansion support", func(t *testing.T) {
		require.NoError(t, cli.Create(ctx, &storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "fixed"}, Provisioner: "example.test/storage", AllowVolumeExpansion: ptr.To(false)}))
		desired := capacityPVC("100Gi", "")
		desired.Name = "fixed"
		desired.Namespace = "volume-test"
		desired.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		desired.Spec.StorageClassName = ptr.To("fixed")
		require.NoError(t, SyncPVCs(ctx, cli, []*corev1.PersistentVolumeClaim{desired}))
		current := &corev1.PersistentVolumeClaim{}
		require.NoError(t, cli.Get(ctx, client.ObjectKeyFromObject(desired), current))
		current.Status.Phase = corev1.ClaimBound
		current.Status.Capacity = corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("150Gi")}
		require.NoError(t, cli.Status().Update(ctx, current))
		desired.SetManagedFields(nil)
		desired.SetResourceVersion("")
		desired.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("50Gi")
		require.NoError(t, SyncPVCs(ctx, cli, []*corev1.PersistentVolumeClaim{desired}))
		require.NoError(t, cli.Get(ctx, client.ObjectKeyFromObject(desired), current))
		require.Zero(t, current.Spec.Resources.Requests.Storage().Cmp(resource.MustParse("100Gi")))
	})
}
