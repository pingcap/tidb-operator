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
	"encoding/json"
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/csaupgrade"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	operatorclient "github.com/pingcap/tidb-operator/v2/pkg/client"
)

// PDServiceUpdateManager tracks fields written by the guarded Create and rollback.
const PDServiceUpdateManager = "tidb-operator-pd-service"

// GetPDService checks the API server on a cache miss because the Service cache is label-filtered.
func GetPDService(ctx context.Context, cached, apiReader client.Reader, key client.ObjectKey) (*corev1.Service, error) {
	service := &corev1.Service{}
	err := cached.Get(ctx, key, service)
	if errors.IsNotFound(err) {
		err = apiReader.Get(ctx, key, service)
	}
	return service, err
}

// ReconcilePDService applies an internal PD Service after verifying its controller owner.
// A Cluster may take over a Service from one of its own PDGroups.
func ReconcilePDService(
	ctx context.Context, c client.Client, apiReader client.Reader, desired *corev1.Service,
	legacyGroups ...*v1alpha1.PDGroup,
) error {
	key := client.ObjectKeyFromObject(desired)
	current, err := GetPDService(ctx, c, apiReader, key)
	if errors.IsNotFound(err) {
		// A create-only request cannot overwrite an object created after the read.
		return c.Create(ctx, desired, client.FieldOwner(PDServiceUpdateManager))
	}
	if err != nil {
		return err
	}

	owner := metav1.GetControllerOf(current)
	target := desired.OwnerReferences[0]
	if owner == nil || owner.UID == "" ||
		(owner.UID != target.UID && !slices.ContainsFunc(legacyGroups, func(group *v1alpha1.PDGroup) bool {
			return group.UID == owner.UID
		})) {
		return fmt.Errorf("PD Service %s is not controlled by the desired owner or a permitted legacy PDGroup", key)
	}
	if current.Spec.Type != corev1.ServiceTypeClusterIP ||
		current.Spec.ClusterIP == corev1.ClusterIPNone || !current.DeletionTimestamp.IsZero() {
		return fmt.Errorf("PD Service %s is not an active, non-headless ClusterIP Service", key)
	}

	// Make fields written by Create or rollback eligible for SSA pruning on
	// ownership transfer. The migration patch is itself resource-version guarded.
	upgrade, err := csaupgrade.UpgradeManagedFieldsPatch(current, sets.New(PDServiceUpdateManager), operatorclient.DefaultFieldManager)
	if err != nil {
		return err
	}
	if len(upgrade) > 0 {
		if err := c.Patch(ctx, current, client.RawPatch(types.JSONPatchType, upgrade)); err != nil {
			return err
		}
	}

	expected := desired.DeepCopy()
	expected.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Service"}
	expected.ResourceVersion = current.ResourceVersion
	data, err := json.Marshal(expected)
	if err != nil {
		return err
	}
	return c.Patch(ctx, current, client.RawPatch(types.ApplyPatchType, data),
		client.FieldOwner(operatorclient.DefaultFieldManager), client.ForceOwnership)
}
