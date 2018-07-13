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
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	mm "github.com/pingcap/tidb-operator/new-operator/pkg/controller/tidbcluster/membermanager"
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
	UpdateTidbCluster(*v1.TidbCluster) error
}

// NewDefaultTidbClusterControl returns a new instance of the default implementation TidbClusterControlInterface that
// implements the documented semantics for TidbClusters.
func NewDefaultTidbClusterControl(
	statusUpdater StatusUpdaterInterface,
	pdMemberManager mm.MemberManager,
	recorder record.EventRecorder) ControlInterface {
	return &defaultTidbClusterControl{
		statusUpdater,
		pdMemberManager,
		recorder,
	}
}

type defaultTidbClusterControl struct {
	statusUpdater   StatusUpdaterInterface
	pdMemberManager mm.MemberManager
	recorder        record.EventRecorder
}

// UpdateStatefulSet executes the core logic loop for a tidbcluster.
func (tcc *defaultTidbClusterControl) UpdateTidbCluster(tc *v1.TidbCluster) error {
	// perform the main update function and get the status
	err := tcc.updateTidbCluster(tc)
	if err != nil {
		return err
	}

	// update the tidbCluster's status
	return tcc.updateTidbClusterStatus(tc, &tc.Status)
}

func (tcc *defaultTidbClusterControl) updateTidbCluster(tc *v1.TidbCluster) error {
	// PD
	err := tcc.pdMemberManager.Sync(tc)
	if err != nil {
		return err
	}

	// // TiKV
	// err = tcc.tikvMemberManager.Sync(tc)
	// if err != nil {
	// 	return err
	// }

	// // TiDB
	// err = tcc.tidbMemberManager.Sync(tc)
	// if err != nil {
	// 	return err
	// }

	// // Monitor
	// err = tcc.monitorMemberManager.Sync(tc)
	// if err != nil {
	//	return err
	// }

	return nil
}

func (tcc *defaultTidbClusterControl) updateTidbClusterStatus(tc *v1.TidbCluster, status *v1.TidbClusterStatus) error {
	tc = tc.DeepCopy()
	status = status.DeepCopy()
	return tcc.statusUpdater.UpdateTidbClusterStatus(tc, status)
}
