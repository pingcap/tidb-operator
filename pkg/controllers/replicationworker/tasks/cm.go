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
	"github.com/pingcap/tidb-operator/v2/pkg/configs/replicationworker"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/toml"
)

func TaskConfigMap(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ConfigMap", func(ctx context.Context) task.Result {
		cfg := replicationworker.Config{}
		cluster := state.Cluster()
		obj := state.Object()
		decoder, encoder := toml.Codec[replicationworker.Config]()
		if err := decoder.Decode([]byte(obj.Spec.Config), &cfg); err != nil {
			return task.Fail().With("ReplicationWorker config cannot be decoded: %v", err)
		}

		if err := cfg.Overlay(cluster, obj); err != nil {
			return task.Fail().With("cannot generate ReplicationWorker config: %v", err)
		}

		data, err := encoder.Encode(&cfg)
		if err != nil {
			return task.Fail().With("ReplicationWorker config cannot be encoded: %v", err)
		}

		expected := newConfigMap(obj, data)
		if err := c.Apply(ctx, expected); err != nil {
			return task.Fail().With("can't create/update the cm of ReplicationWorker: %v", err)
		}
		return task.Complete().With("cm is synced")
	})
}

func newConfigMap(rw *v1alpha1.ReplicationWorker, data []byte) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreutil.PodName[scope.ReplicationWorker](rw),
			Namespace: rw.Namespace,
			Labels:    coreutil.ConfigMapLabels[scope.ReplicationWorker](rw),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(rw, v1alpha1.SchemeGroupVersion.WithKind("ReplicationWorker")),
			},
		},
		Data: map[string]string{
			v1alpha1.FileNameConfig: string(data),
		},
	}
}
