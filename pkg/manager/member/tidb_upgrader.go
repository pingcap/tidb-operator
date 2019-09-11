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
	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	// MaxResignDDLOwnerCount is the max regign DDL owner count
	MaxResignDDLOwnerCount = 3
)

type tidbUpgrader struct {
	podLister   corelisters.PodLister
	tidbControl controller.TiDBControlInterface
}

// NewTiDBUpgrader returns a tidb Upgrader
func NewTiDBUpgrader(tidbControl controller.TiDBControlInterface, podLister corelisters.PodLister) Upgrader {
	return &tidbUpgrader{
		tidbControl: tidbControl,
		podLister:   podLister,
	}
}

func (tdu *tidbUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Status.PD.Phase == v1alpha1.UpgradePhase || tc.Status.TiKV.Phase == v1alpha1.UpgradePhase {
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
	if !templateEqual(newSet.Spec.Template, oldSet.Spec.Template) {
		return nil
	}

	if tc.Status.TiDB.StatefulSet.UpdateRevision == tc.Status.TiDB.StatefulSet.CurrentRevision {
		return nil
	}

	setUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	for i := tc.Status.TiDB.StatefulSet.Replicas - 1; i >= 0; i-- {
		podName := tidbPodName(tcName, i)
		pod, err := tdu.podLister.Pods(ns).Get(podName)
		if err != nil {
			return err
		}
		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == tc.Status.TiDB.StatefulSet.UpdateRevision {
			if member, exist := tc.Status.TiDB.Members[podName]; !exist || !member.Health {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb upgraded pod: [%s] is not ready", ns, tcName, podName)
			}
			continue
		}
		return tdu.upgradeTiDBPod(tc, i, newSet)
	}

	return nil
}

func (tdu *tidbUpgrader) upgradeTiDBPod(tc *v1alpha1.TidbCluster, ordinal int32, newSet *apps.StatefulSet) error {
	tcName := tc.GetName()
	if tc.Spec.TiDB.Replicas > 1 {
		if member, exist := tc.Status.TiDB.Members[tidbPodName(tcName, ordinal)]; exist && member.Health {
			hasResign, err := tdu.tidbControl.ResignDDLOwner(tc, ordinal)
			if (!hasResign || err != nil) && tc.Status.TiDB.ResignDDLOwnerRetryCount < MaxResignDDLOwnerCount {
				glog.Errorf("tidb upgrader: failed to resign ddl owner to %s, %v", member.Name, err)
				tc.Status.TiDB.ResignDDLOwnerRetryCount++
				return err
			}
			glog.Infof("tidb upgrader: resign ddl owner to %s successfully", member.Name)
		}
	}

	tc.Status.TiDB.ResignDDLOwnerRetryCount = 0
	setUpgradePartition(newSet, ordinal)
	return nil
}

type fakeTiDBUpgrader struct{}

// NewFakeTiDBUpgrader returns a fake tidb upgrader
func NewFakeTiDBUpgrader() Upgrader {
	return &fakeTiDBUpgrader{}
}

func (ftdu *fakeTiDBUpgrader) Upgrade(tc *v1alpha1.TidbCluster, _ *apps.StatefulSet, _ *apps.StatefulSet) error {
	tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
	return nil
}
