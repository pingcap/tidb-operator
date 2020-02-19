// Copyright 2020 PingCAP, Inc.
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

package autoscaler

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/autoscaler"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
)

type ControlInterface interface {
	ResconcileAutoScaler(ta *v1alpha1.TidbClusterAutoScaler) error
}

func NewDefaultAutoScalerControl(recorder record.EventRecorder, asm autoscaler.AutoScalerManager) ControlInterface {
	return &defaultAutoScalerControl{
		recoder:           recorder,
		autoScalerManager: asm,
	}
}

type defaultAutoScalerControl struct {
	recoder           record.EventRecorder
	autoScalerManager autoscaler.AutoScalerManager
}

func (tac *defaultAutoScalerControl) ResconcileAutoScaler(ta *v1alpha1.TidbClusterAutoScaler) error {
	var errs []error
	if err := tac.reconcileAutoScaler(ta); err != nil {
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}

func (tac *defaultAutoScalerControl) reconcileAutoScaler(ta *v1alpha1.TidbClusterAutoScaler) error {
	return tac.autoScalerManager.Sync(ta)
}

var _ ControlInterface = &defaultAutoScalerControl{}

type FakeAutoScalerControl struct {
	err error
}

func NewFakeAutoScalerControl() *FakeAutoScalerControl {
	return &FakeAutoScalerControl{}
}

func (tac *FakeAutoScalerControl) SetReconcileAutoScalerError(err error) {
	tac.err = err
}

func (tac *FakeAutoScalerControl) ResconcileAutoScaler(ta *v1alpha1.TidbClusterAutoScaler) error {
	if tac.err != nil {
		return tac.err
	}
	return nil
}

var _ ControlInterface = &FakeAutoScalerControl{}
