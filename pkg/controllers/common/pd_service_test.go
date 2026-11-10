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
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func TestReconcilePDService(t *testing.T) {
	for _, tc := range []struct {
		name      string
		owner     types.UID
		headless  bool
		cacheMiss bool
		wantError bool
	}{
		{name: "update", owner: "cluster"},
		{name: "restore labels after cache miss", owner: "cluster", cacheMiss: true},
		{name: "adopt legacy PDGroup", owner: "legacy"},
		{name: "reject foreign owner after cache miss", owner: "foreign", cacheMiss: true, wantError: true},
		{name: "reject headless", owner: "cluster", headless: true, wantError: true},
		{name: "reject ownerless", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			desired := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "pd",
					Namespace:       "default",
					Labels:          map[string]string{v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator},
					OwnerReferences: []metav1.OwnerReference{{UID: "cluster", Controller: ptr.To(true)}},
				},
				Spec: corev1.ServiceSpec{
					Type:     corev1.ServiceTypeClusterIP,
					Selector: map[string]string{"cluster": "target"},
					Ports: []corev1.ServicePort{
						{Name: "client", Port: 2379, Protocol: corev1.ProtocolTCP},
						{Name: "peer", Port: 2380, Protocol: corev1.ProtocolTCP},
					},
				},
			}
			current := desired.DeepCopy()
			current.UID = "service"
			current.ResourceVersion = "1"
			current.Spec.ClusterIP = "10.0.0.1"
			current.Spec.Selector = map[string]string{"cluster": "original"}
			current.Spec.Ports = []corev1.ServicePort{
				{Name: "client", Port: 2381, Protocol: corev1.ProtocolTCP},
				{Name: "metrics", Port: 2379, Protocol: corev1.ProtocolTCP, AppProtocol: ptr.To("metrics-protocol")},
				{Name: "peer", Port: 2380, Protocol: corev1.ProtocolTCP, AppProtocol: ptr.To("peer-protocol")},
				{Name: "metrics-udp", Port: 2379, Protocol: corev1.ProtocolUDP},
			}
			current.OwnerReferences[0].UID = tc.owner
			if tc.owner == "" {
				current.OwnerReferences = nil
			}
			if tc.headless {
				current.Spec.ClusterIP = corev1.ClusterIPNone
			}

			if tc.cacheMiss {
				current.Labels = nil
			}
			c := client.NewFakeClient(current)
			apiReader := c
			if tc.cacheMiss {
				apiReader = client.NewFakeClient(current)
				c.WithError("get", "services", errors.NewNotFound(schema.GroupResource{Resource: "services"}, current.Name))
			}
			legacy := &v1alpha1.PDGroup{ObjectMeta: metav1.ObjectMeta{UID: "legacy"}}
			err := ReconcilePDService(t.Context(), c, apiReader, desired, legacy)

			var services corev1.ServiceList
			require.NoError(t, c.List(t.Context(), &services))
			require.Len(t, services.Items, 1)
			got := &services.Items[0]
			if tc.wantError {
				require.Error(t, err)
				require.Equal(t, current, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, current.UID, got.UID)
			require.Equal(t, current.Spec.ClusterIP, got.Spec.ClusterIP)
			require.Equal(t, desired.OwnerReferences, got.OwnerReferences)
			require.Equal(t, desired.Spec.Selector, got.Spec.Selector)
			require.Equal(t, desired.Labels, got.Labels)
			wantPorts := []corev1.ServicePort{desired.Spec.Ports[0], desired.Spec.Ports[1], current.Spec.Ports[3]}
			wantPorts[0].AppProtocol = current.Spec.Ports[1].AppProtocol
			wantPorts[1].AppProtocol = current.Spec.Ports[2].AppProtocol
			require.Equal(t, wantPorts, got.Spec.Ports)
		})
	}
}

func TestReconcilePDServiceRejectsStaleWrite(t *testing.T) {
	ctx := t.Context()
	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "pd",
			Namespace:       "default",
			OwnerReferences: []metav1.OwnerReference{{UID: "legacy", Controller: ptr.To(true)}},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"group": "legacy"},
		},
	}
	current := desired.DeepCopy()
	current.Spec.Selector = map[string]string{"group": "original"}
	c := ctrlfake.NewClientBuilder().WithObjects(current).Build()
	key := ctrlclient.ObjectKeyFromObject(desired)
	staleWriter := &beforePDServicePatch{
		Client: c,
		run: func() {
			transferred := &corev1.Service{}
			require.NoError(t, c.Get(ctx, key, transferred))
			transferred.OwnerReferences[0].UID = "cluster"
			transferred.Spec.Selector = map[string]string{"cluster": "current"}
			require.NoError(t, c.Update(ctx, transferred))
		},
	}

	err := ReconcilePDService(ctx, staleWriter, c, desired)
	require.True(t, errors.IsConflict(err), "expected conflict, got %v", err)
	got := &corev1.Service{}
	require.NoError(t, c.Get(ctx, key, got))
	require.Equal(t, types.UID("cluster"), metav1.GetControllerOf(got).UID)
	require.Equal(t, map[string]string{"cluster": "current"}, got.Spec.Selector)
}

type beforePDServicePatch struct {
	ctrlclient.Client
	run func()
}

func (c *beforePDServicePatch) Patch(ctx context.Context, obj ctrlclient.Object, patch ctrlclient.Patch, opts ...ctrlclient.PatchOption) error {
	c.run()
	return c.Client.Patch(ctx, obj, patch, opts...)
}

func TestReconcilePDServiceKeepsUnmanagedPeerPort(t *testing.T) {
	legacy := &v1alpha1.PDGroup{ObjectMeta: metav1.ObjectMeta{UID: "legacy"}}
	for _, tc := range []struct {
		name     string
		owner    types.UID
		peerPort int32
		wantPeer bool
	}{
		{name: "regular reconcile preserves external peer", owner: "cluster", peerPort: 9000, wantPeer: true},
		{name: "migration drops legacy peer", owner: legacy.UID, peerPort: v1alpha1.DefaultPDPortPeer},
		{name: "migration preserves external peer", owner: legacy.UID, peerPort: 9000, wantPeer: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			desired := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pd", Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{{UID: "cluster", Controller: ptr.To(true)}},
				},
				Spec: corev1.ServiceSpec{
					Type:  corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{{Name: "client", Port: 2379, Protocol: corev1.ProtocolTCP}},
				},
			}
			current := desired.DeepCopy()
			current.ResourceVersion = "1"
			current.OwnerReferences[0].UID = tc.owner
			peer := corev1.ServicePort{
				Name: "peer", Port: tc.peerPort, Protocol: corev1.ProtocolTCP,
				TargetPort: intstr.FromString("peer"),
			}
			current.Spec.Ports = append(current.Spec.Ports, peer)
			c := client.NewFakeClient(current)

			require.NoError(t, ReconcilePDService(t.Context(), c, c, desired, legacy))
			var got corev1.Service
			require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(desired), &got))
			want := desired.Spec.Ports
			if tc.wantPeer {
				want = append(want, peer)
			}
			require.Equal(t, want, got.Spec.Ports)
		})
	}
}
