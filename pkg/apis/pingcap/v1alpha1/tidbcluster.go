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
	if tc.Spec.Helper == nil {
		return defaultHelperImage
	}
	image := tc.Spec.Helper.Image
	if image == nil && tc.Spec.TiDB.SlowLogTailer != nil {
		// for backward compatibility
		image = tc.Spec.TiDB.SlowLogTailer.Image
	}
	if image == nil {
		return defaultHelperImage
	}
	return *image
}

func (tc *TidbCluster) HelperImagePullPolicy() corev1.PullPolicy {
	if tc.Spec.Helper == nil {
		return tc.Spec.ImagePullPolicy
	}
	pp := tc.Spec.Helper.ImagePullPolicy
	if pp == nil && tc.Spec.TiDB.SlowLogTailer != nil {
		// for backward compatibility
		pp = tc.Spec.TiDB.SlowLogTailer.ImagePullPolicy
	}
	if pp == nil {
		return tc.Spec.ImagePullPolicy
	}
	return *pp
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
	enableTLCluster := tc.Spec.EnableTLSCluster
	if enableTLCluster == nil {
		return defaultEnableTLSCluster
	}
	return *enableTLCluster
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
	enableTLSClient := tidb.EnableTLSClient
	if enableTLSClient == nil {
		return defaultEnableTLSClient
	}
	return *enableTLSClient
}

func (tidb *TiDBSpec) ShouldSeparateSlowLog() bool {
	separateSlowLog := tidb.SeparateSlowLog
	if separateSlowLog == nil {
		return defaultEnableTLSClient
	}
	return *separateSlowLog
}

func (tidbSvc *TiDBServiceSpec) ShouldExposeStatus() bool {
	exposeStatus := tidbSvc.ExposeStatus
	if exposeStatus == nil {
		return defaultExposeStatus
	}
	return *exposeStatus
}
