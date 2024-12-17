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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
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

// EnsureInstanceSubResourceDeleted ensures the sub resources of an instance are deleted.
// It deletes the pod, pvc and configmap of the instance currently.
// For pod and configmap, the name of the resource is the same as the instance name.
// For pvc, it should contain the instance name as the value of the label "app.kubernetes.io/instance".
// TODO: retain policy support
func EnsureInstanceSubResourceDeleted(ctx context.Context, cli client.Client,
	namespace, name string, podOpts ...client.DeleteOption,
) error {
	var needWait bool // wait after we call delete on some resources
	var pod corev1.Pod
	if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &pod); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get pod %s/%s: %w", namespace, name, err)
		}
	} else {
		if err := cli.Delete(ctx, &pod, podOpts...); err != nil {
			return fmt.Errorf("failed to delete pod %s/%s: %w", namespace, name, err)
		}
		needWait = true
	}

	var cm corev1.ConfigMap
	if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &cm); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get cm %s/%s: %w", namespace, name, err)
		}
	} else {
		if err := cli.Delete(ctx, &cm); err != nil {
			return fmt.Errorf("failed to delete cm %s/%s: %w", namespace, name, err)
		}
		needWait = true
	}

	var pvcList corev1.PersistentVolumeClaimList
	if err := cli.List(ctx, &pvcList, client.InNamespace(namespace),
		client.MatchingLabels{
			v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
			v1alpha1.LabelKeyInstance:  name,
		}); err != nil {
		return fmt.Errorf("failed to list pvc %s/%s: %w", namespace, name, err)
	}
	for i := range pvcList.Items {
		pvc := pvcList.Items[i]
		if err := cli.Delete(ctx, &pvc); err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete pvc %s/%s: %w", namespace, pvc.Name, err)
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
