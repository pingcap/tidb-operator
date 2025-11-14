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
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TaskService(state common.TiProxyGroupState, c client.Client) task.Task {
	return task.NameTaskFunc("Service", func(ctx context.Context) task.Result {
		proxyg := state.TiProxyGroup()

		headless := newHeadlessService(proxyg)
		if err := c.Apply(ctx, headless); err != nil {
			return task.Fail().With(fmt.Sprintf("can't create headless service: %v", err))
		}

		svc := newInternalService(proxyg)
		if err := c.Apply(ctx, svc); err != nil {
			return task.Fail().With(fmt.Sprintf("can't create internal service: %v", err))
		}

		return task.Complete().With("services have been applied")
	})
}

func newHeadlessService(proxyg *v1alpha1.TiProxyGroup) *corev1.Service {
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HeadlessServiceName(proxyg.Name),
			Namespace: proxyg.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiProxy,
				v1alpha1.LabelKeyCluster:   proxyg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     proxyg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(proxyg, v1alpha1.SchemeGroupVersion.WithKind("TiProxyGroup")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiProxy,
				v1alpha1.LabelKeyCluster:   proxyg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     proxyg.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       v1alpha1.TiProxyPortNameAPI,
					Port:       coreutil.TiProxyGroupAPIPort(proxyg),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.TiProxyPortNameAPI),
				},
				{
					Name:       v1alpha1.TiProxyPortNamePeer,
					Port:       coreutil.TiProxyGroupPeerPort(proxyg),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.TiProxyPortNamePeer),
				},
			},
			ClusterIP:                corev1.ClusterIPNone,
			IPFamilyPolicy:           &ipFamilyPolicy,
			PublishNotReadyAddresses: true,
		},
	}
}

func newInternalService(proxyg *v1alpha1.TiProxyGroup) *corev1.Service {
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      InternalServiceName(proxyg.Name),
			Namespace: proxyg.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiProxy,
				v1alpha1.LabelKeyCluster:   proxyg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     proxyg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(proxyg, v1alpha1.SchemeGroupVersion.WithKind("TiProxyGroup")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiProxy,
				v1alpha1.LabelKeyCluster:   proxyg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     proxyg.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       v1alpha1.TiProxyPortNameClient,
					Port:       coreutil.TiProxyGroupClientPort(proxyg),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.TiProxyPortNameClient),
				},
				{
					Name:       v1alpha1.TiProxyPortNameAPI,
					Port:       coreutil.TiProxyGroupAPIPort(proxyg),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.TiProxyPortNameAPI),
				},
			},
			Type:           corev1.ServiceTypeClusterIP,
			IPFamilyPolicy: &ipFamilyPolicy,
		},
	}
}
