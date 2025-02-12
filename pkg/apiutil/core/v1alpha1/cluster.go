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

package coreutil

import (
	"fmt"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

// ShouldSuspendCompute returns whether the cluster should suspend compute.
func ShouldSuspendCompute(c *v1alpha1.Cluster) bool {
	return c.Spec.SuspendAction != nil && c.Spec.SuspendAction.SuspendCompute
}

// IsTLSClusterEnabled returns whether the cluster has enabled mTLS.
func IsTLSClusterEnabled(c *v1alpha1.Cluster) bool {
	return c.Spec.TLSCluster != nil && c.Spec.TLSCluster.Enabled
}

// TLSClusterClientSecretName returns the mTLS secret name for the cluster client.
// TODO: move it to namer pkg
func TLSClusterClientSecretName(clusterName string) string {
	return fmt.Sprintf("%s-cluster-client-secret", clusterName)
}

func ShouldPauseReconcile(c *v1alpha1.Cluster) bool {
	return c.Spec.Paused
}
