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

package autoscaler

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
)

const (
	// The TidbCluster for the external query will be "<original-tcname>-<component>-external"
	externalTcNamePattern = "%s-%s-external"
	specialUseLabelKey    = "specialUse"
	specialUseHotRegion   = "hotRegion"
)

func (am *autoScalerManager) syncExternalResult(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType, targetReplicas int32) error {
	externalTcName := fmt.Sprintf(externalTcNamePattern, tc.ClusterName, component.String())
	externalTc, err := am.tcLister.TidbClusters(tc.Namespace).Get(externalTcName)
	if err != nil {
		if errors.IsNotFound(err) {
			if targetReplicas <= 0 {
				return nil
			}
			return am.createExternalAutoCluster(tc, externalTcName, tac, component, targetReplicas)
		}

		klog.Errorf("tac[%s/%s] failed to get external tc[%s/%s], err: %v", tac.Namespace, tac.Name, tc.Namespace, externalTcName, err)
		return err
	}

	if targetReplicas <= 0 {
		err := am.cli.PingcapV1alpha1().TidbClusters(externalTc.Namespace).Delete(externalTc.Name, nil)
		if err != nil {
			klog.Errorf("tac[%s/%s] failed to delete external tc[%s/%s], err: %v", tac.Namespace, tac.Name, tc.Namespace, externalTcName, err)
		}
		return err
	}

	return am.updateExternalAutoCluster(externalTc, tac, component, targetReplicas)
}

func (am *autoScalerManager) createExternalAutoCluster(tc *v1alpha1.TidbCluster, externalTcName string, tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType, targetReplicas int32) error {
	autoTc := newAutoScalingCluster(tc, tac, externalTcName, component.String())

	switch component {
	case v1alpha1.TiDBMemberType:
		autoTc.Spec.TiDB.Replicas = targetReplicas
	case v1alpha1.TiKVMemberType:
		autoTc.Spec.TiKV.Replicas = targetReplicas
		autoTc.Spec.TiKV.Config.Server.Labels[specialUseLabelKey] = specialUseHotRegion
	}

	_, err := am.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(autoTc)
	if err != nil {
		klog.Errorf("tac[%s/%s] failed to create external tc[%s/%s], err: %v", tac.Namespace, tac.Name, tc.Namespace, externalTcName, err)
	}
	return err
}

func (am *autoScalerManager) updateExternalAutoCluster(externalTc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType, targetReplicas int32) error {
	updated := externalTc.DeepCopy()
	switch component {
	case v1alpha1.TiDBMemberType:
		if updated.Spec.TiDB.Replicas == targetReplicas {
			return nil
		}
		updated.Spec.TiDB.Replicas = targetReplicas
	case v1alpha1.TiKVMemberType:
		if updated.Spec.TiKV.Replicas == targetReplicas {
			return nil
		}
		updated.Spec.TiKV.Replicas = targetReplicas
	}

	_, err := am.tcControl.UpdateTidbCluster(updated, &updated.Status, &externalTc.Status)
	if err != nil {
		klog.Errorf("tac[%s/%s] failed to update external tc[%s/%s], err: %v", tac.Namespace, tac.Name, externalTc.Namespace, externalTc.Name, err)
	}
	return err
}
