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
	"k8s.io/apimachinery/pkg/util/intstr"

	v1alphabr "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	t "github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func condHeadlessSvcExist(rtx *ReconcileContext) t.Condition {
	return t.CondFunc(func() bool {
		return rtx.HeadlessSvc() != nil
	})
}

func taskCreateHeadlessSvc(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("CreateHeadlessSvc", func(ctx context.Context) t.Result {
		svc := &corev1.Service{
			ObjectMeta: v1.ObjectMeta{
				Name:      HeadlessSvcName(rtx.TiBR()),
				Namespace: rtx.NamespacedName().Namespace,
				Labels:    TiBRSubResourceLabels(rtx.TiBR()),
				OwnerReferences: []v1.OwnerReference{
					*v1.NewControllerRef(rtx.TiBR(), v1alphabr.SchemeGroupVersion.WithKind("TiBR")),
				},
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "None",
				Selector:  TiBRSubResourceLabels(rtx.TiBR()),
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Port:       80,
						TargetPort: intstr.FromInt32(APIServerPort),
					},
				},
			},
		}
		err := rtx.Client().Create(ctx, svc)
		if err != nil {
			return t.Fail().With("failed to create headless svc: %s", err.Error())
		}
		return t.Complete().With("headless svc created")
	})
}

func taskDeleteHeadlessSvc(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("DeleteHeadlessSvc", func(ctx context.Context) t.Result {
		svc := &corev1.Service{
			ObjectMeta: v1.ObjectMeta{
				Name:      HeadlessSvcName(rtx.TiBR()),
				Namespace: rtx.NamespacedName().Namespace,
			},
		}
		err := rtx.Client().Delete(ctx, svc)
		if err != nil {
			return t.Fail().With("failed to delete headless svc: %s", err.Error())
		}
		return t.Complete().With("headless svc deleted")
	})
}
