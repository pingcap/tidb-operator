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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

func NewPDGroup(ns string, patches ...GroupPatch[*v1alpha1.PDGroup, *runtime.PDGroup]) *v1alpha1.PDGroup {
	pdg := &runtime.PDGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      defaultPDGroupName,
		},
		Spec: v1alpha1.PDGroupSpec{
			Cluster: v1alpha1.ClusterReference{Name: defaultClusterName},
			Version: defaultVersion,
			Template: v1alpha1.PDTemplate{
				Spec: v1alpha1.PDTemplateSpec{
					Volumes: []v1alpha1.Volume{
						{
							Name:    "data",
							Path:    "/var/lib/pd",
							For:     []v1alpha1.VolumeUsage{{Type: "data"}},
							Storage: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}
	for _, p := range patches {
		p(pdg)
	}

	return runtime.ToPDGroup(pdg)
}
