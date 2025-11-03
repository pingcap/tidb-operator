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
)

func NewTiKVGroup(ns string, patches ...GroupPatch[*v1alpha1.TiKVGroup]) *v1alpha1.TiKVGroup {
	kvg := &v1alpha1.TiKVGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      defaultTiKVGroupName,
		},
		Spec: v1alpha1.TiKVGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: defaultClusterName},
			Replicas: ptr.To[int32](1),
			Template: v1alpha1.TiKVTemplate{
				Spec: v1alpha1.TiKVTemplateSpec{
					Version: defaultVersion,
					Image:   ptr.To(defaultImageRegistry + "tikv"),
					Volumes: []v1alpha1.Volume{
						{
							Name:    "data",
							Mounts:  []v1alpha1.VolumeMount{{Type: "data"}},
							Storage: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}
	for _, p := range patches {
		p.Patch(kvg)
	}

	return kvg
}

// TODO: combine with WithTiDBEvenlySpreadPolicy
func WithTiKVEvenlySpreadPolicy() GroupPatch[*v1alpha1.TiKVGroup] {
	return GroupPatchFunc[*v1alpha1.TiKVGroup](func(obj *v1alpha1.TiKVGroup) {
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
	})
}

func WithTiKVAPIVersionV2() GroupPatch[*v1alpha1.TiKVGroup] {
	return GroupPatchFunc[*v1alpha1.TiKVGroup](func(obj *v1alpha1.TiKVGroup) {
		obj.Spec.Template.Spec.Config = `[storage]
api-version = 2
enable-ttl = true
`
	})
}

func WithTiKVNextGen() GroupPatch[*v1alpha1.TiKVGroup] {
	return GroupPatchFunc[*v1alpha1.TiKVGroup](func(obj *v1alpha1.TiKVGroup) {
		obj.Spec.Template.Spec.Version = "v9.0.0"
		obj.Spec.Template.Spec.Image = ptr.To(defaultImageRegistry + "tikv:dedicated-next-gen")
		obj.Spec.Template.Spec.Config = `[storage]
api-version = 2
enable-ttl = true

[dfs]
prefix = "tikv"
s3-bucket = "local"
s3-endpoint = "http://minio:9000"
s3-key-id = "test12345678"
s3-secret-key = "test12345678"
s3-region = "local"
`
	})
}

func WithTiKVPodAntiAffinity() GroupPatch[*v1alpha1.TiKVGroup] {
	return GroupPatchFunc[*v1alpha1.TiKVGroup](func(obj *v1alpha1.TiKVGroup) {
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

		o.Pod.Spec.Affinity = &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"pingcap.com/component": "tikv",
								"pingcap.com/group":     obj.GetName(),
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		}
	})
}
