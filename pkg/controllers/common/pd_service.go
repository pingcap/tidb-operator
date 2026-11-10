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

package common

import (
	"context"
	"fmt"
	"maps"
	"slices"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

// ReconcilePDService reconciles an internal PD Service with the desired controller owner.
// It may transfer a Service from a legacy PDGroup to its Cluster in place.
func ReconcilePDService(ctx context.Context, c client.Client, desired *corev1.Service, legacyOwners ...types.UID) error {
	key := client.ObjectKeyFromObject(desired)
	current := &corev1.Service{}
	if err := c.Get(ctx, key, current); err != nil {
		if errors.IsNotFound(err) {
			// Create fails on a concurrent name collision; the next reconcile checks ownership.
			return c.Create(ctx, desired)
		}
		return err
	}

	owner := metav1.GetControllerOf(current)
	target := desired.OwnerReferences[0]
	if owner == nil || owner.UID == "" ||
		(owner.UID != target.UID && !slices.Contains(legacyOwners, owner.UID)) {
		return fmt.Errorf("PD Service %s is not controlled by the desired owner or a permitted legacy PDGroup", key)
	}
	if current.Spec.Type != corev1.ServiceTypeClusterIP ||
		current.Spec.ClusterIP == corev1.ClusterIPNone || !current.DeletionTimestamp.IsZero() {
		return fmt.Errorf("PD Service %s is not an active, non-headless ClusterIP Service", key)
	}

	updated := current.DeepCopy()
	if updated.Labels == nil {
		updated.Labels = make(map[string]string)
	}
	maps.Copy(updated.Labels, desired.Labels)
	if owner.UID != target.UID {
		delete(updated.Labels, v1alpha1.LabelKeyGroup)
	}
	for i := range updated.OwnerReferences {
		if updated.OwnerReferences[i].UID == owner.UID {
			updated.OwnerReferences[i] = target
			break
		}
	}
	updated.Spec.Selector = desired.Spec.Selector
	updated.Spec.IPFamilyPolicy = desired.Spec.IPFamilyPolicy
	updated.Spec.Ports = slices.Clone(desired.Spec.Ports)
	for _, port := range current.Spec.Ports {
		if port.Name == v1alpha1.PDPortNameClient || port.Name == v1alpha1.PDPortNamePeer {
			continue
		}
		if slices.ContainsFunc(desired.Spec.Ports, func(want corev1.ServicePort) bool {
			return port.Port == want.Port && port.Protocol == want.Protocol
		}) {
			continue
		}
		updated.Spec.Ports = append(updated.Spec.Ports, port)
	}

	if equality.Semantic.DeepEqual(current, updated) {
		return nil
	}
	patch := client.MergeFromWithOptions(current, client.MergeFromWithOptimisticLock{})
	return c.Patch(ctx, updated, patch)
}
