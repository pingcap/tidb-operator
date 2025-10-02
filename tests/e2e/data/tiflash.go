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
	"github.com/pingcap/tidb-operator/pkg/utils/toml"
)

func NewTiFlashGroup(ns string, patches ...GroupPatch[*runtime.TiFlashGroup]) *v1alpha1.TiFlashGroup {
	flashg := &runtime.TiFlashGroup{
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
		p(flashg)
	}

	return runtime.ToTiFlashGroup(flashg)
}

func WithTiFlashNextGen() GroupPatch[*runtime.TiFlashGroup] {
	return func(obj *runtime.TiFlashGroup) {
		obj.Spec.Template.Spec.Version = "v9.0.0"
		obj.Spec.Template.Spec.Image = ptr.To(defaultImageRegistry + "tiflash:master-next-gen")
		obj.Spec.Template.Spec.ProxyConfig = `[storage]
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
		obj.Spec.Template.Spec.Config = `[storage]
api_version = 2

[flash]
graceful_wait_shutdown_timeout = 300

[storage.s3]
endpoint = "http://minio:9000"
bucket = "local"
root = "/tiflash"
access_key_id = "test12345678"
secret_access_key = "test12345678"
`
		obj.Spec.Template.Spec.Overlay = &v1alpha1.Overlay{
			Pod: &v1alpha1.PodOverlay{
				Spec: &corev1.PodSpec{
					TerminationGracePeriodSeconds: ptr.To[int64](300),
				},
			},
		}
	}
}

type TiFlashConfig struct {
	Flash FlashConfig `toml:"flash"`
}

type FlashConfig struct {
	DisaggregatedMode string `toml:"disaggregated_mode"`
}

func withTiFlashMode(mode string) GroupPatch[*runtime.TiFlashGroup] {
	return func(obj *runtime.TiFlashGroup) {
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
	}
}

func WithTiFlashComputeMode() GroupPatch[*runtime.TiFlashGroup] {
	return withTiFlashMode("tiflash_compute")
}

func WithTiFlashWriteMode() GroupPatch[*runtime.TiFlashGroup] {
	return withTiFlashMode("tiflash_write")
}
