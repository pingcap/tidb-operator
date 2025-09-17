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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

func NewTiKVGroup(ns string, patches ...GroupPatch[*runtime.TiKVGroup]) *v1alpha1.TiKVGroup {
	kvg := &runtime.TiKVGroup{
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
		p(kvg)
	}

	return runtime.ToTiKVGroup(kvg)
}

// TODO: combine with WithTiDBEvenlySpreadPolicy
func WithTiKVEvenlySpreadPolicy() GroupPatch[*runtime.TiKVGroup] {
	return func(obj *runtime.TiKVGroup) {
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

func WithTiKVAPIVersionV2() GroupPatch[*runtime.TiKVGroup] {
	return func(obj *runtime.TiKVGroup) {
		obj.Spec.Template.Spec.Config = `[storage]
api-version = 2
enable-ttl = true
`
	}
}
