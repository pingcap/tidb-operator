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
)

type ControlInterface interface {
	ResconcileAutoScaler(ta *v1alpha1.TidbClusterAutoScaler) error
}

func NewDefaultAutoScalerControl(asm autoscaler.AutoScalerManager) ControlInterface {
	return &defaultAutoScalerControl{
		autoScalerManager: asm,
	}
}

type defaultAutoScalerControl struct {
	autoScalerManager autoscaler.AutoScalerManager
}

func (c *defaultAutoScalerControl) ResconcileAutoScaler(ta *v1alpha1.TidbClusterAutoScaler) error {
	var errs []error
	if err := c.reconcileAutoScaler(ta); err != nil {
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}

func (c *defaultAutoScalerControl) reconcileAutoScaler(ta *v1alpha1.TidbClusterAutoScaler) error {
	return c.autoScalerManager.Sync(ta)
}

var _ ControlInterface = &defaultAutoScalerControl{}

type FakeAutoScalerControl struct {
	err error
}

func NewFakeAutoScalerControl() *FakeAutoScalerControl {
	return &FakeAutoScalerControl{}
}

func (c *FakeAutoScalerControl) SetReconcileAutoScalerError(err error) {
	c.err = err
}

func (c *FakeAutoScalerControl) ResconcileAutoScaler(ta *v1alpha1.TidbClusterAutoScaler) error {
	if c.err != nil {
		return c.err
	}
	return nil
}

var _ ControlInterface = &FakeAutoScalerControl{}
