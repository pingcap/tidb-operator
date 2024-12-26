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
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskService(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("Service", func(ctx context.Context) task.Result {
		pdg := state.PDGroup()

		headless := newHeadlessService(pdg)
		if err := c.Apply(ctx, headless); err != nil {
			return task.Fail().With(fmt.Sprintf("can't create headless service of pd: %v", err))
		}

		svc := newInternalService(pdg)
		if err := c.Apply(ctx, svc); err != nil {
			return task.Fail().With(fmt.Sprintf("can't create internal service of pd: %v", err))
		}

		return task.Complete().With("services of pd have been applied")
	})
}

func newHeadlessService(pdg *v1alpha1.PDGroup) *corev1.Service {
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HeadlessServiceName(pdg.Name),
			Namespace: pdg.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
				v1alpha1.LabelKeyCluster:   pdg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     pdg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(pdg, v1alpha1.SchemeGroupVersion.WithKind("PDGroup")),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeClusterIP,
			ClusterIP:      corev1.ClusterIPNone,
			IPFamilyPolicy: &ipFamilyPolicy,
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
				v1alpha1.LabelKeyCluster:   pdg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     pdg.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       v1alpha1.PDPortNameClient,
					Port:       pdg.GetClientPort(),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.PDPortNameClient),
				},
				{
					Name:       v1alpha1.PDPortNamePeer,
					Port:       pdg.GetPeerPort(),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.PDPortNamePeer),
				},
			},
			PublishNotReadyAddresses: true,
		},
	}
}

func newInternalService(pdg *v1alpha1.PDGroup) *corev1.Service {
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-pd", pdg.Name),
			Namespace: pdg.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
				v1alpha1.LabelKeyCluster:   pdg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     pdg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(pdg, v1alpha1.SchemeGroupVersion.WithKind("PDGroup")),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeClusterIP,
			IPFamilyPolicy: &ipFamilyPolicy,
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
				v1alpha1.LabelKeyCluster:   pdg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     pdg.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       v1alpha1.PDPortNameClient,
					Port:       pdg.GetClientPort(),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.PDPortNameClient),
				},
				{
					Name:       v1alpha1.PDPortNamePeer,
					Port:       pdg.GetPeerPort(),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.PDPortNamePeer),
				},
			},
		},
	}
}
