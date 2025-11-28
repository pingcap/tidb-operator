// Copyright 2025 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package volumes

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
)

type vacPVCModifier struct {
	deps *controller.Dependencies
	pm   PodVolumeModifier
}

func newVACPVCModifier(deps *controller.Dependencies) *vacPVCModifier {
	return &vacPVCModifier{
		deps: deps,
		pm:   NewPodVolumeModifier(deps),
	}
}

func (m *vacPVCModifier) modifyVolumes(ctx *componentVolumeContext) error {
	if ctx.status.GetPhase() != v1alpha1.NormalPhase {
		return fmt.Errorf("component phase is not Normal")
	}

	if err := m.tryToRecreateSTS(ctx); err != nil {
		return err
	}

	if err := m.tryToModifyPVCs(ctx); err != nil {
		return err
	}

	return nil
}

func (m *vacPVCModifier) tryToRecreateSTS(ctx *componentVolumeContext) error {
	ns := ctx.sts.Namespace
	name := ctx.sts.Name

	isSynced, err := m.isStatefulSetSynced(ctx, ctx.sts)
	if err != nil {
		return fmt.Errorf("change in number of volumes is not supported. For pd, tikv, tidb supported with VolumeReplacing feature. Reconciliation is blocked due to : %s", err.Error())
	}
	if isSynced {
		return nil
	}

	if utils.StatefulSetIsUpgrading(ctx.sts) {
		return fmt.Errorf("component sts %s/%s is upgrading", ctx.sts.Name, ctx.sts.Namespace)
	}

	if err := utils.DeleteStatefulSetWithOrphan(ctx, m.deps.StatefulSetControl, m.deps.TiDBClusterControl, ctx.tc, ctx.sts); err != nil {
		return fmt.Errorf("delete sts %s/%s with vac for component %s failed: %s", ns, name, ctx.ComponentID(), err)
	}

	klog.Infof("recreate statefulset %s/%s with vac for component %s", ns, name, ctx.ComponentID())

	// component manager will create the sts in next reconciliation
	return nil
}

func (m *vacPVCModifier) tryToModifyPVCs(ctx *componentVolumeContext) error {
	var errs []error
	for _, pod := range ctx.pods {
		volumes, err := m.pm.GetActualVolumes(pod, ctx.desiredVolumes)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := m.tryToModifyOnePodPVCs(ctx, volumes); err != nil {
			errs = append(errs, err)
			continue
		}
	}
	return errutil.NewAggregate(errs)
}

func (m *vacPVCModifier) tryToModifyOnePodPVCs(ctx *componentVolumeContext, volumes []ActualVolume) error {
	var errs []error
	for i := range volumes {
		actual := &volumes[i]
		if !m.isPVCModified(actual.PVC, actual.Desired.Size, actual.Desired.VolumeAttributesClassName) {
			continue
		}

		pvc := actual.PVC.DeepCopy()
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = actual.Desired.Size
		pvc.Spec.VolumeAttributesClassName = actual.Desired.VolumeAttributesClassName
		updated, err := m.deps.KubeClientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(ctx, pvc, metav1.UpdateOptions{})
		if err != nil {
			errs = append(errs, err)
			continue
		}

		klog.Infof("modify vac to %s and size to %s for pvc %s/%s of component %s successfully",
			ignoreNil(actual.Desired.VolumeAttributesClassName), actual.Desired.Size.String(), pvc.Namespace, pvc.Name, ctx.ComponentID())
		actual.PVC = updated
	}
	return errutil.NewAggregate(errs)
}

func (m *vacPVCModifier) isStatefulSetSynced(ctx *componentVolumeContext, sts *appsv1.StatefulSet) (bool, error) {
	for i := range sts.Spec.VolumeClaimTemplates {
		volTemplate := &sts.Spec.VolumeClaimTemplates[i]
		volName := v1alpha1.StorageVolumeName(volTemplate.Name)
		desired := getDesiredVolumeByName(ctx.desiredVolumes, volName)
		if desired == nil {
			return false, fmt.Errorf("volume %s unsupported to remove from %s", volName, ctx.ComponentID())
		}

		if m.isPVCModified(volTemplate, desired.Size, desired.VolumeAttributesClassName) {
			return false, nil
		}
	}

	return true, nil
}

func (m *vacPVCModifier) isPVCModified(pvc *corev1.PersistentVolumeClaim, desiredSize resource.Quantity, desiredVACName *string) bool {
	size := getStorageSize(pvc.Spec.Resources.Requests)
	if size.Cmp(desiredSize) != 0 {
		return true
	}

	vacName := pvc.Spec.VolumeAttributesClassName
	return ignoreNil(vacName) != ignoreNil(desiredVACName)
}
