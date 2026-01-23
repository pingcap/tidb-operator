// Copyright 2022 PingCAP, Inc.
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
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	errutil "k8s.io/apimachinery/pkg/util/errors"
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

type PVCModifierInterface interface {
	Sync(tc *v1alpha1.TidbCluster) error
}

type pvcModifier struct {
	utils *volCompareUtils

	rawModifier *rawPVCModifier
	vacModifier *vacPVCModifier
}

func NewPVCModifier(deps *controller.Dependencies) PVCModifierInterface {
	return &pvcModifier{
		utils:       newVolCompareUtils(deps),
		rawModifier: newRawPVCModifier(deps),
		vacModifier: newVACPVCModifier(deps),
	}
}

func (p *pvcModifier) Sync(tc *v1alpha1.TidbCluster) error {
	components := tc.AllComponentStatus()
	errs := []error{}

	for _, comp := range components {
		if v1alpha1.IsPDMSMemberType(comp.MemberType()) {
			// not need storage
			continue
		}
		ctx, err := p.utils.BuildContextForTC(tc, comp)
		if err != nil {
			errs = append(errs, fmt.Errorf("build ctx used by modifier for %s/%s:%s failed: %w", tc.Namespace, tc.Name, comp.MemberType(), err))
			continue
		}

		if hasVAC(ctx.desiredVolumes) {
			err = p.vacModifier.modifyVolumes(ctx)
		} else {
			err = p.rawModifier.modifyVolumes(ctx)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("modify volumes for %s failed: %w", ctx.ComponentID(), err))
			continue
		}
	}

	return errutil.NewAggregate(errs)
}

func hasVAC(volumes []DesiredVolume) bool {
	for i := range volumes {
		if volumes[i].VolumeAttributesClassName != nil && len(*volumes[i].VolumeAttributesClassName) > 0 {
			return true
		}
	}
	return false
}
