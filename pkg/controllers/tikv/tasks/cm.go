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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	tikvcfg "github.com/pingcap/tidb-operator/pkg/configs/tikv"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/pkg/utils/toml"
)

func TaskConfigMap(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ConfigMap", func(ctx context.Context) task.Result {
		cfg := tikvcfg.Config{}
		decoder, encoder := toml.Codec[tikvcfg.Config]()
		if err := decoder.Decode([]byte(state.TiKV().Spec.Config), &cfg); err != nil {
			return task.Fail().With("tikv config cannot be decoded: %w", err)
		}
		if err := cfg.Overlay(state.Cluster(), state.TiKV()); err != nil {
			return task.Fail().With("cannot generate tikv config: %w", err)
		}

		data, err := encoder.Encode(&cfg)
		if err != nil {
			return task.Fail().With("tikv config cannot be encoded: %w", err)
		}

		expected := newConfigMap(state.TiKV(), data)
		if e := c.Apply(ctx, expected); e != nil {
			return task.Fail().With("can't create/update cm of tikv: %w", e)
		}
		return task.Complete().With("cm is synced")
	})
}

func newConfigMap(tikv *v1alpha1.TiKV, data []byte) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreutil.PodName[scope.TiKV](tikv),
			Namespace: tikv.Namespace,
			Labels:    coreutil.ConfigMapLabels[scope.TiKV](tikv),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tikv, v1alpha1.SchemeGroupVersion.WithKind("TiKV")),
			},
		},
		Data: map[string]string{
			v1alpha1.FileNameConfig: string(data),
		},
	}
}
