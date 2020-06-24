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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type tikvGroupMemberManager struct {
	cli        versioned.Interface
	genericCli client.Client
}

func NewTiKVGroupMemberManager(
	cli versioned.Interface,
	genericCli client.Client) manager.TiKVGroupManager {
	return &tikvGroupMemberManager{
		cli:        cli,
		genericCli: genericCli,
	}
}

func (tgm *tikvGroupMemberManager) Sync(tg *v1alpha1.TiKVGroup) error {
	err := tgm.checkWhetherRegistered(tg)
	if err != nil {
		return err
	}
	klog.V(4).Infof("tikvgroup member [%s/%s] allowed to syncing", tg.Namespace, tg.Name)
	return nil
}

// checkWhetherRegistered will check whether the tikvgroup have already registered itself
// to the target tidbcluster. If have already, the tikvgroup will be allowed to syncing.
// If not, the tikvgroup will try to register itself to the tidbcluster and wait for the next round.
func (tgm *tikvGroupMemberManager) checkWhetherRegistered(tg *v1alpha1.TiKVGroup) error {
	tcName := tg.Spec.ClusterName
	tcNamespace := tg.Namespace

	klog.Infof("start to register tikvGroup[%s/%s] to tc[%s/%s]", tg.Namespace, tg.Name, tcNamespace, tcName)
	tc, err := tgm.cli.PingcapV1alpha1().TidbClusters(tcNamespace).Get(tcName, metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		return err
	}

	if tc.Status.TiKVGroups == nil || len(tc.Status.TiKVGroups) < 1 {
		return tgm.registerTiKVGroup(tg, tc)
	}

	for _, tikvGroup := range tc.Status.TiKVGroups {
		// found tikvgroup in the tidbcluster's status, allowed to syncing
		if tikvGroup.Name == tg.Name {
			return nil
		}
	}

	return tgm.registerTiKVGroup(tg, tc)
}

// register itself to the target tidbcluster
func (tgm *tikvGroupMemberManager) registerTiKVGroup(tg *v1alpha1.TiKVGroup, tc *v1alpha1.TidbCluster) error {
	tcName := tg.Spec.ClusterName
	tcNamespace := tg.Namespace
	// register itself to the target tidbcluster
	newGroups := append(tc.Status.TiKVGroups, v1alpha1.GroupRef{Name: tg.Name})
	err := controller.GuaranteedUpdate(tgm.genericCli, tc, func() error {
		tc.Status.TiKVGroups = newGroups
		return nil
	})
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("tg[%s/%s] register itself to tc[%s/%s] successfully, requeue", tg.Namespace, tg.Name, tcNamespace, tcName)
	return controller.RequeueErrorf(msg)
}
