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

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TaskRBAC(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("RBAC", func(ctx context.Context) task.Result {
		fg := state.FeatureGates()
		if !fg.Enabled(metav1alpha1.TiCDCDynamicSecretSyncer) {
			return task.Complete().With("feature TiCDCDynamicSecretSyncer is not enabled")
		}

		cg := state.TiCDCGroup()

		role := newRole(cg)
		rb := newRoleBinding(cg)
		if err := c.Apply(ctx, role); err != nil {
			return task.Fail().With(fmt.Sprintf("can't create role: %v", err))
		}
		if err := c.Apply(ctx, rb); err != nil {
			return task.Fail().With(fmt.Sprintf("can't create role binding: %v", err))
		}

		return task.Complete().With("rbac config has been applied")
	})
}

func newRole(cg *v1alpha1.TiCDCGroup) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RBACName(cg.Name),
			Namespace: cg.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiCDC,
				v1alpha1.LabelKeyCluster:   cg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     cg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cg, v1alpha1.SchemeGroupVersion.WithKind("TiCDCGroup")),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"secrets",
				},
			},
		},
	}
}

func newRoleBinding(cg *v1alpha1.TiCDCGroup) *rbacv1.RoleBinding {
	sa := "default"
	o := cg.Spec.Template.Spec.Overlay
	if o != nil && o.Pod != nil && o.Pod.Spec != nil && o.Pod.Spec.ServiceAccountName != "" {
		sa = o.Pod.Spec.ServiceAccountName
	}
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RBACName(cg.Name),
			Namespace: cg.Namespace,
			Labels: map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiCDC,
				v1alpha1.LabelKeyCluster:   cg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     cg.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cg, v1alpha1.SchemeGroupVersion.WithKind("TiCDCGroup")),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     RBACName(cg.Name),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      sa,
				Namespace: cg.Namespace,
			},
		},
	}
}
