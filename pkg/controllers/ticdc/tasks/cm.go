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
	ticdccfg "github.com/pingcap/tidb-operator/pkg/configs/ticdc"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/pkg/utils/toml"
)

func TaskConfigMap(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ConfigMap", func(ctx context.Context) task.Result {
		cfg := ticdccfg.Config{}
		decoder, encoder := toml.Codec[ticdccfg.Config]()
		if err := decoder.Decode([]byte(state.TiCDC().Spec.Config), &cfg); err != nil {
			return task.Fail().With("ticdc config cannot be decoded: %w", err)
		}
		if err := cfg.Overlay(state.Cluster(), state.TiCDC()); err != nil {
			return task.Fail().With("cannot generate ticdc config: %w", err)
		}

		data, err := encoder.Encode(&cfg)
		if err != nil {
			return task.Fail().With("ticdc config cannot be encoded: %w", err)
		}

		expected := newConfigMap(state.TiCDC(), data)
		if e := c.Apply(ctx, expected); e != nil {
			return task.Fail().With("can't create/update cm of ticdc: %w", e)
		}
		return task.Complete().With("cm is synced")
	})
}

func newConfigMap(ticdc *v1alpha1.TiCDC, data []byte) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreutil.PodName[scope.TiCDC](ticdc),
			Namespace: ticdc.Namespace,
			Labels:    coreutil.ConfigMapLabels[scope.TiCDC](ticdc),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ticdc, v1alpha1.SchemeGroupVersion.WithKind("TiCDC")),
			},
		},
		Data: map[string]string{
			v1alpha1.FileNameConfig: string(data),
		},
	}
}
