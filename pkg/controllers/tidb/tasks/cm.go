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
	tidbcfg "github.com/pingcap/tidb-operator/pkg/configs/tidb"
	"github.com/pingcap/tidb-operator/pkg/utils/hasher"
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

	c := tidbcfg.Config{}
	decoder, encoder := toml.Codec[tidbcfg.Config]()
	if err := decoder.Decode([]byte(rtx.TiDB.Spec.Config), &c); err != nil {
		return task.Fail().With("tidb config cannot be decoded: %w", err)
	}
	if err := c.Overlay(rtx.Cluster, rtx.TiDB); err != nil {
		return task.Fail().With("cannot generate tidb config: %w", err)
	}
	rtx.GracefulWaitTimeInSeconds = int64(c.GracefulWaitBeforeShutdown)

	data, err := encoder.Encode(&c)
	if err != nil {
		return task.Fail().With("tidb config cannot be encoded: %w", err)
	}

	rtx.ConfigHash, err = hasher.GenerateHash(rtx.TiDB.Spec.Config)
	if err != nil {
		return task.Fail().With("failed to generate hash for `tidb.spec.config`: %w", err)
	}
	expected := newConfigMap(rtx.TiDB, data, rtx.ConfigHash)
	if e := t.Client.Apply(rtx, expected); e != nil {
		return task.Fail().With("can't create/update cm of tidb: %w", e)
	}
	return task.Complete().With("cm is synced")
}

func newConfigMap(tidb *v1alpha1.TiDB, data []byte, hash string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tidb.PodName(),
			Namespace: tidb.Namespace,
			Labels: maputil.Merge(tidb.Labels, map[string]string{
				v1alpha1.LabelKeyInstance:   tidb.Name,
				v1alpha1.LabelKeyConfigHash: hash,
			}),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tidb, v1alpha1.SchemeGroupVersion.WithKind("TiDB")),
			},
		},
		Data: map[string]string{
			v1alpha1.ConfigFileName: string(data),
		},
	}
}
