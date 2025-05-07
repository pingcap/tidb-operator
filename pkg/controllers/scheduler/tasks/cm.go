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
	schedulercfg "github.com/pingcap/tidb-operator/pkg/configs/scheduler"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/pkg/utils/toml"
)

func TaskConfigMap(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ConfigMap", func(ctx context.Context) task.Result {
		cfg := schedulercfg.Config{}
		cluster := state.Cluster()
		obj := state.Object()
		decoder, encoder := toml.Codec[schedulercfg.Config]()
		if err := decoder.Decode([]byte(obj.Spec.Config), &cfg); err != nil {
			return task.Fail().With("scheduler config cannot be decoded: %v", err)
		}

		if err := cfg.Overlay(cluster, obj); err != nil {
			return task.Fail().With("cannot generate scheduler config: %v", err)
		}

		data, err := encoder.Encode(&cfg)
		if err != nil {
			return task.Fail().With("scheduler config cannot be encoded: %v", err)
		}

		expected := newConfigMap(obj, data)
		if err := c.Apply(ctx, expected); err != nil {
			return task.Fail().With("can't create/update the cm of scheduler: %v", err)
		}
		return task.Complete().With("cm is synced")
	})
}

func newConfigMap(scheduler *v1alpha1.Scheduler, data []byte) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreutil.PodName[scope.Scheduler](scheduler),
			Namespace: scheduler.Namespace,
			Labels:    coreutil.ConfigMapLabels[scope.Scheduler](scheduler),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(scheduler, v1alpha1.SchemeGroupVersion.WithKind("Scheduler")),
			},
		},
		Data: map[string]string{
			v1alpha1.FileNameConfig: string(data),
		},
	}
}
