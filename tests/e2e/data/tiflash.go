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
	"github.com/pingcap/tidb-operator/pkg/utils/toml"
)

func NewTiFlashGroup(ns string, patches ...GroupPatch[*v1alpha1.TiFlashGroup]) *v1alpha1.TiFlashGroup {
	flashg := &v1alpha1.TiFlashGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      defaultTiFlashGroupName,
		},
		Spec: v1alpha1.TiFlashGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: defaultClusterName},
			Replicas: ptr.To[int32](1),
			Template: v1alpha1.TiFlashTemplate{
				Spec: v1alpha1.TiFlashTemplateSpec{
					Version: defaultVersion,
					Image:   ptr.To(defaultImageRegistry + "tiflash"),
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
		p.Patch(flashg)
	}

	return flashg
}

type TiFlashConfig struct {
	Flash FlashConfig `toml:"flash"`
}

type FlashConfig struct {
	DisaggregatedMode string `toml:"disaggregated_mode"`
}

func withTiFlashMode(mode string) GroupPatch[*v1alpha1.TiFlashGroup] {
	return GroupPatchFunc[*v1alpha1.TiFlashGroup](func(obj *v1alpha1.TiFlashGroup) {
		if mode == "tiflash_write" {
			obj.Name = obj.Name + "-wn"
		}
		if mode == "tiflash_compute" {
			obj.Name = obj.Name + "-cn"
		}
		c := TiFlashConfig{}
		d, e := toml.Codec[TiFlashConfig]()
		if err := d.Decode([]byte(obj.Spec.Template.Spec.Config), &c); err != nil {
			panic("decode tiflash config: " + err.Error())
		}

		c.Flash.DisaggregatedMode = mode
		data, err := e.Encode(&c)
		if err != nil {
			panic("encode tiflash config: " + err.Error())
		}
		obj.Spec.Template.Spec.Config = v1alpha1.ConfigFile(data)
	})
}

func WithTiFlashComputeMode() GroupPatch[*v1alpha1.TiFlashGroup] {
	return withTiFlashMode("tiflash_compute")
}

func WithTiFlashWriteMode() GroupPatch[*v1alpha1.TiFlashGroup] {
	return withTiFlashMode("tiflash_write")
}
