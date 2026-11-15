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
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func TestReconcilePDServiceRejectsForeignOwnerOnCacheMiss(t *testing.T) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pd", Namespace: "default", UID: "service", ResourceVersion: "1",
			OwnerReferences: []metav1.OwnerReference{{UID: "foreign", Controller: ptr.To(true)}},
		},
		Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, ClusterIP: "10.0.0.1"},
	}
	c := client.NewFakeClient(service)
	c.WithError("get", "services", errors.NewNotFound(schema.GroupResource{Resource: "services"}, service.Name))
	desired := service.DeepCopy()
	desired.OwnerReferences[0].UID = "cluster"
	err := ReconcilePDService(t.Context(), c, client.NewFakeClient(service), desired)
	require.ErrorContains(t, err, "not controlled")
	var services corev1.ServiceList
	require.NoError(t, c.List(t.Context(), &services))
	require.Equal(t, []corev1.Service{*service}, services.Items)
}
