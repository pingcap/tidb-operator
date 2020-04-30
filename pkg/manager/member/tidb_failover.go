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
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

type tidbFailover struct {
	tidbFailoverPeriod time.Duration
	recorder           record.EventRecorder
	podLister          corelisters.PodLister
}

// NewTiDBFailover returns a tidbFailover instance
func NewTiDBFailover(failoverPeriod time.Duration, recorder record.EventRecorder, podLister corelisters.PodLister) Failover {
	return &tidbFailover{
		tidbFailoverPeriod: failoverPeriod,
		recorder:           recorder,
		podLister:          podLister,
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
			klog.Infof("tidb failover: delete %s from tidb failoverMembers", tidbMember.Name)
		}
	}

	if tc.Spec.TiDB.MaxFailoverCount == nil || *tc.Spec.TiDB.MaxFailoverCount <= 0 {
		klog.Infof("tidb failover is disabled for %s/%s, skipped", tc.Namespace, tc.Name)
		return nil
	}

	maxFailoverCount := *tc.Spec.TiDB.MaxFailoverCount
	for _, tidbMember := range tc.Status.TiDB.Members {
		_, exist := tc.Status.TiDB.FailureMembers[tidbMember.Name]
		deadline := tidbMember.LastTransitionTime.Add(tf.tidbFailoverPeriod)
		if !tidbMember.Health && time.Now().After(deadline) && !exist {
			if len(tc.Status.TiDB.FailureMembers) >= int(maxFailoverCount) {
				klog.Warningf("the failover count reachs the limit (%d), no more failover pods will be created", maxFailoverCount)
				break
			}
			pod, err := tf.podLister.Pods(tc.Namespace).Get(tidbMember.Name)
			if err != nil {
				return err
			}
			_, condition := podutil.GetPodCondition(&pod.Status, corev1.PodScheduled)
			if condition == nil || condition.Status != corev1.ConditionTrue {
				// if a member is unheathy because it's not scheduled yet, we
				// should not create failover pod for it
				klog.Warningf("pod %s/%s is not scheduled yet, skipping failover", pod.Namespace, pod.Name)
				continue
			}
			tc.Status.TiDB.FailureMembers[tidbMember.Name] = v1alpha1.TiDBFailureMember{
				PodName:   tidbMember.Name,
				CreatedAt: metav1.Now(),
			}
			msg := fmt.Sprintf("tidb[%s] is unhealthy", tidbMember.Name)
			tf.recorder.Event(tc, corev1.EventTypeWarning, unHealthEventReason, fmt.Sprintf(unHealthEventMsgPattern, "tidb", tidbMember.Name, msg))
			break
		}
	}

	return nil
}

func (tf *tidbFailover) Recover(tc *v1alpha1.TidbCluster) {
	tc.Status.TiDB.FailureMembers = nil
}

type fakeTiDBFailover struct {
}

// NewFakeTiDBFailover returns a fake Failover
func NewFakeTiDBFailover() Failover {
	return &fakeTiDBFailover{}
}

func (ftf *fakeTiDBFailover) Failover(_ *v1alpha1.TidbCluster) error {
	return nil
}

func (ftf *fakeTiDBFailover) Recover(tc *v1alpha1.TidbCluster) {
	tc.Status.TiDB.FailureMembers = nil
}
