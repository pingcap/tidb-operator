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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/features"
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

	fg := features.NewFromFeatures(coreutil.EnabledFeatures(rtx.Cluster))
	if !fg.Enabled(metav1alpha1.ClusterSubdomain) {
		return task.Complete().With("cluster subdomain feature is not enabled, skip headless svc creation")
	}

	headless := newHeadlessService(rtx.Cluster)
	if err := t.Client.Apply(ctx, headless); err != nil {
		return task.Fail().With(fmt.Sprintf("can't create headless service of cluster: %v", err))
	}

	return task.Complete().With("service of the cluster has been applied")
}

func newHeadlessService(c *v1alpha1.Cluster) *corev1.Service {
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreutil.ClusterSubdomain(c.Name),
			Namespace: c.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyCluster:   c.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(c, v1alpha1.SchemeGroupVersion.WithKind("Cluster")),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeClusterIP,
			ClusterIP:      corev1.ClusterIPNone,
			IPFamilyPolicy: &ipFamilyPolicy,
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyCluster:   c.Name,
			},
			PublishNotReadyAddresses: true,
		},
	}
}
