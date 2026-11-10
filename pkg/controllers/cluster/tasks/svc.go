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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task"
)

type TaskService struct {
	APIReader ctrlclient.Reader
	Logger    logr.Logger
	Client    client.Client
}

func NewTaskService(logger logr.Logger, c client.Client, apiReader ctrlclient.Reader) task.Task[ReconcileContext] {
	return &TaskService{
		APIReader: apiReader,
		Logger:    logger,
		Client:    c,
	}
}

func (*TaskService) Name() string {
	return "Service"
}

func (t *TaskService) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	fg := features.NewFromCluster(rtx.Cluster.Spec.FeatureGates)
	if fg.Enabled(metav1alpha1.ClusterSubdomain) {
		headless := newHeadlessService(rtx.Cluster)
		if err := t.Client.Apply(ctx, headless); err != nil {
			return task.Fail().With("%s", fmt.Sprintf("can't create headless service of cluster: %v", err))
		}
	}
	if fg.Enabled(metav1alpha1.MultiPDGroup) {
		pd := newPDService(rtx.Cluster)
		if err := common.ReconcilePDService(ctx, t.Client, t.APIReader, pd, rtx.PDGroups...); err != nil {
			return task.Fail().With("can't reconcile pd service of cluster: %v", err)
		}
	} else if group := singleActivePDGroup(rtx.PDGroups); group != nil {
		if err := t.returnPDGroupService(ctx, rtx.Cluster, group); err != nil {
			return task.Fail().With("can't return pd service to pd group: %v", err)
		}
	}

	return task.Complete().With("service of the cluster has been applied")
}

func singleActivePDGroup(groups []*v1alpha1.PDGroup) *v1alpha1.PDGroup {
	var active *v1alpha1.PDGroup
	for _, group := range groups {
		if !group.DeletionTimestamp.IsZero() {
			continue
		}
		if active != nil {
			return nil
		}
		active = group
	}
	return active
}

// The Cluster controller handles both directions of the transfer. A PDGroup
// reconcile that started before MultiPDGroup was enabled cannot take it back.
func (t *TaskService) returnPDGroupService(ctx context.Context, cluster *v1alpha1.Cluster, group *v1alpha1.PDGroup) error {
	key := client.ObjectKey{Namespace: cluster.Namespace, Name: coreutil.InternalServiceName[scope.PDGroup](group)}
	current, err := common.GetPDService(ctx, t.Client, t.APIReader, key)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	if !metav1.IsControlledBy(current, cluster) || !current.DeletionTimestamp.IsZero() ||
		current.Spec.Type != corev1.ServiceTypeClusterIP || current.Spec.ClusterIP == corev1.ClusterIPNone {
		return nil
	}

	updated := current.DeepCopy()
	for i := range updated.OwnerReferences {
		if updated.OwnerReferences[i].UID == cluster.UID {
			updated.OwnerReferences[i] = *metav1.NewControllerRef(group, v1alpha1.SchemeGroupVersion.WithKind("PDGroup"))
			break
		}
	}
	patch := ctrlclient.MergeFromWithOptions(current, ctrlclient.MergeFromWithOptimisticLock{})
	return t.Client.Patch(ctx, updated, patch, ctrlclient.FieldOwner(common.PDServiceUpdateManager))
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

func newPDService(c *v1alpha1.Cluster) *corev1.Service {
	ipFamilyPolicy := corev1.IPFamilyPolicyPreferDualStack
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreutil.ClusterPD(c),
			Namespace: c.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
				v1alpha1.LabelKeyCluster:   c.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(c, v1alpha1.SchemeGroupVersion.WithKind("Cluster")),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeClusterIP,
			IPFamilyPolicy: &ipFamilyPolicy,
			Selector: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
				v1alpha1.LabelKeyCluster:   c.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       v1alpha1.PDPortNameClient,
					Port:       v1alpha1.DefaultPDPortClient,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(v1alpha1.PDPortNameClient),
				},
			},
		},
	}
}
