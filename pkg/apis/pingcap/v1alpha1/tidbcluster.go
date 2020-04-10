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

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
)

const (
	// defaultHelperImage is default image of helper
	defaultHelperImage      = "busybox:1.26.2"
	defaultTimeZone         = "UTC"
	defaultEnableTLSCluster = false
	defaultEnableTLSClient  = false
	defaultExposeStatus     = true
	defaultSeparateSlowLog  = true
	defaultEnablePVReclaim  = false
)

var (
	defaultTailerSpec = TiDBSlowLogTailerSpec{
		ResourceRequirements: corev1.ResourceRequirements{},
	}
	defaultHelperSpec = HelperSpec{}
)

func (tc *TidbCluster) PDImage() string {
	image := tc.Spec.PD.Image
	baseImage := tc.Spec.PD.BaseImage
	// base image takes higher priority
	if baseImage != "" {
		version := tc.Spec.PD.Version
		if version == nil {
			version = &tc.Spec.Version
		}
		image = fmt.Sprintf("%s:%s", baseImage, *version)
	}
	return image
}

func (tc *TidbCluster) TiKVImage() string {
	image := tc.Spec.TiKV.Image
	baseImage := tc.Spec.TiKV.BaseImage
	// base image takes higher priority
	if baseImage != "" {
		version := tc.Spec.TiKV.Version
		if version == nil {
			version = &tc.Spec.Version
		}
		image = fmt.Sprintf("%s:%s", baseImage, *version)
	}
	return image
}

func (tc *TidbCluster) TiKVContainerPrivilege() *bool {
	if tc.Spec.TiKV.Privileged == nil {
		pri := false
		return &pri
	}
	return tc.Spec.TiKV.Privileged
}

func (tc *TidbCluster) TiDBImage() string {
	image := tc.Spec.TiDB.Image
	baseImage := tc.Spec.TiDB.BaseImage
	// base image takes higher priority
	if baseImage != "" {
		version := tc.Spec.TiDB.Version
		if version == nil {
			version = &tc.Spec.Version
		}
		image = fmt.Sprintf("%s:%s", baseImage, *version)
	}
	return image
}

func (tc *TidbCluster) PumpImage() *string {
	if tc.Spec.Pump == nil {
		return nil
	}
	image := tc.Spec.Pump.Image
	baseImage := tc.Spec.Pump.BaseImage
	// base image takes higher priority
	if baseImage != "" {
		version := tc.Spec.Pump.Version
		if version == nil {
			version = &tc.Spec.Version
		}
		image = fmt.Sprintf("%s:%s", baseImage, *version)
	}
	return &image
}

func (tc *TidbCluster) HelperImage() string {
	image := tc.GetHelperSpec().Image
	if image == nil {
		// for backward compatibility
		image = tc.Spec.TiDB.GetSlowLogTailerSpec().Image
	}
	if image == nil {
		return defaultHelperImage
	}
	return *image
}

func (tc *TidbCluster) HelperImagePullPolicy() corev1.PullPolicy {
	pp := tc.GetHelperSpec().ImagePullPolicy
	if pp == nil {
		// for backward compatibility
		pp = tc.Spec.TiDB.GetSlowLogTailerSpec().ImagePullPolicy
	}
	if pp == nil {
		return tc.Spec.ImagePullPolicy
	}
	return *pp
}

func (tc *TidbCluster) GetHelperSpec() HelperSpec {
	if tc.Spec.Helper == nil {
		return defaultHelperSpec
	}
	return *tc.Spec.Helper
}

func (mt MemberType) String() string {
	return string(mt)
}

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
	return tc.PDStsDesiredReplicas() == tc.PDStsActualReplicas()
}

func (tc *TidbCluster) PDAllMembersReady() bool {
	if int(tc.PDStsDesiredReplicas()) != len(tc.Status.PD.Members) {
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

func (tc *TidbCluster) PDStsDesiredReplicas() int32 {
	return tc.Spec.PD.Replicas + int32(len(tc.Status.PD.FailureMembers))
}

func (tc *TidbCluster) PDStsActualReplicas() int32 {
	stsStatus := tc.Status.PD.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.Replicas
}

func (tc *TidbCluster) TiKVAllPodsStarted() bool {
	return tc.TiKVStsDesiredReplicas() == tc.TiKVStsActualReplicas()
}

func (tc *TidbCluster) TiKVAllStoresReady() bool {
	if int(tc.TiKVStsDesiredReplicas()) != len(tc.Status.TiKV.Stores) {
		return false
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.State != TiKVStateUp {
			return false
		}
	}

	return true
}

func (tc *TidbCluster) TiKVStsDesiredReplicas() int32 {
	return tc.Spec.TiKV.Replicas + int32(len(tc.Status.TiKV.FailureStores))
}

func (tc *TidbCluster) TiKVStsActualReplicas() int32 {
	stsStatus := tc.Status.TiKV.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.Replicas
}

func (tc *TidbCluster) TiDBAllPodsStarted() bool {
	return tc.TiDBStsDesiredReplicas() == tc.TiDBStsActualReplicas()
}

func (tc *TidbCluster) TiDBAllMembersReady() bool {
	if int(tc.TiDBStsDesiredReplicas()) != len(tc.Status.TiDB.Members) {
		return false
	}

	for _, member := range tc.Status.TiDB.Members {
		if !member.Health {
			return false
		}
	}

	return true
}

func (tc *TidbCluster) TiDBStsDesiredReplicas() int32 {
	return tc.Spec.TiDB.Replicas + int32(len(tc.Status.TiDB.FailureMembers))
}

func (tc *TidbCluster) TiDBStsActualReplicas() int32 {
	stsStatus := tc.Status.TiDB.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.Replicas
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

func (tc *TidbCluster) IsTLSClusterEnabled() bool {
	return tc.Spec.TLSCluster != nil && tc.Spec.TLSCluster.Enabled
}

func (tc *TidbCluster) Scheme() string {
	if tc.IsTLSClusterEnabled() {
		return "https"
	}
	return "http"
}

func (tc *TidbCluster) Timezone() string {
	tz := tc.Spec.Timezone
	if tz == "" {
		return defaultTimeZone
	}
	return tz
}

func (tc *TidbCluster) IsPVReclaimEnabled() bool {
	enabled := tc.Spec.EnablePVReclaim
	if enabled == nil {
		return defaultEnablePVReclaim
	}
	return *enabled
}

func (tc *TidbCluster) IsTiDBBinlogEnabled() bool {
	binlogEnabled := tc.Spec.TiDB.BinlogEnabled
	if binlogEnabled == nil {
		isPumpCreated := tc.Spec.Pump != nil
		return isPumpCreated
	}
	return *binlogEnabled
}

func (tidb *TiDBSpec) IsTLSClientEnabled() bool {
	return tidb.TLSClient != nil && tidb.TLSClient.Enabled
}

func (tidb *TiDBSpec) ShouldSeparateSlowLog() bool {
	separateSlowLog := tidb.SeparateSlowLog
	if separateSlowLog == nil {
		return defaultSeparateSlowLog
	}
	return *separateSlowLog
}

func (tidb *TiDBSpec) GetSlowLogTailerSpec() TiDBSlowLogTailerSpec {
	if tidb.SlowLogTailer == nil {
		return defaultTailerSpec
	}
	return *tidb.SlowLogTailer
}

func (tidbSvc *TiDBServiceSpec) ShouldExposeStatus() bool {
	exposeStatus := tidbSvc.ExposeStatus
	if exposeStatus == nil {
		return defaultExposeStatus
	}
	return *exposeStatus
}

func (tc *TidbCluster) GetInstanceName() string {
	labels := tc.ObjectMeta.GetLabels()
	// Keep backward compatibility for helm.
	// This introduce a hidden danger that change this label will trigger rolling-update of most of the components
	// TODO(aylei): disallow mutation of this label or adding this label with value other than the cluster name in ValidateUpdate()
	if inst, ok := labels[label.InstanceLabelKey]; ok {
		return inst
	}
	return tc.Name
}
