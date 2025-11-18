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

package volumes

import (
	context "context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	errutil "k8s.io/apimachinery/pkg/util/errors"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

// LegacySyncPVCs gets the actual PVCs and compares them with the expected PVCs.
// If the actual PVCs are different from the expected PVCs, it will update the PVCs.
func LegacySyncPVCs(ctx context.Context, cli client.Client,
	expectPVCs []*corev1.PersistentVolumeClaim, vm Modifier, logger logr.Logger,
) (wait bool, err error) {
	for _, expectPVC := range expectPVCs {
		var actualPVC corev1.PersistentVolumeClaim
		if err := cli.Get(ctx, client.ObjectKey{Namespace: expectPVC.Namespace, Name: expectPVC.Name}, &actualPVC); err != nil {
			if !errors.IsNotFound(err) {
				return false, fmt.Errorf("can't get PVC %s/%s: %w", expectPVC.Namespace, expectPVC.Name, err)
			}

			// Create PVC if it doesn't exist
			if e := cli.Apply(ctx, expectPVC); e != nil {
				return false, fmt.Errorf("can't create expectPVC %s/%s: %w", expectPVC.Namespace, expectPVC.Name, e)
			}
			continue
		}

		if actualPVC.Status.Phase != corev1.ClaimBound {
			// do not try to modify the PVC if it's not bound yet
			wait = true
			continue
		}

		// Set default storage class name if it's not specified and the claim is bound.
		// Otherwise, it will be considered as a change and trigger a PVC update.
		if expectPVC.Spec.StorageClassName == nil && actualPVC.Status.Phase == corev1.ClaimBound {
			expectPVC.Spec.StorageClassName = actualPVC.Spec.StorageClassName
		}

		vol, err := vm.GetActualVolume(ctx, expectPVC, &actualPVC)
		if err != nil {
			return false, fmt.Errorf("failed to get the actual volume: %w", err)
		}

		needWait, skipUpdate, err := handleVolumeModification(ctx, vm, vol, expectPVC, logger)
		if err != nil {
			return false, err
		}
		logger.Info("handle volume modification", "needWait", needWait, "skipUpdate", skipUpdate, "pvc", expectPVC.Name)
		if needWait {
			wait = true
		}
		if skipUpdate {
			continue
		}

		if err := updatePVC(ctx, cli, expectPVC, &actualPVC); err != nil {
			return false, err
		}
	}
	return wait, nil
}

func SyncPVCs(ctx context.Context, c client.Client, pvcs []*corev1.PersistentVolumeClaim) error {
	var errList []error
	var waitErrList []error
	for _, pvc := range pvcs {
		if err := syncPVC(ctx, c, pvc); err != nil {
			if task.IsWaitError(err) {
				waitErrList = append(waitErrList, err)
				continue
			}
			errList = append(errList, err)
		}
	}
	if len(errList) != 0 {
		return errutil.NewAggregate(errList)
	}
	if len(waitErrList) != 0 {
		return errutil.NewAggregate(waitErrList)
	}

	return nil
}

func syncPVC(ctx context.Context, c client.Client, pvc *corev1.PersistentVolumeClaim) error {
	if err := c.Apply(
		ctx,
		pvc,
		// If VAC is not enabled, we use params in SC to modify volume.
		// So we allow SC ref being changed.
		// If VAC is enabled, we should also not apply changed SC to PVC.
		// Reset to SC of current PVC.
		client.Immutable("spec", "storageClassName"),
	); err != nil {
		return err
	}
	msgs := isPVCSynced(pvc)
	if len(msgs) != 0 {
		return fmt.Errorf("%w: %v", task.ErrWait, msgs)
	}

	return nil
}

func isPVCSynced(pvc *corev1.PersistentVolumeClaim) []string {
	var msgs []string
	for _, cond := range pvc.Status.Conditions {
		if cond.Status == corev1.ConditionTrue {
			msgs = append(msgs, fmt.Sprintf("%s: %s", cond.Type, cond.Message))
		}
	}

	return msgs
}
