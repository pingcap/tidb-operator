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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

// nolint:staticcheck
// TODO: extract svc to a common task
func TaskService(state State, c client.Client) task.Task {
	return task.NameTaskFunc("Service", func(ctx context.Context) task.Result {
		wg := state.Object()

		headless := newHeadlessService(wg)
		if err := c.Apply(ctx, headless); err != nil {
			return task.Fail().With(fmt.Sprintf("can't create headless service of tikv worker: %v", err))
		}

		internal := newInternalService(wg)
		if err := c.Apply(ctx, internal); err != nil {
			return task.Fail().With(fmt.Sprintf("can't create internal service of tikv worker: %v", err))
		}

		return task.Complete().With("services of tikv worker has been applied")
	})
}

func newHeadlessService(wg *v1alpha1.TiKVWorkerGroup) *corev1.Service {
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreutil.HeadlessServiceName[scope.TiKVWorkerGroup](wg),
			Namespace: wg.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKVWorker,
				v1alpha1.LabelKeyCluster:   wg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     wg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(wg, v1alpha1.SchemeGroupVersion.WithKind("TiKVWorkerGroup")),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeClusterIP,
			ClusterIP:      corev1.ClusterIPNone,
			IPFamilyPolicy: &ipFamilyPolicy,
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKVWorker,
				v1alpha1.LabelKeyCluster:   wg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     wg.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       v1alpha1.TiKVWorkerPortNameAPI,
					Port:       coreutil.TiKVWorkerGroupAPIPort(wg),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.TiKVWorkerPortNameAPI),
				},
			},
			PublishNotReadyAddresses: true,
		},
	}
}

func newInternalService(wg *v1alpha1.TiKVWorkerGroup) *corev1.Service {
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreutil.InternalServiceName[scope.TiKVWorkerGroup](wg),
			Namespace: wg.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKVWorker,
				v1alpha1.LabelKeyCluster:   wg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     wg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(wg, v1alpha1.SchemeGroupVersion.WithKind("TiKVWorkerGroup")),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeClusterIP,
			IPFamilyPolicy: &ipFamilyPolicy,
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKVWorker,
				v1alpha1.LabelKeyCluster:   wg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     wg.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       v1alpha1.TiKVWorkerPortNameAPI,
					Port:       v1alpha1.DefaultTiKVWorkerPortAPI,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.TiKVWorkerPortNameAPI),
				},
			},
		},
	}
}
