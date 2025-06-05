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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alphabr "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	t "github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func condConfigMapExist(rtx *ReconcileContext) t.Condition {
	return t.CondFunc(func() bool {
		return rtx.ConfigMap() != nil
	})
}

func taskCreateConfigMap(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("CreateConfigMap", func(ctx context.Context) t.Result {
		configMap := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      ConfigMapName(rtx.TiBR()),
				Namespace: rtx.NamespacedName().Namespace,
				Labels:    TiBRSubResourceLabels(rtx.TiBR()),
				OwnerReferences: []v1.OwnerReference{
					*v1.NewControllerRef(rtx.TiBR(), v1alphabr.SchemeGroupVersion.WithKind("TiBR")),
				},
			},
			Data: map[string]string{
				ConfigFileName: string(rtx.TiBR().Spec.Config),
			},
		}
		err := rtx.Client().Create(ctx, configMap)
		if err != nil {
			return t.Fail().With("failed to create configmap: %s", err.Error())
		}

		return t.Complete().With("configmap created")
	})
}

func taskDeleteConfigMap(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("DeleteConfigMap", func(ctx context.Context) t.Result {
		configMap := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      ConfigMapName(rtx.TiBR()),
				Namespace: rtx.NamespacedName().Namespace,
			},
		}
		err := rtx.Client().Delete(ctx, configMap)
		if err != nil {
			return t.Fail().With("failed to delete configmap: %s", err.Error())
		}
		return t.Complete().With("configmap deleted")
	})
}
