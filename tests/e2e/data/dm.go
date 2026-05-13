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
)

func NewDMGroup(ns string, patches ...GroupPatch[*v1alpha1.DMGroup]) *v1alpha1.DMGroup {
	dmg := &v1alpha1.DMGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      defaultDMGroupName,
		},
		Spec: v1alpha1.DMGroupSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: defaultClusterName,
			},
			Replicas: ptr.To[int32](1),
			Template: v1alpha1.DMTemplate{
				Spec: v1alpha1.DMTemplateSpec{
					Version: defaultVersion,
					Image:   ptr.To(defaultImageRegistry + "dm"),
					DataVolume: v1alpha1.Volume{
						Name:    "data",
						Mounts:  []v1alpha1.VolumeMount{{Type: v1alpha1.VolumeMountTypeDMData}},
						Storage: resource.MustParse("1Gi"),
					},
				},
			},
		},
	}
	for _, p := range patches {
		p.Patch(dmg)
	}

	return dmg
}
