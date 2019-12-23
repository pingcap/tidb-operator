// Copyright 2019 PingCAP, Inc.
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

package tests

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/slack"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/klog"
)

func (oa *operatorActions) setCabundleFromApiServer(info *OperatorConfig) error {

	serverVersion, err := oa.kubeCli.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to get api server version")
	}
	sv := utilversion.MustParseSemantic(serverVersion.GitVersion)
	klog.Infof("ServerVersion: %v", serverVersion.String())

	if sv.LessThan(utilversion.MustParseSemantic("v1.13.0")) && len(info.Cabundle) < 1 {
		namespace := "kube-system"
		name := "extension-apiserver-authentication"
		cm, err := oa.kubeCli.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		content, existed := cm.Data["client-ca-file"]
		if !existed {
			return fmt.Errorf("failed to get caBundle from configmap[%s/%s]", namespace, name)
		}
		info.Cabundle = content
		return nil
	}
	return nil
}

func (oa *operatorActions) CheckUpgradeWithPodWebhook(info *TidbClusterConfig) error {
	ns := info.Namespace
	tcName := info.ClusterName

	pdStsName := getStsName(tcName, v1alpha1.PDMemberType)
	tikvStsName := getStsName(tcName, v1alpha1.TiKVMemberType)
	tidbStsName := getStsName(tcName, v1alpha1.TiDBMemberType)

	pdDesiredReplicas, err := strconv.ParseInt(info.Resources["pd.replicas"], 10, 32)
	if err != nil {
		return err
	}
	tikvDesiredReplicas, err := strconv.ParseInt(info.Resources["tikv.replicas"], 10, 32)
	if err != nil {
		return err
	}
	tidbDesiredReplicas, err := strconv.ParseInt(info.Resources["tidb.replicas"], 10, 32)
	if err != nil {
		return err
	}

	f := func(stsName, namespace string, desiredReplicas int64) (*apps.StatefulSet, bool, error) {
		sts, err := oa.tcStsGetter.StatefulSets(namespace).Get(stsName, metav1.GetOptions{})
		if err != nil {
			klog.Infof("failed to fetch sts[%s/%s]", namespace, stsName)
			return nil, false, err
		}
		if sts.Status.UpdatedReplicas != int32(desiredReplicas) {
			klog.Infof("sts[%s/%s]'s updatedReplicas[%d]!=desiredReplicas[%d]", namespace, stsName, sts.Status.UpdatedReplicas, int32(desiredReplicas))
			return nil, false, nil
		}
		if sts.Status.CurrentReplicas != int32(desiredReplicas) {
			klog.Infof("sts[%s/%s]'s currentReplicas[%d]!=desiredReplicas[%d]", namespace, stsName, sts.Status.CurrentReplicas, int32(desiredReplicas))
			return nil, false, nil
		}
		return sts, true, nil
	}

	return wait.Poll(10*time.Second, 50*time.Minute, func() (done bool, err error) {

		pdsts, ready, err := f(pdStsName, ns, pdDesiredReplicas)
		if !ready || err != nil {
			return ready, err
		}
		if pdsts.Spec.Template.Spec.Containers[0].Image != info.PDImage {
			return false, nil
		}

		tikvsts, ready, err := f(tikvStsName, ns, tikvDesiredReplicas)
		if !ready || err != nil {
			return ready, err
		}

		if tikvsts.Spec.Template.Spec.Containers[0].Image != info.TiKVImage {
			return false, nil
		}
		tidbsts, ready, err := f(tidbStsName, ns, tidbDesiredReplicas)
		if !ready || err != nil {
			return ready, err
		}

		if tidbsts.Spec.Template.Spec.Containers[0].Image != info.TiDBImage && tidbsts.Spec.Template.Spec.Containers[1].Image != info.TiDBImage {
			return false, nil
		}
		return true, nil
	})
}

func (oa *operatorActions) CheckUpgradeWithPodWebhookOrDie(info *TidbClusterConfig) {
	if err := oa.CheckUpgradeWithPodWebhook(info); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func getStsName(tcName string, memberType v1alpha1.MemberType) string {
	return fmt.Sprintf("%s-%s", tcName, memberType.String())
}
