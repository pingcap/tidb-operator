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
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apps "k8s.io/client-go/listers/apps/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type tikvGroupMemberManager struct {
	cli          versioned.Interface
	setControl   controller.StatefulSetControlInterface
	svcControl   controller.ServiceControlInterface
	pdControl    pdapi.PDControlInterface
	typedControl controller.TypedControlInterface
	setLister    apps.StatefulSetLister
	genericCli   client.Client
}

func NewTiKVGroupMemberManager() manager.TiKVGroupManager {
	return &tikvGroupMemberManager{}
}

func (tgm *tikvGroupMemberManager) Sync(tg *v1alpha1.TiKVGroup) error {
	err := tgm.checkWhetherRegistered(tg)
	if err != nil {
		return err
	}
	klog.V(4).Infof("tikvgroup member [%s/%s] allowed to syncing", tg.Namespace, tg.Name)
	return nil
}

func (tgm *tikvGroupMemberManager) checkWhetherRegistered(tg *v1alpha1.TiKVGroup) error {
	tcName := tg.Spec.ClusterName
	tcNamespace := tg.Namespace

	tc, err := tgm.cli.PingcapV1alpha1().TidbClusters(tcNamespace).Get(tcName, metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		return err
	}

	for _, tikvGroup := range tc.Status.TiKVGroups {
		// found tikvgroup in the tidbcluster's status, allowed to syncing
		if tikvGroup.Name == tg.Name {
			return nil
		}
	}

	// register itself to the target tidbcluster
	newGroups := append(tc.Status.TiKVGroups, v1alpha1.GroupRef{Name: tg.Name})
	err = controller.GuaranteedUpdate(tgm.genericCli, tc, func() error {
		tc.Status.TiKVGroups = newGroups
		return nil
	})
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("tg[%s/%s] register itself to tc[%s/%s] successfully, requeue", tg.Namespace, tg.Name, tcNamespace, tcName)
	return controller.RequeueErrorf(msg)
}
