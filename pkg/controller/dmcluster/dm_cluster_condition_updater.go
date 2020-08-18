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

package dmcluster

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	utildmcluster "github.com/pingcap/tidb-operator/pkg/util/dmcluster"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

// DMClusterConditionUpdater interface that translates cluster state into
// into dm cluster status conditions.
type DMClusterConditionUpdater interface {
	Update(*v1alpha1.DMCluster) error
}

type dmClusterConditionUpdater struct {
}

var _ DMClusterConditionUpdater = &dmClusterConditionUpdater{}

func (u *dmClusterConditionUpdater) Update(dc *v1alpha1.DMCluster) error {
	u.updateReadyCondition(dc)
	// in the future, we may return error when we need to Kubernetes API, etc.
	return nil
}

func allStatefulSetsAreUpToDate(dc *v1alpha1.DMCluster) bool {
	isUpToDate := func(status *appsv1.StatefulSetStatus, requireExist bool) bool {
		if status == nil {
			return !requireExist
		}
		return status.CurrentRevision == status.UpdateRevision
	}
	return (isUpToDate(dc.Status.Master.StatefulSet, true)) &&
		(isUpToDate(dc.Status.Worker.StatefulSet, false))
}

func (u *dmClusterConditionUpdater) updateReadyCondition(dc *v1alpha1.DMCluster) {
	status := v1.ConditionFalse
	reason := ""
	message := ""

	switch {
	case !allStatefulSetsAreUpToDate(dc):
		reason = utildmcluster.StatfulSetNotUpToDate
		message = "Statefulset(s) are in progress"
	case !dc.MasterAllMembersReady():
		reason = utildmcluster.MasterUnhealthy
		message = "dm-master(s) are not healthy"
	default:
		status = v1.ConditionTrue
		reason = utildmcluster.Ready
		message = "DM cluster is fully up and running"
	}
	cond := utildmcluster.NewDMClusterCondition(v1alpha1.DMClusterReady, status, reason, message)
	utildmcluster.SetDMClusterCondition(&dc.Status, *cond)
}
