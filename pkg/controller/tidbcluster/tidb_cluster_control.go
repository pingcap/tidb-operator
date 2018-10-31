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

package tidbcluster

import (
	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/util"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
)

const (
// unused
// pdConnTimeout = 2 * time.Second
)

// ControlInterface implements the control logic for updating TidbClusters and their children StatefulSets.
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateTidbCluster implements the control logic for StatefulSet creation, update, and deletion
	UpdateTidbCluster(*v1alpha1.TidbCluster) error
}

// NewDefaultTidbClusterControl returns a new instance of the default implementation TidbClusterControlInterface that
// implements the documented semantics for TidbClusters.
func NewDefaultTidbClusterControl(
	tcControl controller.TidbClusterControlInterface,
	pdMemberManager manager.Manager,
	tikvMemberManager manager.Manager,
	tidbMemberManager manager.Manager,
	reclaimPolicyManager manager.Manager,
	metaManager manager.Manager,
	recorder record.EventRecorder) ControlInterface {
	return &defaultTidbClusterControl{
		tcControl,
		pdMemberManager,
		tikvMemberManager,
		tidbMemberManager,
		reclaimPolicyManager,
		metaManager,
		recorder,
	}
}

type defaultTidbClusterControl struct {
	tcControl            controller.TidbClusterControlInterface
	pdMemberManager      manager.Manager
	tikvMemberManager    manager.Manager
	tidbMemberManager    manager.Manager
	reclaimPolicyManager manager.Manager
	metaManager          manager.Manager
	recorder             record.EventRecorder
}

// UpdateStatefulSet executes the core logic loop for a tidbcluster.
func (tcc *defaultTidbClusterControl) UpdateTidbCluster(tc *v1alpha1.TidbCluster) error {
	// perform the main update function and get the status
	oldStatus := tc.Status.DeepCopy()
	var errs []error

	err := tcc.updateTidbCluster(tc)
	if err != nil {
		errs = append(errs, err)
	}

	if !apiequality.Semantic.DeepEqual(&tc.Status, oldStatus) {
		_, err := tcc.tcControl.UpdateTidbCluster(tc.DeepCopy(), &tc.Status, oldStatus)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errorutils.NewAggregate(errs)
}

func (tcc *defaultTidbClusterControl) updateTidbCluster(tc *v1alpha1.TidbCluster) error {
	// PD
	err := tcc.pdMemberManager.Sync(tc)
	if err != nil {
		return err
	}

	if !tcc.IsPDAvailable(tc) {
		glog.Infof("tidbcluster: [%s/%s]'s pd cluster is not running.", tc.GetNamespace(), tc.GetName())
		return nil
	}

	// TiKV
	err = tcc.tikvMemberManager.Sync(tc)
	if err != nil {
		return err
	}

	// Wait tikv status sync
	if !tcc.IsTiKVAvailable(tc) {
		glog.Infof("tidbcluster: [%s/%s]'s tikv cluster is not running.", tc.GetNamespace(), tc.GetName())
		return nil
	}

	// TiDB
	err = tcc.tidbMemberManager.Sync(tc)
	if err != nil {
		return err
	}

	// ReclaimPolicyManager
	err = tcc.reclaimPolicyManager.Sync(tc)
	if err != nil {
		return err
	}

	// MetaManager
	err = tcc.metaManager.Sync(tc)
	if err != nil {
		return err
	}

	return nil
}

func (tcc *defaultTidbClusterControl) IsPDAvailable(tc *v1alpha1.TidbCluster) bool {
	lowerLimit := tc.Spec.PD.Replicas/2 + 1
	if int32(len(tc.Status.PD.Members)) < lowerLimit {
		return false
	}

	var availableNum int32
	for _, pdMember := range tc.Status.PD.Members {
		if pdMember.Health {
			availableNum++
		}
	}

	if availableNum < lowerLimit {
		return false
	}

	if tc.Status.PD.StatefulSet.ReadyReplicas < lowerLimit {
		return false
	}

	return true
}

func (tcc *defaultTidbClusterControl) IsTiKVAvailable(tc *v1alpha1.TidbCluster) bool {
	var lowerLimit int32 = 1
	if int32(len(tc.Status.TiKV.Stores)) < lowerLimit {
		return false
	}

	var availableNum int32
	for _, store := range tc.Status.TiKV.Stores {
		if store.State == util.StoreUpState {
			availableNum++
		}
	}

	if availableNum < lowerLimit {
		return false
	}

	if tc.Status.TiKV.StatefulSet.ReadyReplicas < lowerLimit {
		return false
	}

	return true
}
