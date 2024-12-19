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
	"context"
	"fmt"
	"strconv"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/volumes/cloud/aws"
)

const (
	annoKeyPVCSpecRevision     = "spec.tidb.pingcap.com/revision"
	annoKeyPVCSpecStorageClass = "spec.tidb.pingcap.com/storage-class"
	annoKeyPVCSpecStorageSize  = "spec.tidb.pingcap.com/storage-size"

	annoKeyPVCStatusRevision     = "status.tidb.pingcap.com/revision"
	annoKeyPVCStatusStorageClass = "status.tidb.pingcap.com/storage-class"
	annoKeyPVCStatusStorageSize  = "status.tidb.pingcap.com/storage-size"

	annoKeyPVCLastTransitionTimestamp = "status.tidb.pingcap.com/last-transition-timestamp"

	defaultModifyWaitingDuration = time.Minute * 1
)

func getBoundPVFromPVC(ctx context.Context, cli client.Client, pvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolume, error) {
	if pvc.Status.Phase != corev1.ClaimBound {
		return nil, fmt.Errorf("pvc %s/%s is not bound", pvc.Namespace, pvc.Name)
	}

	name := pvc.Spec.VolumeName
	var pv corev1.PersistentVolume
	if err := cli.Get(ctx, client.ObjectKey{Name: name}, &pv); err != nil {
		return nil, fmt.Errorf("failed to get PV %s: %w", name, err)
	}

	return &pv, nil
}

func getStorageSize(r corev1.ResourceList) resource.Quantity {
	return r[corev1.ResourceStorage]
}

func ignoreNil(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func setLastTransitionTimestamp(pvc *corev1.PersistentVolumeClaim) {
	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}

	pvc.Annotations[annoKeyPVCLastTransitionTimestamp] = metav1.Now().Format(time.RFC3339)
}

func upgradeRevision(pvc *corev1.PersistentVolumeClaim) {
	rev := 1
	str, ok := pvc.Annotations[annoKeyPVCSpecRevision]
	if ok {
		oldRev, err := strconv.Atoi(str)
		if err != nil {
			klog.Warningf("revision format err: %v, reset to 0", err)
			oldRev = 0
		}
		rev = oldRev + 1
	}

	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}

	pvc.Annotations[annoKeyPVCSpecRevision] = strconv.Itoa(rev)
}

// isPVCSpecMatched checks if the storage class or storage size of the PVC is changed.
func isPVCSpecMatched(pvc *corev1.PersistentVolumeClaim, scName string, size resource.Quantity) bool {
	isChanged := false

	oldSc := ignoreNil(pvc.Spec.StorageClassName)
	scAnno, ok := pvc.Annotations[annoKeyPVCSpecStorageClass]
	if ok && scAnno != "" {
		oldSc = scAnno
	}

	if scName != "" && oldSc != scName {
		isChanged = true
	}

	oldSize, ok := pvc.Annotations[annoKeyPVCSpecStorageSize]
	if !ok {
		quantity := getStorageSize(pvc.Spec.Resources.Requests)
		oldSize = quantity.String()
	}
	if oldSize != size.String() {
		isChanged = true
	}

	return isChanged
}

func snapshotStorageClassAndSize(pvc *corev1.PersistentVolumeClaim, scName string, size resource.Quantity) bool {
	isChanged := isPVCSpecMatched(pvc, scName, size)

	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}

	if scName != "" {
		pvc.Annotations[annoKeyPVCSpecStorageClass] = scName
	}
	pvc.Annotations[annoKeyPVCSpecStorageSize] = size.String()

	return isChanged
}

func getStorageClassNameFromPVC(pvc *corev1.PersistentVolumeClaim) string {
	sc := ignoreNil(pvc.Spec.StorageClassName)

	scAnno, ok := pvc.Annotations[annoKeyPVCStatusStorageClass]
	if ok && scAnno != "" {
		sc = scAnno
	}

	return sc
}

func needModify(pvc *corev1.PersistentVolumeClaim, desired *DesiredVolume) bool {
	size := desired.Size
	scName := desired.GetStorageClassName()

	return isPVCStatusMatched(pvc, scName, size)
}

func isPVCStatusMatched(pvc *corev1.PersistentVolumeClaim, scName string, size resource.Quantity) bool {
	oldSc := getStorageClassNameFromPVC(pvc)
	isChanged := isStorageClassChanged(oldSc, scName)

	oldSize, ok := pvc.Annotations[annoKeyPVCStatusStorageSize]
	if !ok {
		quantity := getStorageSize(pvc.Spec.Resources.Requests)
		oldSize = quantity.String()
	}
	if oldSize != size.String() {
		isChanged = true
	}
	if isChanged {
		klog.Infof("volume %s/%s is changed, sc (%s => %s), size (%s => %s)", pvc.Namespace, pvc.Name, oldSc, scName, oldSize, size.String())
	}

	return isChanged
}

func isStorageClassChanged(pre, cur string) bool {
	if cur != "" && pre != cur {
		return true
	}
	return false
}

func isVolumeAttributesClassChanged(actual *ActualVolume) bool {
	return areStringsDifferent(actual.VACName, actual.Desired.VACName)
}

func areStringsDifferent(pre, cur *string) bool {
	if pre == cur {
		return false
	}
	return pre == nil || cur == nil || *pre != *cur
}

func isPVCRevisionChanged(pvc *corev1.PersistentVolumeClaim) bool {
	specRevision, statusRevision := pvc.Annotations[annoKeyPVCSpecRevision], pvc.Annotations[annoKeyPVCStatusRevision]
	return specRevision != statusRevision
}

func isVolumeExpansionSupported(sc *storagev1.StorageClass) (bool, error) {
	if sc == nil {
		// always assume expansion is supported
		return true, fmt.Errorf("expansion cap of volume is unknown")
	}
	if sc.AllowVolumeExpansion == nil {
		return false, nil
	}
	return *sc.AllowVolumeExpansion, nil
}

// NewModifier creates a volume modifier.
// TODO: check the feature gate via webhook to decide whether to use the native modifier.
func NewModifier(ctx context.Context, logger logr.Logger, cli client.Client) (Modifier, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error(err, "failed to load aws config, will use native modifier")
		return NewNativeModifier(cli, logger), nil
	}
	logger.Info("use aws ebs modifier")
	return NewRawModifier(aws.NewEBSModifier(&awsCfg, logger), cli, logger), nil
}

// SyncPVCs gets the actual PVCs and compares them with the expected PVCs.
// If the actual PVCs are different from the expected PVCs, it will update the PVCs.
func SyncPVCs(ctx context.Context, cli client.Client,
	expectPVCs []*corev1.PersistentVolumeClaim, vm Modifier, logger logr.Logger,
) (wait bool, err error) {
	for _, expectPVC := range expectPVCs {
		var actualPVC corev1.PersistentVolumeClaim
		if err := cli.Get(ctx, client.ObjectKey{Namespace: expectPVC.Namespace, Name: expectPVC.Name}, &actualPVC); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return false, fmt.Errorf("can't get expectPVC %s/%s: %w", expectPVC.Namespace, expectPVC.Name, err)
			}

			// Create PVC
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
		if vm.ShouldModify(ctx, vol) {
			logger.Info("modifying volume's attributes", "volume", vol.String())
			if e := vm.Modify(ctx, vol); e != nil {
				return false, fmt.Errorf("failed to modify volume's attributes %s/%s: %w", expectPVC.Namespace, expectPVC.Name, e)
			}
			continue
		}

		logger.Info("volume's attributes are not changed", "volume", vol.String())
		if expectPVC.Spec.StorageClassName != nil &&
			actualPVC.Spec.StorageClassName != nil &&
			*expectPVC.Spec.StorageClassName != *actualPVC.Spec.StorageClassName {
			// Avoid updating the storage class name as it's immutable.
			expectPVC.Spec.StorageClassName = actualPVC.Spec.StorageClassName
		}
		if err := cli.Apply(ctx, expectPVC); err != nil {
			return false, fmt.Errorf("can't update expectPVC %s/%s: %w", expectPVC.Namespace, expectPVC.Name, err)
		}
	}
	return wait, nil
}