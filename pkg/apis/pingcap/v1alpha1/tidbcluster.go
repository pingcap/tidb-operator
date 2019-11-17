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

package v1alpha1

func (tc *TidbCluster) PDUpgrading() bool {
	return tc.Status.PD.Phase == UpgradePhase
}

func (tc *TidbCluster) TiKVUpgrading() bool {
	return tc.Status.TiKV.Phase == UpgradePhase
}

func (tc *TidbCluster) TiDBUpgrading() bool {
	return tc.Status.TiDB.Phase == UpgradePhase
}

func (tc *TidbCluster) PDAllPodsStarted() bool {
	return tc.PDRealReplicas() == tc.Status.PD.StatefulSet.Replicas
}

func (tc *TidbCluster) PDAllMembersReady() bool {
	if int(tc.PDRealReplicas()) != len(tc.Status.PD.Members) {
		return false
	}

	for _, member := range tc.Status.PD.Members {
		if !member.Health {
			return false
		}
	}
	return true
}

func (tc *TidbCluster) PDAutoFailovering() bool {
	if len(tc.Status.PD.FailureMembers) == 0 {
		return false
	}

	for _, failureMember := range tc.Status.PD.FailureMembers {
		if !failureMember.MemberDeleted {
			return true
		}
	}
	return false
}

func (tc *TidbCluster) PDRealReplicas() int32 {
	return tc.Spec.PD.Replicas + int32(len(tc.Status.PD.FailureMembers))
}

func (tc *TidbCluster) TiKVAllPodsStarted() bool {
	return tc.TiKVRealReplicas() == tc.Status.TiKV.StatefulSet.Replicas
}

func (tc *TidbCluster) TiKVAllStoresReady() bool {
	if int(tc.TiKVRealReplicas()) != len(tc.Status.TiKV.Stores) {
		return false
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.State != TiKVStateUp {
			return false
		}
	}

	return true
}

func (tc *TidbCluster) TiKVRealReplicas() int32 {
	return tc.Spec.TiKV.Replicas + int32(len(tc.Status.TiKV.FailureStores))
}

func (tc *TidbCluster) TiDBAllPodsStarted() bool {
	return tc.TiDBRealReplicas() == tc.Status.TiDB.StatefulSet.Replicas
}

func (tc *TidbCluster) TiDBAllMembersReady() bool {
	if int(tc.TiDBRealReplicas()) != len(tc.Status.TiDB.Members) {
		return false
	}

	for _, member := range tc.Status.TiDB.Members {
		if !member.Health {
			return false
		}
	}

	return true
}

func (tc *TidbCluster) TiDBRealReplicas() int32 {
	return tc.Spec.TiDB.Replicas + int32(len(tc.Status.TiDB.FailureMembers))
}

func (tc *TidbCluster) PDIsAvailable() bool {
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

	if tc.Status.PD.StatefulSet == nil || tc.Status.PD.StatefulSet.ReadyReplicas < lowerLimit {
		return false
	}

	return true
}

func (tc *TidbCluster) TiKVIsAvailable() bool {
	var lowerLimit int32 = 1
	if int32(len(tc.Status.TiKV.Stores)) < lowerLimit {
		return false
	}

	var availableNum int32
	for _, store := range tc.Status.TiKV.Stores {
		if store.State == TiKVStateUp {
			availableNum++
		}
	}

	if availableNum < lowerLimit {
		return false
	}

	if tc.Status.TiKV.StatefulSet == nil || tc.Status.TiKV.StatefulSet.ReadyReplicas < lowerLimit {
		return false
	}

	return true
}

func (tc *TidbCluster) GetClusterID() string {
	return tc.Status.ClusterID
}

func (tc *TidbCluster) Scheme() string {
	if tc.Spec.EnableTLSCluster {
		return "https"
	}
	return "http"
}
