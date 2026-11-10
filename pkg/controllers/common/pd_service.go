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
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
)

// GetPDService checks the API server on a cache miss because the Service cache is label-filtered.
func GetPDService(ctx context.Context, cached, apiReader client.Reader, key client.ObjectKey) (*corev1.Service, error) {
	service := &corev1.Service{}
	err := cached.Get(ctx, key, service)
	if errors.IsNotFound(err) {
		err = apiReader.Get(ctx, key, service)
	}
	return service, err
}

// ReconcilePDService reconciles an internal PD Service with the desired controller owner.
// It may transfer a Service from a legacy PDGroup to its Cluster in place.
func ReconcilePDService(
	ctx context.Context, c client.Client, apiReader client.Reader, desired *corev1.Service,
	legacyGroups ...*v1alpha1.PDGroup,
) error {
	key := client.ObjectKeyFromObject(desired)
	current, err := GetPDService(ctx, c, apiReader, key)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create fails on a concurrent name collision; the next reconcile checks ownership.
			return c.Create(ctx, desired)
		}
		return err
	}

	owner := metav1.GetControllerOf(current)
	target := desired.OwnerReferences[0]
	if owner == nil || owner.UID == "" {
		return fmt.Errorf("PD Service %s has no controller owner", key)
	}
	var legacyPeerPort int32
	if owner.UID != target.UID {
		index := slices.IndexFunc(legacyGroups, func(group *v1alpha1.PDGroup) bool {
			return group.UID == owner.UID
		})
		if index < 0 {
			return fmt.Errorf("PD Service %s is not controlled by the desired owner or a permitted legacy PDGroup", key)
		}
		legacyPeerPort = coreutil.PDGroupPeerPort(legacyGroups[index])
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
	updated.Spec.Ports = mergePDServicePorts(current.Spec.Ports, desired.Spec.Ports, legacyPeerPort)

	if equality.Semantic.DeepEqual(current, updated) {
		return nil
	}
	patch := client.MergeFromWithOptions(current, client.MergeFromWithOptimisticLock{})
	return c.Patch(ctx, updated, patch)
}

func mergePDServicePorts(current, desired []corev1.ServicePort, legacyPeerPort int32) []corev1.ServicePort {
	ports := slices.Clone(desired)
	for _, port := range current {
		index := slices.IndexFunc(desired, func(want corev1.ServicePort) bool {
			return port.Port == want.Port && port.Protocol == want.Protocol
		})
		if index >= 0 {
			// appProtocol is supplied by other managers on the matching port.
			if ports[index].AppProtocol == nil {
				ports[index].AppProtocol = port.AppProtocol
			}
			continue
		}
		if slices.ContainsFunc(desired, func(want corev1.ServicePort) bool { return want.Name == port.Name }) {
			continue
		}
		// The Cluster Service drops the old PDGroup's peer port during adoption.
		if port.Name == v1alpha1.PDPortNamePeer && port.Port == legacyPeerPort && port.Protocol == corev1.ProtocolTCP &&
			port.TargetPort == intstr.FromString(v1alpha1.PDPortNamePeer) {
			continue
		}
		ports = append(ports, port)
	}
	return ports
}
