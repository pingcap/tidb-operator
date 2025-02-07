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

package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

// EnsureGroupSubResourceDeleted ensures the sub resources of a group are deleted.
// It only deletes the service of the group currently.
func EnsureGroupSubResourceDeleted(ctx context.Context, cli client.Client,
	namespace, name string, _ ...client.DeleteOption,
) error {
	var needWait bool // wait after we call delete on some resources
	var svcList corev1.ServiceList
	if err := cli.List(ctx, &svcList, client.InNamespace(namespace),
		client.MatchingLabels{
			v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
			v1alpha1.LabelKeyGroup:     name,
		}); err != nil {
		return fmt.Errorf("failed to list svc %s/%s: %w", namespace, name, err)
	}
	for i := range svcList.Items {
		svc := svcList.Items[i]
		if err := cli.Delete(ctx, &svc); err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete svc %s/%s: %w", namespace, svc.Name, err)
			}
			continue
		}
		needWait = true
	}

	if needWait {
		return fmt.Errorf("wait for all sub resources of %s/%s being removed", namespace, name)
	}

	return nil
}

// DeleteInstanceSubresource try to delete a subresource of an instance, e.g. pods, cms, pvcs
func DeleteInstanceSubresource[I runtime.Instance](
	ctx context.Context,
	c client.Client,
	instance I,
	objs client.ObjectList,
	opts ...client.DeleteOption,
) (wait bool, _ error) {
	wait, err := deleteSubresource(ctx, c, instance.GetNamespace(), objs, map[string]string{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyInstance:  instance.GetName(),
		v1alpha1.LabelKeyCluster:   instance.Cluster(),
		v1alpha1.LabelKeyComponent: instance.Component(),
	}, opts...)
	if err != nil {
		return false, fmt.Errorf("cannot delete sub resource for instance %s/%s: %w", instance.GetNamespace(), instance.GetName(), err)
	}
	return wait, nil
}

func DeleteGroupSubresource[G runtime.Group](
	ctx context.Context,
	c client.Client,
	group G,
	objs client.ObjectList,
	opts ...client.DeleteOption,
) (wait bool, _ error) {
	wait, err := deleteSubresource(ctx, c, group.GetNamespace(), objs, map[string]string{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyCluster:   group.Cluster(),
		v1alpha1.LabelKeyComponent: group.Component(),
		v1alpha1.LabelKeyGroup:     group.GetName(),
	}, opts...)
	if err != nil {
		return false, fmt.Errorf("cannot delete sub resource for group %s/%s: %w", group.GetNamespace(), group.GetName(), err)
	}
	return wait, nil
}

func deleteSubresource(
	ctx context.Context,
	c client.Client,
	ns string,
	objs client.ObjectList,
	labels map[string]string,
	opts ...client.DeleteOption,
) (wait bool, _ error) {
	if err := c.List(ctx, objs, client.InNamespace(ns), client.MatchingLabels(labels)); err != nil {
		return false, fmt.Errorf("failed to list %T in %s: %w", objs, ns, err)
	}

	if meta.LenList(objs) == 0 {
		return false, nil
	}

	items, err := meta.ExtractList(objs)
	if err != nil {
		return false, fmt.Errorf("failed to extract %T: %w", objs, err)
	}
	for _, item := range items {
		obj, ok := item.(client.Object)
		if !ok {
			return false, fmt.Errorf("unexpected %T", item)
		}
		if !obj.GetDeletionTimestamp().IsZero() {
			wait = true
			continue
		}
		if err := c.Delete(ctx, obj, opts...); err != nil {
			if !errors.IsNotFound(err) {
				return false, fmt.Errorf("failed to delete sub resource %s/%s: %w",
					obj.GetNamespace(),
					obj.GetName(),
					err,
				)
			}
			continue
		}
		wait = true
	}

	return wait, nil
}
