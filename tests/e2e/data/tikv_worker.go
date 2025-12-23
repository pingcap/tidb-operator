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
)

func NewTiKVWorkerGroup(ns string, patches ...GroupPatch[*v1alpha1.TiKVWorkerGroup]) *v1alpha1.TiKVWorkerGroup {
	wg := &v1alpha1.TiKVWorkerGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      defaultTiKVWorkerGroupName,
		},
		Spec: v1alpha1.TiKVWorkerGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: defaultClusterName},
			Replicas: ptr.To[int32](1),
			Template: v1alpha1.TiKVWorkerTemplate{
				Spec: v1alpha1.TiKVWorkerTemplateSpec{
					Version: defaultVersion,
					Image:   ptr.To(defaultImageRegistry + "tikv"),
				},
			},
		},
	}
	for _, p := range patches {
		p.Patch(wg)
	}

	return wg
}

func WithTiKVWorkerNextGen() GroupPatch[*v1alpha1.TiKVWorkerGroup] {
	return GroupPatchFunc[*v1alpha1.TiKVWorkerGroup](func(obj *v1alpha1.TiKVWorkerGroup) {
		obj.Spec.Template.Spec.Version = "v9.0.0"
		obj.Spec.Template.Spec.Image = ptr.To(defaultImageRegistry + "tikv:dedicated-next-gen")
		obj.Spec.Template.Spec.Config = `[dfs]
prefix = "tikv"
s3-bucket = "local"
s3-endpoint = "http://minio:9000"
s3-key-id = "test12345678"
s3-secret-key = "test12345678"
s3-region = "local"
`
	})
}
