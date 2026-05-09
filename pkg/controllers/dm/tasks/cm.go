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
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	dmcfg "github.com/pingcap/tidb-operator/v2/pkg/configs/dm"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/toml"
)

func TaskConfigMap(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ConfigMap", func(ctx context.Context) task.Result {
		cfg := dmcfg.Config{}
		decoder, encoder := toml.Codec[dmcfg.Config]()
		if err := decoder.Decode([]byte(state.DM().Spec.Config), &cfg); err != nil {
			return task.Fail().With("dm config cannot be decoded: %v", err)
		}

		joinAddr := coreutil.ServiceHost(state.Cluster(), state.DM().Spec.Subdomain) +
			":" + strconv.Itoa(int(coreutil.DMPort(state.DM())))

		if err := cfg.Overlay(state.Cluster(), state.DM(), state.DMSlice(), joinAddr); err != nil {
			return task.Fail().With("cannot generate dm config: %v", err)
		}

		data, err := encoder.Encode(&cfg)
		if err != nil {
			return task.Fail().With("dm config cannot be encoded: %v", err)
		}

		expected := newConfigMap(state.DM(), data)
		if err := c.Apply(ctx, expected); err != nil {
			return task.Fail().With("can't create/update the cm of dm: %v", err)
		}
		return task.Complete().With("cm is synced")
	})
}

func newConfigMap(dm *v1alpha1.DM, data []byte) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreutil.PodName[scope.DM](dm),
			Namespace: dm.Namespace,
			Labels:    coreutil.ConfigMapLabels[scope.DM](dm),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(dm, v1alpha1.SchemeGroupVersion.WithKind("DM")),
			},
		},
		Data: map[string]string{
			v1alpha1.FileNameConfig: string(data),
		},
	}
}
