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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
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
			Replicas: ptr.To[int32](1),
			Template: v1alpha1.TiDBTemplate{
				Spec: v1alpha1.TiDBTemplateSpec{
					Version: defaultVersion,
					Image:   ptr.To(defaultImageRegistry + "tidb"),
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

func WithMySQLTLS(ca, certKeyPair string) GroupPatch[*runtime.TiDBGroup] {
	return func(obj *runtime.TiDBGroup) {
		if obj.Spec.Template.Spec.Security == nil {
			obj.Spec.Template.Spec.Security = &v1alpha1.TiDBSecurity{}
		}

		obj.Spec.Template.Spec.Security.TLS = &v1alpha1.TiDBTLS{
			MySQL: &v1alpha1.TLS{
				Enabled: true,
				TLSSecret: v1alpha1.TLSSecret{
					CA: &v1alpha1.CAReference{
						Name: ca,
					},
					CertKeyPair: &v1alpha1.CertKeyPairReference{
						Name: certKeyPair,
					},
				},
			},
		}
	}
}

func WithHotReloadPolicy() GroupPatch[*runtime.TiDBGroup] {
	return func(obj *runtime.TiDBGroup) {
		obj.Spec.Template.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyHotReload
	}
}

func WithEphemeralVolume() GroupPatch[*runtime.TiDBGroup] {
	return func(obj *runtime.TiDBGroup) {
		if obj.Spec.Template.Spec.Overlay == nil {
			obj.Spec.Template.Spec.Overlay = &v1alpha1.Overlay{}
		}
		o := obj.Spec.Template.Spec.Overlay
		if o.Pod == nil {
			o.Pod = &v1alpha1.PodOverlay{}
		}

		if o.Pod.Spec == nil {
			o.Pod.Spec = &corev1.PodSpec{}
		}

		o.Pod.Spec.Volumes = append(o.Pod.Spec.Volumes, corev1.Volume{
			Name: "ephemeral-vol-test",
			VolumeSource: corev1.VolumeSource{
				Ephemeral: &corev1.EphemeralVolumeSource{
					VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
		})
	}
}

// TODO: combine with WithTiKVEvenlySpreadPolicy
func WithTiDBEvenlySpreadPolicy() GroupPatch[*runtime.TiDBGroup] {
	return func(obj *runtime.TiDBGroup) {
		obj.Spec.SchedulePolicies = append(obj.Spec.SchedulePolicies, v1alpha1.SchedulePolicy{
			Type: v1alpha1.SchedulePolicyTypeEvenlySpread,
			EvenlySpread: &v1alpha1.SchedulePolicyEvenlySpread{
				Topologies: []v1alpha1.ScheduleTopology{
					{
						Topology: v1alpha1.Topology{
							"zone": "zone-a",
						},
					},
					{
						Topology: v1alpha1.Topology{
							"zone": "zone-b",
						},
					},
					{
						Topology: v1alpha1.Topology{
							"zone": "zone-c",
						},
					},
				},
			},
		})
	}
}

func WithTiDBStandbyMode() GroupPatch[*runtime.TiDBGroup] {
	return func(obj *runtime.TiDBGroup) {
		obj.Spec.Template.Spec.Mode = v1alpha1.TiDBModeStandBy
	}
}

func WithKeyspace(keyspace string) GroupPatch[*runtime.TiDBGroup] {
	return func(obj *runtime.TiDBGroup) {
		obj.Spec.Template.Spec.Keyspace = keyspace
	}
}

func WithTiDBCommandProbe() GroupPatch[*runtime.TiDBGroup] {
	return func(obj *runtime.TiDBGroup) {
		obj.Spec.Template.Spec.Probes = v1alpha1.TiDBProbes{
			Readiness: &v1alpha1.TiDBProb{
				Type: ptr.To(v1alpha1.CommandProbeType),
			},
		}
	}
}
