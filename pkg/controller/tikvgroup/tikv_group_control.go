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
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/errors"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog"
)

type ControlInterface interface {
	ReconcileTiKVGroup(ta *v1alpha1.TiKVGroup) error
}

func NewDefaultTikvGroupControl(
	tgControl controller.TiKVGroupControlInterface,
	manager manager.TiKVGroupManager) ControlInterface {
	return &defaultTiKVGroupControl{
		tgControl: tgControl,
		manager:   manager,
	}
}

type defaultTiKVGroupControl struct {
	// TODO: sync manager who control the TiKVGroup
	tgControl controller.TiKVGroupControlInterface
	manager   manager.TiKVGroupManager
}

func (dtc *defaultTiKVGroupControl) ReconcileTiKVGroup(tg *v1alpha1.TiKVGroup) error {
	var errs []error
	if err := dtc.reconcileTiKVGroup(tg); err != nil {
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}

func (dtc *defaultTiKVGroupControl) reconcileTiKVGroup(tg *v1alpha1.TiKVGroup) error {
	klog.Infof("sync TiKVGroup[%s/%s]", tg.Namespace, tg.Name)

	//TODO: defaulting and validating

	var errs []error
	oldStatus := tg.Status.DeepCopy()

	if err := dtc.updateTiKVGroup(tg); err != nil {
		errs = append(errs, err)
	}

	// TODO: update conditionUpdater

	if apiequality.Semantic.DeepEqual(&tg.Status, oldStatus) {
		return errorutils.NewAggregate(errs)
	}

	if _, err := dtc.tgControl.UpdateTiKVGroup(tg.DeepCopy(), &tg.Status, oldStatus); err != nil {
		errs = append(errs, err)
	}

	return errorutils.NewAggregate(errs)
}

func (dtc *defaultTiKVGroupControl) updateTiKVGroup(tg *v1alpha1.TiKVGroup) error {
	// TODO: sync PV

	// TODO: clean orphan Pods

	// TODO: syncing restart pods

	if err := dtc.manager.Sync(tg); err != nil {
		return err
	}

	// TODO: metaManager syncing

	// TODO: pvc cleaner

	// TODO: syncing status manager
	return nil
}
