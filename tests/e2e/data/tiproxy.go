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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

func NewTiProxyGroup(ns string, patches ...GroupPatch[*runtime.TiProxyGroup]) *v1alpha1.TiProxyGroup {
	proxyg := &runtime.TiProxyGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      defaultTiProxyGroupName,
		},
		Spec: v1alpha1.TiProxyGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: defaultClusterName},
			Replicas: ptr.To[int32](1),
			Template: v1alpha1.TiProxyTemplate{
				Spec: v1alpha1.TiProxyTemplateSpec{
					Image: ptr.To(defaultImageRegistry + "tiproxy:main"),
				},
			},
		},
	}

	for _, p := range patches {
		p(proxyg)
	}

	return runtime.ToTiProxyGroup(proxyg)
}
