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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

const (
	BootstrapSQLName = "bootstrap-sql"
)

func NewCluster(namespace string, patches ...ClusterPatch) *v1alpha1.Cluster {
	c := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      defaultClusterName,
		},
		Spec: v1alpha1.ClusterSpec{
			UpgradePolicy: v1alpha1.UpgradePolicyDefault,
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

func WithClusterTLS() ClusterPatch {
	return func(obj *v1alpha1.Cluster) {
		obj.Spec.TLSCluster = &v1alpha1.TLSCluster{
			Enabled: true,
		}
	}
}
