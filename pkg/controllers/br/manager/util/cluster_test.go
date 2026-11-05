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

package util

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func TestBackupClusterConsent(t *testing.T) {
	for _, tc := range []struct {
		name, source, target, allowed string
		ok                            bool
	}{
		{"same namespace", "tenant", "tenant", "", true},
		{"absent consent", "tenant", "other", "", false},
		{"explicit consent", "tenant", "other", "tenant", true},
		{"consent list", "tenant", "other", "one, tenant, two", true},
		{"different source", "tenant", "other", "tenant-other", false},
		{"wildcard not supported", "tenant", "other", "*", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: tc.target, Annotations: map[string]string{AllowedBackupNamespaces: tc.allowed}}}
			cli := client.NewFakeClient(cluster)
			got, err := GetBackupCluster(context.Background(), cli, tc.source, &br.BRConfig{Cluster: "cluster", ClusterNamespace: tc.target})
			if tc.ok {
				require.NoError(t, err)
				require.Equal(t, tc.target, got.Namespace)
			} else {
				require.True(t, apierrors.IsForbidden(err))
				require.Nil(t, got)
			}
		})
	}
	cli := client.NewFakeClient()
	_, err := GetBackupCluster(context.Background(), cli, "tenant", nil)
	require.Error(t, err)
	_, err = GetBackupCluster(context.Background(), cli, "tenant", &br.BRConfig{Cluster: "missing"})
	require.True(t, apierrors.IsNotFound(err))
}
