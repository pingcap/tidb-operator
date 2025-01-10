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
	pdcfg "github.com/pingcap/tidb-operator/pkg/configs/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/hasher"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/pkg/utils/toml"
)

func TaskConfigMap(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ConfigMap", func(ctx context.Context) task.Result {
		// TODO: DON'T add bootstrap config back
		// We need to check current config and forbid adding bootstrap cfg back

		cfg := pdcfg.Config{}
		decoder, encoder := toml.Codec[pdcfg.Config]()
		if err := decoder.Decode([]byte(state.PD().Spec.Config), &cfg); err != nil {
			return task.Fail().With("pd config cannot be decoded: %v", err)
		}

		if err := cfg.Overlay(state.Cluster(), state.PD(), state.PDSlice()); err != nil {
			return task.Fail().With("cannot generate pd config: %v", err)
		}

		data, err := encoder.Encode(&cfg)
		if err != nil {
			return task.Fail().With("pd config cannot be encoded: %v", err)
		}

		// TODO(liubo02): avoid decode toml twice
		hash, err := hasher.GenerateHash(state.PD().Spec.Config)
		if err != nil {
			return task.Fail().With("failed to generate hash for `pd.spec.config`: %v", err)
		}
		state.ConfigHash = hash
		expected := newConfigMap(state.PD(), data, state.ConfigHash)
		if err := c.Apply(ctx, expected); err != nil {
			return task.Fail().With("can't create/update the cm of pd: %v", err)
		}
		return task.Complete().With("cm is synced")
	})
}

func newConfigMap(pd *v1alpha1.PD, data []byte, hash string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pd.PodName(),
			Namespace: pd.Namespace,
			Labels: maputil.Merge(pd.Labels, map[string]string{
				v1alpha1.LabelKeyInstance:   pd.Name,
				v1alpha1.LabelKeyConfigHash: hash,
			}),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(pd, v1alpha1.SchemeGroupVersion.WithKind("PD")),
			},
		},
		Data: map[string]string{
			v1alpha1.ConfigFileName: string(data),
		},
	}
}
