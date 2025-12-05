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

package data

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

const (
	BootstrapSQLName = "bootstrap-sql"
	SEMConfigName    = "sem-config"
)

func NewCluster(namespace string, patches ...ClusterPatch) *v1alpha1.Cluster {
	c := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      defaultClusterName,
		},
		Spec: v1alpha1.ClusterSpec{
			UpgradePolicy: v1alpha1.UpgradePolicyDefault,
			FeatureGates: []metav1alpha1.FeatureGate{
				{
					// Disable PD Probe by default in e2e
					Name: metav1alpha1.DisablePDDefaultReadinessProbe,
				},
			},
		},
	}

	for _, p := range patches {
		p(c)
	}

	return c
}

func WithBootstrapSQL() ClusterPatch {
	return func(obj *v1alpha1.Cluster) {
		obj.Spec.BootstrapSQL = &corev1.LocalObjectReference{
			Name: BootstrapSQLName,
		}
	}
}

func WithClusterTLSEnabled() ClusterPatch {
	return func(obj *v1alpha1.Cluster) {
		obj.Spec.TLSCluster = &v1alpha1.TLSCluster{
			Enabled: true,
		}
	}
}

func WithClusterName(name string) ClusterPatch {
	return func(obj *v1alpha1.Cluster) {
		obj.Name = name
	}
}

func WithCustomizedPDServiceName(name string) ClusterPatch {
	return func(obj *v1alpha1.Cluster) {
		obj.Spec.CustomizedPDServiceName = ptr.To(name)
	}
}

func WithFeatureGates(featureGates ...metav1alpha1.Feature) ClusterPatch {
	return func(c *v1alpha1.Cluster) {
		// Clear existing feature gates and add new ones
		c.Spec.FeatureGates = nil
		for _, fg := range featureGates {
			c.Spec.FeatureGates = append(c.Spec.FeatureGates, metav1alpha1.FeatureGate{
				Name: fg,
			})
		}
	}
}

func WithClusterTLSAndTiProxyConfig() ClusterPatch {
	return func(obj *v1alpha1.Cluster) {
		obj.Spec.TLSCluster = &v1alpha1.TLSCluster{
			Enabled: true,
		}
		// Add SessionTokenSigning feature gate
		obj.Spec.FeatureGates = append(obj.Spec.FeatureGates, metav1alpha1.FeatureGate{
			Name: metav1alpha1.SessionTokenSigning,
		})
		// SessionTokenSigning to use cluster TLS secret
		obj.Spec.Security = &v1alpha1.ClusterSecurity{
			SessionTokenSigningCertKeyPair: &corev1.LocalObjectReference{
				Name: "dbg-tidb-cluster-secret", // Default TiDB cluster secret name
			},
		}
	}
}
