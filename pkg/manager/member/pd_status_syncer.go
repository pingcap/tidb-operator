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

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	beta1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/listers/apps/v1beta1"
)

// PDStatusSyncer implements the logic for syncing main statefulset status, extra statefulsets statuses and the pd members status
type PDStatusSyncer interface {
	SyncStatefulSetStatus(*v1alpha1.TidbCluster) error
	SyncExtraStatefulSetsStatus(*v1alpha1.TidbCluster) error
	SyncPDMembers(*v1alpha1.TidbCluster) error
}

type pdStatusSyncer struct {
	setLister v1beta1.StatefulSetLister
	pdControl controller.PDControlInterface
}

func (pss *pdStatusSyncer) SyncStatefulSetStatus(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	set, err := pss.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName, tc.Spec.PD.Name))
	if err != nil {
		return err
	}

	tc.Status.PD.StatefulSet = set.Status.DeepCopy()
	return nil
}

func (pss *pdStatusSyncer) SyncExtraStatefulSetsStatus(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	for _, pdSpec := range tc.Spec.PDs {
		set, err := pss.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName, pdSpec.Name))
		if err != nil {
			return err
		}

		setName := set.GetName()
		if tc.Status.PD.StatefulSets == nil {
			tc.Status.PD.StatefulSets = map[string]beta1.StatefulSetStatus{}
		}
		tc.Status.PD.StatefulSets[setName] = *set.Status.DeepCopy()
	}
	return nil
}

func (pss *pdStatusSyncer) SyncPDMembers(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	var err error
	defer func() {
		tc.Status.PD.Synced = false
		if err == nil {
			tc.Status.PD.Synced = true
		}
	}()

	pdClient := pss.pdControl.GetPDClient(tc)
	healthInfo, err := pdClient.GetHealth()
	if err != nil {
		return err
	}
	leader, err := pdClient.GetPDLeader()
	if err != nil {
		return err
	}

	pdStatus := map[string]v1alpha1.PDMember{}
	for _, memberHealth := range healthInfo.Healths {
		id := memberHealth.MemberID
		memberID := fmt.Sprintf("%d", id)
		var clientURL string
		if len(memberHealth.ClientUrls) > 0 {
			clientURL = memberHealth.ClientUrls[0]
		}
		name := memberHealth.Name
		if name == "" {
			glog.Warningf("PD member: [%d] doesn't have a name, and can't get it from clientUrls: [%s], memberHealth Info: [%v] in [%s/%s]",
				id, memberHealth.ClientUrls, memberHealth, ns, tcName)
			continue
		}

		status := v1alpha1.PDMember{
			Name:      name,
			ID:        memberID,
			ClientURL: clientURL,
			Health:    memberHealth.Health,
		}

		oldPDMember, exist := tc.Status.PD.Members[name]
		if exist {
			status.LastTransitionTime = oldPDMember.LastTransitionTime
		}
		if !exist || status.Health != oldPDMember.Health {
			status.LastTransitionTime = metav1.Now()
		}

		pdStatus[name] = status
	}

	tc.Status.PD.Members = pdStatus
	tc.Status.PD.Leader = tc.Status.PD.Members[leader.GetName()]
	return nil
}
