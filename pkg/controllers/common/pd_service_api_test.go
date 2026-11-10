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
	"context"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	operatorclient "github.com/pingcap/tidb-operator/v2/pkg/client"
)

func TestReconcilePDServiceAPI(t *testing.T) {
	assets := os.Getenv("KUBEBUILDER_ASSETS")
	if assets == "" {
		t.Skip("set KUBEBUILDER_ASSETS to run the Kubernetes API-server test")
	}
	server := &envtest.Environment{BinaryAssetsDirectory: assets}
	config, err := server.Start()
	require.NoError(t, err)
	defer func() { require.NoError(t, server.Stop()) }()
	c, err := operatorclient.New(config, operatorclient.GroupVersions(corev1.SchemeGroupVersion))
	require.NoError(t, err)
	for _, first := range []string{"old SSA", "guarded create"} {
		t.Run(first, func(t *testing.T) {
			ctx := t.Context()
			group := &v1alpha1.PDGroup{ObjectMeta: metav1.ObjectMeta{Name: "legacy", UID: "group-uid"}}
			clusterOwner := metav1.OwnerReference{APIVersion: "v1", Kind: "ConfigMap", Name: "cluster", UID: "cluster-uid", Controller: ptr.To(true)}
			groupOwner := metav1.OwnerReference{APIVersion: "v1", Kind: "ConfigMap", Name: "legacy", UID: group.UID, Controller: ptr.To(true)}
			desired := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "old-ssa", Namespace: "default", OwnerReferences: []metav1.OwnerReference{groupOwner}},
				Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, Selector: map[string]string{"group": "legacy"},
					Ports: []corev1.ServicePort{
						{Name: "client", Protocol: corev1.ProtocolTCP, Port: 2379, TargetPort: intstr.FromString("client")},
						{Name: "peer", Protocol: corev1.ProtocolTCP, Port: 2380, TargetPort: intstr.FromString("peer")},
					}},
			}
			if first == "old SSA" {
				require.NoError(t, c.Apply(ctx, desired.DeepCopy()))
			} else {
				desired.Name = "guarded-create"
				require.NoError(t, ReconcilePDService(ctx, c, c, desired.DeepCopy()))
			}

			key := client.ObjectKeyFromObject(desired)
			get := func() *corev1.Service {
				var service corev1.Service
				require.NoError(t, c.Get(ctx, key, &service))
				return &service
			}
			before := get()
			// The spec may already have changed while the old Service still has port 2380.
			group.Spec.Template.Spec.Server.Ports.Peer = &v1alpha1.Port{Port: 3380}
			cluster := desired.DeepCopy()
			cluster.OwnerReferences = []metav1.OwnerReference{clusterOwner}
			cluster.Spec.Ports = cluster.Spec.Ports[:1]
			cluster.Spec.Selector = map[string]string{"cluster": "cluster"}
			require.NoError(t, ReconcilePDService(ctx, c, c, cluster, group))
			migrated := get()
			require.Equal(t, before.UID, migrated.UID)
			require.Equal(t, before.Spec.ClusterIP, migrated.Spec.ClusterIP)
			require.Equal(t, []metav1.OwnerReference{clusterOwner}, migrated.OwnerReferences)
			require.Len(t, migrated.Spec.Ports, 1)

			apply := []byte(`{"apiVersion":"v1","kind":"Service","spec":{"ports":[{"name":"peer","port":9000,"protocol":"TCP","targetPort":9000},{"name":"client","port":2379,"protocol":"TCP","appProtocol":"kubernetes.io/h2c"}]}}`)
			require.NoError(t, c.Patch(ctx, migrated, client.RawPatch(types.ApplyPatchType, apply), client.FieldOwner("external")))
			external := get()
			require.NoError(t, ReconcilePDService(ctx, c, c, cluster, group))
			current := get()
			require.Equal(t, external.ResourceVersion, current.ResourceVersion)

			require.Len(t, current.Spec.Ports, 2)
			require.True(t, slices.ContainsFunc(current.Spec.Ports, func(port corev1.ServicePort) bool {
				return port.Name == "client" && port.AppProtocol != nil && *port.AppProtocol == "kubernetes.io/h2c"
			}), "%+v", current.Spec.Ports)
			require.True(t, slices.ContainsFunc(current.Spec.Ports, func(port corev1.ServicePort) bool {
				return port.Name == "peer" && port.Port == 9000
			}), "%+v", current.Spec.Ports)

			// A competing mutation after the read invalidates the guarded Apply.
			stale := &beforePDServiceAPIPatch{Client: c, run: func() {
				newService := get()
				newService.Annotations = map[string]string{"concurrent": "write"}
				require.NoError(t, c.Update(ctx, newService))
			}}
			cluster.Spec.Selector["extra"] = "change"
			require.True(t, errors.IsConflict(ReconcilePDService(ctx, stale, c, cluster, group)))

			// Restore the old owner with the rollback writer and reapply its Service.
			current = get()
			returned := current.DeepCopy()
			returned.OwnerReferences = []metav1.OwnerReference{groupOwner}
			require.NoError(t, c.Patch(ctx, returned, client.MergeFromWithOptions(current, client.MergeFromWithOptimisticLock{}), client.FieldOwner(PDServiceUpdateManager)))
			// Remove the external port to isolate the legacy name and ownership assertions.
			clearExternal := []byte(`{"apiVersion":"v1","kind":"Service"}`)
			require.NoError(t, c.Patch(ctx, returned, client.RawPatch(types.ApplyPatchType, clearExternal), client.FieldOwner("external")))
			require.NoError(t, ReconcilePDService(ctx, c, c, desired))
			require.Equal(t, []metav1.OwnerReference{groupOwner}, get().OwnerReferences)
			require.Len(t, get().Spec.Ports, 2)
			require.NoError(t, ReconcilePDService(ctx, c, c, cluster, group))
			require.Len(t, get().Spec.Ports, 1)
		})
	}
}

type beforePDServiceAPIPatch struct {
	client.Client
	run func()
}

func (c *beforePDServiceAPIPatch) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	c.run()
	return c.Client.Patch(ctx, obj, patch, opts...)
}
