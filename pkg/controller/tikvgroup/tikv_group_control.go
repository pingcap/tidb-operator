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

package tikvgroup

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog"
)

type ControlInterface interface {
	ReconcileTiKVGroup(ta *v1alpha1.TiKVGroup) error
}

func NewDefaultTikvGroupControl() ControlInterface {
	return &defaultTiKVGroupControl{}
}

type defaultTiKVGroupControl struct {
	// TODO: sync manager who control the TiKVGroup
}

func (dtc *defaultTiKVGroupControl) ReconcileTiKVGroup(tg *v1alpha1.TiKVGroup) error {
	var errs []error
	if err := dtc.reconcileTiKVGroup(tg); err != nil {
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}

func (dtc *defaultTiKVGroupControl) reconcileTiKVGroup(tg *v1alpha1.TiKVGroup) error {
	//TODO: start syncing
	klog.Infof("sync TiKVGroup[%s/%s]", tg.Namespace, tg.Name)
	return nil
}
