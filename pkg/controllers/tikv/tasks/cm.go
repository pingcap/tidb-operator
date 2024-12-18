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
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	tikvcfg "github.com/pingcap/tidb-operator/pkg/configs/tikv"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
	"github.com/pingcap/tidb-operator/pkg/utils/toml"
)

type TaskConfigMap struct {
	Client client.Client
	Logger logr.Logger
}

func NewTaskConfigMap(logger logr.Logger, c client.Client) task.Task[ReconcileContext] {
	return &TaskConfigMap{
		Client: c,
		Logger: logger,
	}
}

func (*TaskConfigMap) Name() string {
	return "ConfigMap"
}

func (t *TaskConfigMap) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	c := tikvcfg.Config{}
	decoder, encoder := toml.Codec[tikvcfg.Config]()
	if err := decoder.Decode([]byte(rtx.TiKV.Spec.Config), &c); err != nil {
		return task.Fail().With("tikv config cannot be decoded: %w", err)
	}
	if err := c.Overlay(rtx.Cluster, rtx.TiKV); err != nil {
		return task.Fail().With("cannot generate tikv config: %w", err)
	}

	data, err := encoder.Encode(&c)
	if err != nil {
		return task.Fail().With("tikv config cannot be encoded: %w", err)
	}

	rtx.ConfigHash, err = toml.GenerateHash(rtx.TiKV.Spec.Config)
	if err != nil {
		return task.Fail().With("failed to generate hash for `tikv.spec.config`: %w", err)
	}
	expected := newConfigMap(rtx.TiKV, data, rtx.ConfigHash)
	if e := t.Client.Apply(rtx, expected); e != nil {
		return task.Fail().With("can't create/update cm of tikv: %w", e)
	}
	return task.Complete().With("cm is synced")
}

func newConfigMap(tikv *v1alpha1.TiKV, data []byte, hash string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tikv.PodName(),
			Namespace: tikv.Namespace,
			Labels: maputil.Merge(tikv.Labels, map[string]string{
				v1alpha1.LabelKeyInstance:   tikv.Name,
				v1alpha1.LabelKeyConfigHash: hash,
			}),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tikv, v1alpha1.SchemeGroupVersion.WithKind("TiKV")),
			},
		},
		Data: map[string]string{
			v1alpha1.ConfigFileName: string(data),
		},
	}
}
