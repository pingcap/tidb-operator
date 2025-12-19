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
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

// ShouldSuspendCompute returns whether the cluster should suspend compute.
func ShouldSuspendCompute(c *v1alpha1.Cluster) bool {
	return c.Spec.SuspendAction != nil && c.Spec.SuspendAction.SuspendCompute
}

// IsTLSClusterEnabled returns whether the cluster has enabled mTLS.
func IsTLSClusterEnabled(c *v1alpha1.Cluster) bool {
	return c.Spec.TLSCluster != nil && c.Spec.TLSCluster.Enabled
}

// LegacyTLSClusterClientSecretName returns the mTLS secret name for the cluster client.
// NOTE: now BR still use this secret to visit the cluster
//
// Deprecated: use coreutil.ClientCASecretName and coreutil.CLientCertKeyPairSecretName
func LegacyTLSClusterClientSecretName(clusterName string) string {
	return fmt.Sprintf("%s-cluster-client-secret", clusterName)
}

func ShouldPauseReconcile(c *v1alpha1.Cluster) bool {
	return c.Spec.Paused
}

func EnabledFeatures(c *v1alpha1.Cluster) []metav1alpha1.Feature {
	fs := make([]metav1alpha1.Feature, 0, len(c.Spec.FeatureGates))
	for _, fg := range c.Spec.FeatureGates {
		fs = append(fs, fg.Name)
	}

	return fs
}

func IsFeatureEnabled(c *v1alpha1.Cluster, f metav1alpha1.Feature) bool {
	for _, fg := range c.Spec.FeatureGates {
		if fg.Name == f {
			return true
		}
	}

	return false
}

// ClusterSubdomain returns the subdomain for all components of the cluster
func ClusterSubdomain(clusterName string) string {
	// add a suffix to avoid svc name conflict
	return clusterName + "-cluster"
}

// ClusterPD returns the pd service of the cluster
func ClusterPD(c *v1alpha1.Cluster) string {
	// add a suffix to avoid svc name conflict
	if c.Spec.CustomizedPDServiceName != nil {
		return *c.Spec.CustomizedPDServiceName
	}
	return c.Name + "-pd"
}

func ClientCertKeyPairSecretName(c *v1alpha1.Cluster) string {
	sec := c.Spec.Security
	if sec != nil && sec.TLS != nil && sec.TLS.Client != nil && sec.TLS.Client.CertKeyPair != nil {
		return sec.TLS.Client.CertKeyPair.Name
	}
	return c.Name + "-cluster-client-secret"
}

func ClientCASecretName(c *v1alpha1.Cluster) string {
	sec := c.Spec.Security
	if sec != nil && sec.TLS != nil && sec.TLS.Client != nil && sec.TLS.Client.CA != nil {
		return sec.TLS.Client.CA.Name
	}
	return c.Name + "-cluster-client-secret"
}

func ClientInsecureSkipTLSVerify(c *v1alpha1.Cluster) bool {
	sec := c.Spec.Security
	if sec != nil && sec.TLS != nil && sec.TLS.Client != nil {
		return sec.TLS.Client.InsecureSkipTLSVerify
	}
	return false
}
