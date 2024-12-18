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
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
)

type TaskService struct {
	Logger logr.Logger
	Client client.Client
}

func NewTaskService(logger logr.Logger, c client.Client) task.Task[ReconcileContext] {
	return &TaskService{
		Logger: logger,
		Client: c,
	}
}

func (*TaskService) Name() string {
	return "Service"
}

func (t *TaskService) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	if rtx.Cluster.ShouldSuspendCompute() {
		return task.Complete().With("skip service for suspension")
	}

	tidbg := rtx.TiDBGroup

	svcHeadless := newHeadlessService(tidbg)
	if err := t.Client.Apply(ctx, svcHeadless); err != nil {
		return task.Fail().With(fmt.Sprintf("can't create headless service of tidb: %v", err))
	}

	svc := newService(tidbg)
	if err := t.Client.Apply(ctx, svc); err != nil {
		return task.Fail().With(fmt.Sprintf("can't create service of tidb: %v", err))
	}

	return task.Complete().With("service of tidb has been applied")
}

func newHeadlessService(tidbg *v1alpha1.TiDBGroup) *corev1.Service {
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HeadlessServiceName(tidbg.Name),
			Namespace: tidbg.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
				v1alpha1.LabelKeyCluster:   tidbg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     tidbg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tidbg, v1alpha1.SchemeGroupVersion.WithKind("TiDBGroup")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
				v1alpha1.LabelKeyCluster:   tidbg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     tidbg.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       v1alpha1.TiDBPortNameStatus,
					Port:       tidbg.GetStatusPort(),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.TiDBPortNameStatus),
				},
			},
			ClusterIP:                corev1.ClusterIPNone,
			IPFamilyPolicy:           &ipFamilyPolicy,
			PublishNotReadyAddresses: true,
		},
	}
}

func newService(tidbg *v1alpha1.TiDBGroup) *corev1.Service {
	svcType := corev1.ServiceTypeClusterIP
	if tidbg.Spec.Service != nil && tidbg.Spec.Service.Type != "" {
		svcType = tidbg.Spec.Service.Type
	}
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tidbg.Name + "-tidb",
			Namespace: tidbg.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
				v1alpha1.LabelKeyCluster:   tidbg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     tidbg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tidbg, v1alpha1.SchemeGroupVersion.WithKind("TiDBGroup")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
				v1alpha1.LabelKeyCluster:   tidbg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     tidbg.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       v1alpha1.TiDBPortNameClient,
					Port:       tidbg.GetClientPort(),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.TiDBPortNameClient),
				},
				{
					Name:       v1alpha1.TiDBPortNameStatus,
					Port:       tidbg.GetStatusPort(),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.TiDBPortNameStatus),
				},
			},
			Type:           svcType,
			IPFamilyPolicy: &ipFamilyPolicy,
		},
	}
}
