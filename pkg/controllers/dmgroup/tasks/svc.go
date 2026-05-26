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

func TaskService(state State, c client.Client) task.Task {
	return task.NameTaskFunc("Service", func(ctx context.Context) task.Result {
		dmg := state.DMGroup()

		headless := newHeadlessService(dmg)
		if err := c.Apply(ctx, headless); err != nil {
			return task.Fail().With(fmt.Sprintf("can't create headless service of dm-master: %v", err))
		}

		svc := newInternalService(dmg)
		if err := c.Apply(ctx, svc); err != nil {
			return task.Fail().With(fmt.Sprintf("can't create internal service of dm-master: %v", err))
		}

		return task.Complete().With("services of dm-master have been applied")
	})
}

func newHeadlessService(dmg *v1alpha1.DMGroup) *corev1.Service {
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreutil.HeadlessServiceName[scope.DMGroup](dmg),
			Namespace: dmg.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentDMMaster,
				v1alpha1.LabelKeyCluster:   dmg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     dmg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(dmg, v1alpha1.SchemeGroupVersion.WithKind("DMGroup")),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeClusterIP,
			ClusterIP:      corev1.ClusterIPNone,
			IPFamilyPolicy: &ipFamilyPolicy,
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentDMMaster,
				v1alpha1.LabelKeyCluster:   dmg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     dmg.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       v1alpha1.DMPortName,
					Port:       coreutil.DMGroupPort(dmg),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.DMPortName),
				},
				{
					Name:       v1alpha1.DMPeerPortName,
					Port:       coreutil.DMGroupPeerPort(dmg),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.DMPeerPortName),
				},
			},
			PublishNotReadyAddresses: true,
		},
	}
}

func newInternalService(dmg *v1alpha1.DMGroup) *corev1.Service {
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      InternalServiceName(dmg.Name),
			Namespace: dmg.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentDMMaster,
				v1alpha1.LabelKeyCluster:   dmg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     dmg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(dmg, v1alpha1.SchemeGroupVersion.WithKind("DMGroup")),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeClusterIP,
			IPFamilyPolicy: &ipFamilyPolicy,
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentDMMaster,
				v1alpha1.LabelKeyCluster:   dmg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     dmg.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:     v1alpha1.DMPortName,
					Port:     v1alpha1.DefaultDMPort,
					Protocol: corev1.ProtocolTCP,
					// TargetPort by name routes to the pod's named port, keeping this
					// ClusterIP port stable even when spec.server.ports.port is customised.
					TargetPort: intstr.FromString(v1alpha1.DMPortName),
				},
				{
					Name:       v1alpha1.DMPeerPortName,
					Port:       v1alpha1.DefaultDMPeerPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.DMPeerPortName),
				},
			},
		},
	}
}
