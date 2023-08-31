// Copyright 2023 PingCAP, Inc.
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
	"context"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
)

type PVCReplacerInterface interface {
	UpdateStatus(tc *v1alpha1.TidbCluster) error
	Sync(tc *v1alpha1.TidbCluster) error
}

type pvcReplacer struct {
	deps  *controller.Dependencies
	utils *volCompareUtils
}

func NewPVCReplacer(deps *controller.Dependencies) PVCReplacerInterface {
	return &pvcReplacer{
		deps:  deps,
		utils: newVolCompareUtils(deps),
	}
}

func (p *pvcReplacer) getVolReplaceStatusForComponent(tc *v1alpha1.TidbCluster, comp v1alpha1.ComponentStatus) (bool, error) {
	// Note: This runs before component, so minimize returning errors as they will block cluster creation
	ctx, err := p.utils.BuildContextForTC(tc, comp)
	if err != nil {
		// Do not return an actual error as this may block cluster creation.
		klog.Warningf("skipping replace status: build ctx used by replacer for %s/%s:%s failed: %v", tc.Namespace, tc.Name, comp.MemberType(), err)
		return comp.GetVolReplaceInProgress(), nil // Do not change existing status.
	}

	// Ignore errors as they only indicate change in number of volume which replacer can handle.
	isSynced, _ := p.utils.IsStatefulSetSynced(ctx, ctx.sts)

	if !isSynced {
		klog.Infof("Statefulset not synced for volumes! %s/%s for component %s", ctx.sts.Namespace, ctx.sts.Name, ctx.ComponentID())
		return true, nil
	}
	for _, pod := range ctx.pods {
		podSynced, err := p.utils.IsPodSyncedForReplacement(ctx, pod)
		if err != nil {
			return comp.GetVolReplaceInProgress(), err // Statefulset is fine, so okay to block until pod is accessible without error.
		}
		if !podSynced {
			klog.Infof("Pod not synced for volumes! %s/%s for component %s", pod.Namespace, pod.Name, ctx.ComponentID())
			return true, nil
		}
	}
	return false, nil
}

func (p *pvcReplacer) UpdateStatus(tc *v1alpha1.TidbCluster) error {
	components := tc.AllComponentStatus()
	errs := []error{}

	for _, comp := range components {
		status, err := p.getVolReplaceStatusForComponent(tc, comp)
		if err != nil {
			errs = append(errs, err)
		}
		// TiKV from 1 -> 2 for volume replace messes up with max-replication=3 preventing delete stores afterwards.
		if status && comp.MemberType() == v1alpha1.TiKVMemberType && tc.TiKVStsDesiredReplicas() < 3 {
			status = false
			klog.Warningf("%s for cluster %s/%s need volume replace, but has too few replicas, so refusing to replace.", comp.MemberType(), tc.GetNamespace(), tc.GetName())
		}
		if status != comp.GetVolReplaceInProgress() {
			klog.Infof("changing VolReplaceInProgress status to %t for %s/%s/%s", status, tc.GetNamespace(), tc.GetName(), comp.MemberType())
		}
		comp.SetVolReplaceInProgress(status)
	}

	return errutil.NewAggregate(errs)
}

func (p *pvcReplacer) Sync(tc *v1alpha1.TidbCluster) error {
	components := tc.AllComponentStatus()
	errs := []error{}

	// Note: implicit ordering of tc components PD > TiDB > TiKV ... important for correct order of syncing.
	for _, comp := range components {
		if !comp.GetVolReplaceInProgress() {
			continue
		}
		ctx, err := p.utils.BuildContextForTC(tc, comp)
		if err != nil {
			errs = append(errs, fmt.Errorf("build ctx used by replacer sync for %s/%s:%s failed: %w", tc.Namespace, tc.Name, comp.MemberType(), err))
			continue
		}
		err = p.replaceVolumes(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("replace volumes for %s/%s %s failed: %w", tc.Namespace, tc.Name, ctx.ComponentID(), err))
		}
	}

	return errutil.NewAggregate(errs)
}

func (p *pvcReplacer) replaceVolumes(ctx *componentVolumeContext) error {
	if ctx.status.GetPhase() == v1alpha1.ScalePhase {
		// Note: only wait for scaling, phase may show up as upgrading but will be blocked
		// for replacing here to effect the config + volume change together.
		return fmt.Errorf("component phase is Scaling, waiting to complete.")
	}
	if err := p.tryToRecreateSTS(ctx); err != nil {
		return err
	}
	if err := p.tryToReplacePVC(ctx); err != nil {
		return err
	}
	return nil
}

func (p *pvcReplacer) tryToRecreateSTS(ctx *componentVolumeContext) error {
	ns := ctx.sts.Namespace
	name := ctx.sts.Name

	// Ignore errors as they only indicate change in number of volume which replacer can handle.
	isSynced, _ := p.utils.IsStatefulSetSynced(ctx, ctx.sts)
	if isSynced {
		return nil
	}
	if utils.StatefulSetIsUpgrading(ctx.sts) {
		return fmt.Errorf("component sts %s/%s is upgrading", ctx.sts.Namespace, ctx.sts.Name)
	}

	orphan := metav1.DeletePropagationOrphan
	if err := p.deps.KubeClientset.AppsV1().StatefulSets(ns).Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: &orphan}); err != nil {
		return fmt.Errorf("delete sts %s/%s for component %s failed: %s", ns, name, ctx.ComponentID(), err)
	}

	return fmt.Errorf("waiting on recreate statefulset %s/%s for component %s", ns, name, ctx.ComponentID())
}

func isVolumeReplacing(pod *corev1.Pod) bool {
	_, exist := pod.Annotations[v1alpha1.ReplaceVolumeAnnKey]
	return exist
}

func (p *pvcReplacer) startVolumeReplace(pod *corev1.Pod) error {
	pod = pod.DeepCopy()

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	pod.Annotations[v1alpha1.ReplaceVolumeAnnKey] = v1alpha1.ReplaceVolumeValueTrue
	if _, err := p.deps.KubeClientset.CoreV1().Pods(pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("add replace volume annotation to pod %s/%s failed: %s", pod.Namespace, pod.Name, err)
	}
	klog.Infof("added replace volume annotation to pod %s/%s", pod.Namespace, pod.Name)
	return nil
}

func (p *pvcReplacer) tryToReplacePVC(ctx *componentVolumeContext) error {
	for _, pod := range ctx.pods {
		// Ensure only one is being replaced at a time.
		if isVolumeReplacing(pod) {
			return fmt.Errorf("waiting for pending volume replace for pod %s", pod.Name)
		}
	}
	for _, pod := range ctx.pods {
		podSynced, err := p.utils.IsPodSyncedForReplacement(ctx, pod)
		if err != nil {
			return err
		}
		if podSynced {
			continue
		}
		if err := p.startVolumeReplace(pod); err != nil {
			return err
		}
		return fmt.Errorf("started volume replace for pod %s, waiting", pod.Name)
	}
	return nil
}

type fakePVCReplacer struct {
}

func (f fakePVCReplacer) UpdateStatus(tc *v1alpha1.TidbCluster) error {
	return nil
}

func (f fakePVCReplacer) Sync(tc *v1alpha1.TidbCluster) error {
	return nil
}

func NewFakePVCReplacer() PVCReplacerInterface {
	return &fakePVCReplacer{}
}
