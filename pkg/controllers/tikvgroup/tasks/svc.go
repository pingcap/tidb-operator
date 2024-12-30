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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

// TODO: extract svc to a common task
func TaskService(state common.TiKVGroupState, c client.Client) task.Task {
	return task.NameTaskFunc("Service", func(ctx context.Context) task.Result {
		kvg := state.TiKVGroup()

		svc := newHeadlessService(kvg)
		if err := c.Apply(ctx, svc); err != nil {
			return task.Fail().With(fmt.Sprintf("can't create headless service of tikv: %v", err))
		}

		return task.Complete().With("headless service of tikv has been applied")
	})
}

func newHeadlessService(kvg *v1alpha1.TiKVGroup) *corev1.Service {
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HeadlessServiceName(kvg.Name),
			Namespace: kvg.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
				v1alpha1.LabelKeyCluster:   kvg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     kvg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(kvg, v1alpha1.SchemeGroupVersion.WithKind("TiKVGroup")),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeClusterIP,
			ClusterIP:      corev1.ClusterIPNone,
			IPFamilyPolicy: &ipFamilyPolicy,
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
				v1alpha1.LabelKeyCluster:   kvg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     kvg.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       v1alpha1.TiKVPortNameClient,
					Port:       kvg.GetClientPort(),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.TiKVPortNameClient),
				},
				{
					Name:       v1alpha1.TiKVPortNameStatus,
					Port:       kvg.GetStatusPort(),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.TiKVPortNameStatus),
				},
			},
			PublishNotReadyAddresses: true,
		},
	}
}
