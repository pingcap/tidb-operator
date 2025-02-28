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

func NewTiCDCGroup(ns string, patches ...GroupPatch[*runtime.TiCDCGroup]) *v1alpha1.TiCDCGroup {
	cg := &runtime.TiCDCGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      defaultTiCDCGroupName,
		},
		Spec: v1alpha1.TiCDCGroupSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: defaultClusterName,
			},
			Replicas: ptr.To[int32](1),
			Template: v1alpha1.TiCDCTemplate{
				Spec: v1alpha1.TiCDCTemplateSpec{
					Version: defaultVersion,
					Image:   ptr.To(defaultImageRegistry + "ticdc"),
				},
			},
		},
	}
	for _, p := range patches {
		p(cg)
	}

	return runtime.ToTiCDCGroup(cg)
}
