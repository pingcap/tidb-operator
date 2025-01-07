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

package tasks

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	tiflashcfg "github.com/pingcap/tidb-operator/pkg/configs/tiflash"
	"github.com/pingcap/tidb-operator/pkg/utils/hasher"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/pkg/utils/toml"
)

func TaskConfigMap(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ConfigMap", func(ctx context.Context) task.Result {
		cfg := tiflashcfg.Config{}
		decoder, encoder := toml.Codec[tiflashcfg.Config]()
		if err := decoder.Decode([]byte(state.TiFlash().Spec.Config), &cfg); err != nil {
			return task.Fail().With("tiflash config cannot be decoded: %w", err)
		}
		if err := cfg.Overlay(state.Cluster(), state.TiFlash()); err != nil {
			return task.Fail().With("cannot generate tiflash config: %w", err)
		}
		flashData, err := encoder.Encode(&cfg)
		if err != nil {
			return task.Fail().With("tiflash config cannot be encoded: %w", err)
		}

		proxyConfig := tiflashcfg.ProxyConfig{}
		decoderProxy, encoderProxy := toml.Codec[tiflashcfg.ProxyConfig]()
		if err = decoderProxy.Decode([]byte(state.TiFlash().Spec.ProxyConfig), &proxyConfig); err != nil {
			return task.Fail().With("tiflash proxy config cannot be decoded: %w", err)
		}
		if err = proxyConfig.Overlay(state.Cluster(), state.TiFlash()); err != nil {
			return task.Fail().With("cannot generate tiflash proxy config: %w", err)
		}
		proxyData, err := encoderProxy.Encode(&proxyConfig)
		if err != nil {
			return task.Fail().With("tiflash proxy config cannot be encoded: %w", err)
		}

		state.ConfigHash, err = hasher.GenerateHash(state.TiFlash().Spec.Config)
		if err != nil {
			return task.Fail().With("failed to generate hash for `tiflash.spec.config`: %w", err)
		}
		expected := newConfigMap(state.TiFlash(), flashData, proxyData, state.ConfigHash)
		if e := c.Apply(ctx, expected); e != nil {
			return task.Fail().With("can't create/update cm of tiflash: %w", e)
		}
		return task.Complete().With("cm is synced")
	})
}

func newConfigMap(tiflash *v1alpha1.TiFlash, flashData, proxyData []byte, hash string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tiflash.PodName(),
			Namespace: tiflash.Namespace,
			Labels: maputil.Merge(tiflash.Labels, map[string]string{
				v1alpha1.LabelKeyInstance:   tiflash.Name,
				v1alpha1.LabelKeyConfigHash: hash,
			}),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tiflash, v1alpha1.SchemeGroupVersion.WithKind("TiFlash")),
			},
		},
		Data: map[string]string{
			v1alpha1.ConfigFileName:             string(flashData),
			v1alpha1.ConfigFileTiFlashProxyName: string(proxyData),
		},
	}
}
