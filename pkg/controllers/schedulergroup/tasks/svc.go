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
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

// TODO: extract svc to a common task
func TaskService(state common.ObjectState[*v1alpha1.SchedulerGroup], c client.Client) task.Task { // Changed TSOGroup to SchedulerGroup
	return task.NameTaskFunc("Service", func(ctx context.Context) task.Result {
		sg := state.Object() // Changed tg to sg

		svc := newHeadlessService(sg) // Changed tg to sg
		if err := c.Apply(ctx, svc); err != nil {
			return task.Fail().With(fmt.Sprintf("can't create headless service of scheduler: %v", err)) // Changed tso to scheduler
		}

		return task.Complete().With("headless service of scheduler has been applied") // Changed tso to scheduler
	})
}

func newHeadlessService(sg *v1alpha1.SchedulerGroup) *corev1.Service { // Changed tg to sg
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HeadlessServiceName(sg.Name), // Changed tg to sg
			Namespace: sg.Namespace,                 // Changed tg to sg
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentScheduler, // Changed TSO to Scheduler
				v1alpha1.LabelKeyCluster:   sg.Spec.Cluster.Name,                // Changed tg to sg
				v1alpha1.LabelKeyGroup:     sg.Name,                             // Changed tg to sg
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(sg, v1alpha1.SchemeGroupVersion.WithKind("SchedulerGroup")), // Changed tg to sg, TSOGroup to SchedulerGroup
			},
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeClusterIP,
			ClusterIP:      corev1.ClusterIPNone,
			IPFamilyPolicy: &ipFamilyPolicy,
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentScheduler, // Changed TSO to Scheduler
				v1alpha1.LabelKeyCluster:   sg.Spec.Cluster.Name,                // Changed tg to sg
				v1alpha1.LabelKeyGroup:     sg.Name,                             // Changed tg to sg
			},
			Ports: []corev1.ServicePort{
				{
					Name:       v1alpha1.SchedulerPortNameClient,      // TODO: Define SchedulerPortNameClient in api/v2/core/v1alpha1/constants.go (similar to TSOPortNameClient)
					Port:       coreutil.SchedulerGroupClientPort(sg), // TODO: Define SchedulerGroupClientPort in apiutil/core/v1alpha1/util.go (similar to TSOGroupClientPort)
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.SchedulerPortNameClient), // TODO: Use defined SchedulerPortNameClient
				},
			},
			PublishNotReadyAddresses: true,
		},
	}
}
