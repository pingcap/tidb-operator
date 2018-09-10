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
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
)

type tidbFailover struct {
	tidbFailoverPeriod time.Duration
}

// NewTiDBFailover returns a tidbFailover instance
func NewTiDBFailover(failoverPeriod time.Duration) Failover {
	return &tidbFailover{
		tidbFailoverPeriod: failoverPeriod,
	}
}

func (tf *tidbFailover) Failover(tc *v1alpha1.TidbCluster) error {
	if tc.Status.TiDB.FailureMembers == nil {
		tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{}
	}

	for _, tidbMember := range tc.Status.TiDB.Members {
		_, exist := tc.Status.TiDB.FailureMembers[tidbMember.Name]
		deadline := tidbMember.LastTransitionTime.Add(tf.tidbFailoverPeriod)
		if !tidbMember.Health && time.Now().After(deadline) && !exist {
			tc.Status.TiDB.FailureMembers[tidbMember.Name] = v1alpha1.TiDBFailureMember{PodName: tidbMember.Name, Replicas: tc.Spec.TiDB.Replicas}
			break
		}
	}

	// increase the replicas to add a new TiDB peer
	for _, failureMember := range tc.Status.TiDB.FailureMembers {
		if failureMember.Replicas+1 > tc.Spec.TiDB.Replicas {
			tc.Spec.TiDB.Replicas = failureMember.Replicas + 1
		}
	}

	return nil
}

func (tf *tidbFailover) Recover(tc *v1alpha1.TidbCluster) {
	maxReplicas := int32(0)
	minReplicas := int32(0)
	for _, failureMember := range tc.Status.TiDB.FailureMembers {
		if minReplicas == int32(0) {
			minReplicas = failureMember.Replicas
		}
		if failureMember.Replicas > maxReplicas {
			maxReplicas = failureMember.Replicas
		} else if failureMember.Replicas < minReplicas {
			minReplicas = failureMember.Replicas
		}
	}
	if maxReplicas+1 == tc.Spec.TiDB.Replicas {
		tc.Spec.TiDB.Replicas = minReplicas
	}

	tc.Status.TiDB.FailureMembers = nil
}

func needRecover(tc *v1alpha1.TidbCluster) bool {
	if int(tc.Spec.TiDB.Replicas) != len(tc.Status.TiDB.Members) {
		return false
	}
	for _, tidbMember := range tc.Status.TiDB.Members {
		if !tidbMember.Health {
			return false
		}
	}
	if tc.Status.TiDB.FailureMembers == nil || len(tc.Status.TiDB.FailureMembers) == 0 {
		return false
	}
	return true
}

func needFailover(tc *v1alpha1.TidbCluster) bool {
	if tc.Spec.TiDB.Replicas != tc.Status.TiDB.StatefulSet.Replicas {
		return false
	}
	for _, tidbMember := range tc.Status.TiDB.Members {
		if !tidbMember.Health {
			return true
		}
	}
	return false
}

type fakeTiDBFailover struct{}

// NewFakeTiDBFailover returns a fake Failover
func NewFakeTiDBFailover() Failover {
	return &fakeTiDBFailover{}
}

func (ftf *fakeTiDBFailover) Failover(tc *v1alpha1.TidbCluster) error {
	return nil
}

func (ftf *fakeTiDBFailover) Recover(tc *v1alpha1.TidbCluster) {
	return
}
