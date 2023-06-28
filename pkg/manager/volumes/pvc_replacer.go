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
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	errutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
)

type PVCReplacerInterface interface {
	UpdateStatus(tc *v1alpha1.TidbCluster) error
	Sync(tc *v1alpha1.TidbCluster) error
}

type pvcReplacer struct {
	deps  *controller.Dependencies
	pm    PodVolumeModifier
	utils *volCompareUtils
}

func NewPVCReplacer(deps *controller.Dependencies) PVCReplacerInterface {
	return &pvcReplacer{
		deps:  deps,
		pm:    NewPodVolumeModifier(deps),
		utils: newVolCompareUtils(deps),
	}
}

func (p *pvcReplacer) UpdateStatus(tc *v1alpha1.TidbCluster) error {
	components := tc.AllComponentStatus()
	errs := []error{}

	for _, comp := range components {
		isVolReplacing := false
		ctx, err := p.utils.BuildContextForTC(tc, comp)
		if err != nil {
			errs = append(errs, fmt.Errorf("build ctx used by replacer for %s/%s:%s failed: %w", tc.Namespace, tc.Name, comp.MemberType(), err))
			continue
		}
		isSynced, err := p.utils.IsStatefulSetSynced(ctx, ctx.sts, false)
		if err != nil {
			errs = append(errs, err)
		}
		if !isSynced {
			klog.Infof("Statefulset not synced for volumes! %s/%s for component %s", ctx.sts.Namespace, ctx.sts.Name, ctx.ComponentID())
			isVolReplacing = true
			// mark status and continue?
		}
		for _, pod := range ctx.pods {
			podSynced, err := p.utils.IsPodSyncedForReplacement(ctx, pod)
			if err != nil {
				errs = append(errs, err)
				continue // Do not mark status for error, maybe API call will retry.
			}
			if !podSynced {
				klog.Infof("Pod not synced for volumes! %s/%s for component %s", pod.Namespace, pod.Name, ctx.ComponentID())
				isVolReplacing = true
				//break
			}
		}
		// TODO: mark status
		klog.Infof("Marking vol replacing status for %s as %t", ctx.ComponentID(), isVolReplacing)
	}

	return errutil.NewAggregate(errs)
}

func (p *pvcReplacer) Sync(tc *v1alpha1.TidbCluster) error {
	return nil
}
