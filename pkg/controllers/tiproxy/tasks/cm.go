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
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	tiproxycfg "github.com/pingcap/tidb-operator/v2/pkg/configs/tiproxy"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/toml"
)

func TaskConfigMap(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ConfigMap", func(ctx context.Context) task.Result {
		cfg := tiproxycfg.Config{}
		decoder, encoder := toml.Codec[tiproxycfg.Config]()
		if err := decoder.Decode([]byte(state.TiProxy().Spec.Config), &cfg); err != nil {
			return task.Fail().With("tiproxy config cannot be decoded: %w", err)
		}
		if err := cfg.Overlay(state.Cluster(), state.TiProxy()); err != nil {
			return task.Fail().With("cannot generate tiproxy config: %w", err)
		}

		data, err := encoder.Encode(&cfg)
		if err != nil {
			return task.Fail().With("tiproxy config cannot be encoded: %w", err)
		}

		expected := newConfigMap(state.TiProxy(), data)
		if e := c.Apply(ctx, expected); e != nil {
			return task.Fail().With("can't create/update cm of tiproxy: %w", e)
		}
		return task.Complete().With("cm is synced")
	})
}

func newConfigMap(tiproxy *v1alpha1.TiProxy, data []byte) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreutil.PodName[scope.TiProxy](tiproxy),
			Namespace: tiproxy.Namespace,
			Labels:    coreutil.ConfigMapLabels[scope.TiProxy](tiproxy),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tiproxy, v1alpha1.SchemeGroupVersion.WithKind("TiProxy")),
			},
		},
		Data: map[string]string{
			v1alpha1.FileNameConfig: string(data),
		},
	}
}
