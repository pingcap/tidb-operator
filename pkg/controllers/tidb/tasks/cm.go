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
	tidbcfg "github.com/pingcap/tidb-operator/pkg/configs/tidb"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/pkg/utils/toml"
)

func TaskConfigMap(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ConfigMap", func(ctx context.Context) task.Result {
		cfg := tidbcfg.Config{}
		decoder, encoder := toml.Codec[tidbcfg.Config]()
		if err := decoder.Decode([]byte(state.TiDB().Spec.Config), &cfg); err != nil {
			return task.Fail().With("tidb config cannot be decoded: %w", err)
		}
		if err := cfg.Overlay(state.Cluster(), state.TiDB()); err != nil {
			return task.Fail().With("cannot generate tidb config: %w", err)
		}
		// NOTE(liubo02): don't use val in config file to generate pod
		// TODO(liubo02): add a new field in api to control both config file and pod spec
		state.GracefulWaitTimeInSeconds = int64(cfg.GracefulWaitBeforeShutdown)

		data, err := encoder.Encode(&cfg)
		if err != nil {
			return task.Fail().With("tidb config cannot be encoded: %w", err)
		}

		expected := newConfigMap(state.TiDB(), data)
		if e := c.Apply(ctx, expected); e != nil {
			return task.Fail().With("can't create/update cm of tidb: %w", e)
		}
		return task.Complete().With("cm is synced")
	})
}

func newConfigMap(tidb *v1alpha1.TiDB, data []byte) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreutil.PodName[scope.TiDB](tidb),
			Namespace: tidb.Namespace,
			Labels: maputil.Merge(tidb.Labels, map[string]string{
				v1alpha1.LabelKeyInstance: tidb.Name,
			}),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tidb, v1alpha1.SchemeGroupVersion.WithKind("TiDB")),
			},
		},
		Data: map[string]string{
			v1alpha1.FileNameConfig: string(data),
		},
	}
}
