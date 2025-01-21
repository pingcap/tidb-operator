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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

const (
	JWKsSecretName = "jwks-secret"
)

func NewTiDBGroup(ns string, patches ...GroupPatch[*runtime.TiDBGroup]) *v1alpha1.TiDBGroup {
	kvg := &runtime.TiDBGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      defaultTiDBGroupName,
		},
		Spec: v1alpha1.TiDBGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: defaultClusterName},
			Version:  defaultVersion,
			Replicas: ptr.To[int32](1),
			Template: v1alpha1.TiDBTemplate{
				Spec: v1alpha1.TiDBTemplateSpec{
					Image: ptr.To(defaultImageRegistry + "tidb"),
					SlowLog: &v1alpha1.TiDBSlowLog{
						Image: ptr.To(defaultHelperImage),
					},
				},
			},
		},
	}
	for _, p := range patches {
		p(kvg)
	}

	return runtime.ToTiDBGroup(kvg)
}

func WithAuthToken() GroupPatch[*runtime.TiDBGroup] {
	return func(obj *runtime.TiDBGroup) {
		if obj.Spec.Template.Spec.Security == nil {
			obj.Spec.Template.Spec.Security = &v1alpha1.TiDBSecurity{}
		}

		obj.Spec.Template.Spec.Security.AuthToken = &v1alpha1.TiDBAuthToken{
			JWKs: corev1.LocalObjectReference{
				Name: JWKsSecretName,
			},
		}
	}
}

func WithTLS() GroupPatch[*runtime.TiDBGroup] {
	return func(obj *runtime.TiDBGroup) {
		if obj.Spec.Template.Spec.Security == nil {
			obj.Spec.Template.Spec.Security = &v1alpha1.TiDBSecurity{}
		}

		obj.Spec.Template.Spec.Security.TLS = &v1alpha1.TiDBTLS{
			MySQL: &v1alpha1.TLS{
				Enabled: true,
			},
		}
	}
}
