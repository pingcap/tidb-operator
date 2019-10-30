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

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		if exist && tidbMember.Health {
			delete(tc.Status.TiDB.FailureMembers, tidbMember.Name)
			glog.Infof("tidb failover: delete %s from tidb failoverMembers", tidbMember.Name)
		}
	}

	if tc.Spec.TiDB.MaxFailoverCount > 0 && len(tc.Status.TiDB.FailureMembers) >= int(tc.Spec.TiDB.MaxFailoverCount) {
		glog.Warningf("the failure members count reached the limit:%d", tc.Spec.TiDB.MaxFailoverCount)
		return nil
	}
	for _, tidbMember := range tc.Status.TiDB.Members {
		_, exist := tc.Status.TiDB.FailureMembers[tidbMember.Name]
		deadline := tidbMember.LastTransitionTime.Add(tf.tidbFailoverPeriod)
		if !tidbMember.Health && time.Now().After(deadline) && !exist {
			tc.Status.TiDB.FailureMembers[tidbMember.Name] = v1alpha1.TiDBFailureMember{
				PodName:   tidbMember.Name,
				CreatedAt: metav1.Now(),
			}
			break
		}
	}

	return nil
}

func (tf *tidbFailover) Recover(tc *v1alpha1.TidbCluster) {
	tc.Status.TiDB.FailureMembers = nil
}

type fakeTiDBFailover struct{}

// NewFakeTiDBFailover returns a fake Failover
func NewFakeTiDBFailover() Failover {
	return &fakeTiDBFailover{}
}

func (ftf *fakeTiDBFailover) Failover(_ *v1alpha1.TidbCluster) error {
	return nil
}

func (ftf *fakeTiDBFailover) Recover(_ *v1alpha1.TidbCluster) {
	return
}
