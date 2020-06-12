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

package tidbgroup

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog"
)

type ControlInterface interface {
	ReconcileTiDBGroup(ta *v1alpha1.TiDBGroup) error
}

func NewDefaultTiDBGroupControl() ControlInterface {
	return &defaultTiDBGroupControl{}
}

type defaultTiDBGroupControl struct {
	// TODO: sync manager who control the TiDBGroup
}

func (dtc *defaultTiDBGroupControl) ReconcileTiDBGroup(tg *v1alpha1.TiDBGroup) error {
	var errs []error
	if err := dtc.reconcileTiDBGroup(tg); err != nil {
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}

func (dtc *defaultTiDBGroupControl) reconcileTiDBGroup(tg *v1alpha1.TiDBGroup) error {
	//TODO: start syncing
	klog.Infof("sync TiDBGroup[%s/%s]", tg.Namespace, tg.Name)
	return nil
}
