// Copyright 2018 PingCAP, Inc.
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

package member

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1beta1"
)

type pdScaler struct {
	pdControl controller.PDControlInterface
}

// NewPDScaler returns a Scaler
func NewPDScaler(pdControl controller.PDControlInterface) Scaler {
	return &pdScaler{pdControl}
}

func (psd *pdScaler) ScaleOut(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	return nil
}

// We need remove member from cluster before reducing statefulset replicas
// only remove one member at a time when scale down
func (psd *pdScaler) ScaleIn(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if tc.PDUpgrading() {
		*newSet.Spec.Replicas = *oldSet.Spec.Replicas
		return nil
	}

	ordinal := *oldSet.Spec.Replicas - 1
	memberName := fmt.Sprintf("%s-pd-%d", tc.GetName(), ordinal)
	if err := psd.pdControl.GetPDClient(tc).DeleteMember(memberName); err != nil {
		// for unit test
		*newSet.Spec.Replicas = *oldSet.Spec.Replicas
		return err
	}

	*newSet.Spec.Replicas = ordinal
	return nil
}

var _ Scaler = &pdScaler{}

type fakePDScaler struct{}

// NewFakePDScaler returns a fake Scaler
func NewFakePDScaler() Scaler {
	return &fakePDScaler{}
}

func (fsd *fakePDScaler) ScaleOut(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	return nil
}

func (fsd *fakePDScaler) ScaleIn(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	return nil
}

var _ Scaler = &fakePDScaler{}
