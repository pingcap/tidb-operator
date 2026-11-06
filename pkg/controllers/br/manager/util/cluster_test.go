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
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func TestBackupClusterKeepsCrossNamespaceReferences(t *testing.T) {
	cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "target"}}
	cli := client.NewFakeClient(cluster)
	got, err := GetBackupCluster(t.Context(), cli, "source", &br.BRConfig{Cluster: "cluster", ClusterNamespace: "target", Options: []string{"--pd=custom:2379"}})
	require.NoError(t, err)
	require.Equal(t, "target", got.Namespace)
	for _, config := range []*br.BRConfig{nil, {}} {
		_, err := GetBackupCluster(t.Context(), cli, "source", config)
		require.Error(t, err)
	}
}
func TestBROptionsPreserveUnrelatedArguments(t *testing.T) {
	for _, options := range [][]string{nil, {"-c=false", "-u", "custom:2379"}, {"--ratelimit", "-1"}, {"--pd=custom:2379"}, {"--filter=--storage=literal"}} {
		require.NoError(t, ValidateBROptions(&br.BRConfig{Options: options}))
	}
	for _, option := range []string{"--storage=s3://other/backup", "--s3.endpoint=https://other", "--gcs.endpoint=https://other", "--azblob.account-name=other"} {
		require.ErrorContains(t, ValidateBROptions(&br.BRConfig{Options: []string{option}}), "managed storage")
	}
}

func TestBROptionsCannotEndManagedStorageParsing(t *testing.T) {
	require.ErrorContains(t, ValidateBROptions(&br.BRConfig{Options: []string{"-ss3://other/backup", "--"}}), "terminate flag parsing")
}
