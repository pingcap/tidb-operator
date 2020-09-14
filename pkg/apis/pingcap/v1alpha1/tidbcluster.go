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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// defaultHelperImage is default image of helper
	defaultHelperImage     = "busybox:1.26.2"
	defaultTimeZone        = "UTC"
	defaultExposeStatus    = true
	defaultSeparateSlowLog = true
	defaultEnablePVReclaim = false
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
		if *version == "" {
			image = baseImage
		} else {
			image = fmt.Sprintf("%s:%s", baseImage, *version)
		}
	}
	return image
}

func (tc *TidbCluster) PDVersion() string {
	image := tc.PDImage()
	colonIdx := strings.LastIndexByte(image, ':')
	if colonIdx >= 0 {
		return image[colonIdx+1:]
	}

	return "latest"
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
		if *version == "" {
			image = baseImage
		} else {
			image = fmt.Sprintf("%s:%s", baseImage, *version)
		}
	}
	return image
}

func (tc *TidbCluster) TiKVVersion() string {
	image := tc.TiKVImage()
	colonIdx := strings.LastIndexByte(image, ':')
	if colonIdx >= 0 {
		return image[colonIdx+1:]
	}

	return "latest"
}

func (tc *TidbCluster) TiKVContainerPrivilege() *bool {
	if tc.Spec.TiKV.Privileged == nil {
		pri := false
		return &pri
	}
	return tc.Spec.TiKV.Privileged
}

func (tc *TidbCluster) TiFlashImage() string {
	image := tc.Spec.TiFlash.Image
	baseImage := tc.Spec.TiFlash.BaseImage
	// base image takes higher priority
	if baseImage != "" {
		version := tc.Spec.TiFlash.Version
		if version == nil {
			version = &tc.Spec.Version
		}
		if *version == "" {
			image = baseImage
		} else {
			image = fmt.Sprintf("%s:%s", baseImage, *version)
		}
	}
	return image
}

func (tc *TidbCluster) TiCDCImage() string {
	image := tc.Spec.TiCDC.Image
	baseImage := tc.Spec.TiCDC.BaseImage
	// base image takes higher priority
	if baseImage != "" {
		version := tc.Spec.TiCDC.Version
		if version == nil {
			version = &tc.Spec.Version
		}
		if *version == "" {
			image = baseImage
		} else {
			image = fmt.Sprintf("%s:%s", baseImage, *version)
		}
	}
	return image
}

func (tc *TidbCluster) TiFlashContainerPrivilege() *bool {
	if tc.Spec.TiFlash.Privileged == nil {
		pri := false
		return &pri
	}
	return tc.Spec.TiFlash.Privileged
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
		if *version == "" {
			image = baseImage
		} else {
			image = fmt.Sprintf("%s:%s", baseImage, *version)
		}
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
		if *version == "" {
			image = baseImage
		} else {
			image = fmt.Sprintf("%s:%s", baseImage, *version)
		}
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

func (tc *TidbCluster) PDScaling() bool {
	return tc.Status.PD.Phase == ScalePhase
}

func (tc *TidbCluster) TiKVUpgrading() bool {
	return tc.Status.TiKV.Phase == UpgradePhase
}

func (tc *TidbCluster) TiKVScaling() bool {
	return tc.Status.TiKV.Phase == ScalePhase
}

func (tc *TidbCluster) TiDBUpgrading() bool {
	return tc.Status.TiDB.Phase == UpgradePhase
}

func (tc *TidbCluster) TiDBScaling() bool {
	return tc.Status.TiDB.Phase == ScalePhase
}

func (tc *TidbCluster) TiFlashUpgrading() bool {
	return tc.Status.TiFlash.Phase == UpgradePhase
}

func (tc *TidbCluster) getDeleteSlots(component string) (deleteSlots sets.Int32) {
	deleteSlots = sets.NewInt32()
	annotations := tc.GetAnnotations()
	if annotations == nil {
		return deleteSlots
	}
	var key string
	if component == label.PDLabelVal {
		key = label.AnnPDDeleteSlots
	} else if component == label.TiDBLabelVal {
		key = label.AnnTiDBDeleteSlots
	} else if component == label.TiKVLabelVal {
		key = label.AnnTiKVDeleteSlots
	} else if component == label.TiFlashLabelVal {
		key = label.AnnTiFlashDeleteSlots
	} else {
		return
	}
	value, ok := annotations[key]
	if !ok {
		return
	}
	var slice []int32
	err := json.Unmarshal([]byte(value), &slice)
	if err != nil {
		return
	}
	deleteSlots.Insert(slice...)
	return
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

func (tc *TidbCluster) PDStsDesiredOrdinals(excludeFailover bool) sets.Int32 {
	replicas := tc.Spec.PD.Replicas
	if !excludeFailover {
		replicas = tc.PDStsDesiredReplicas()
	}
	return helper.GetPodOrdinalsFromReplicasAndDeleteSlots(replicas, tc.getDeleteSlots(label.PDLabelVal))
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

func (tc *TidbCluster) TiKVStsDesiredOrdinals(excludeFailover bool) sets.Int32 {
	replicas := tc.Spec.TiKV.Replicas
	if !excludeFailover {
		replicas = tc.TiKVStsDesiredReplicas()
	}
	return helper.GetPodOrdinalsFromReplicasAndDeleteSlots(replicas, tc.getDeleteSlots(label.TiKVLabelVal))
}

func (tc *TidbCluster) TiFlashAllPodsStarted() bool {
	return tc.TiFlashStsDesiredReplicas() == tc.TiFlashStsActualReplicas()
}

func (tc *TidbCluster) TiFlashAllStoresReady() bool {
	if int(tc.TiFlashStsDesiredReplicas()) != len(tc.Status.TiFlash.Stores) {
		return false
	}

	for _, store := range tc.Status.TiFlash.Stores {
		if store.State != TiKVStateUp {
			return false
		}
	}

	return true
}

func (tc *TidbCluster) TiFlashStsDesiredReplicas() int32 {
	if tc.Spec.TiFlash == nil {
		return 0
	}
	return tc.Spec.TiFlash.Replicas + int32(len(tc.Status.TiFlash.FailureStores))
}

func (tc *TidbCluster) TiCDCDeployDesiredReplicas() int32 {
	if tc.Spec.TiCDC == nil {
		return 0
	}

	return tc.Spec.TiCDC.Replicas
}

func (tc *TidbCluster) TiFlashStsActualReplicas() int32 {
	stsStatus := tc.Status.TiFlash.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.Replicas
}

func (tc *TidbCluster) TiFlashStsDesiredOrdinals(excludeFailover bool) sets.Int32 {
	if tc.Spec.TiFlash == nil {
		return sets.Int32{}
	}
	replicas := tc.Spec.TiFlash.Replicas
	if !excludeFailover {
		replicas = tc.TiFlashStsDesiredReplicas()
	}
	return helper.GetPodOrdinalsFromReplicasAndDeleteSlots(replicas, tc.getDeleteSlots(label.TiFlashLabelVal))
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

func (tc *TidbCluster) TiDBStsDesiredOrdinals(excludeFailover bool) sets.Int32 {
	replicas := tc.Spec.TiDB.Replicas
	if !excludeFailover {
		replicas = tc.TiDBStsDesiredReplicas()
	}
	return helper.GetPodOrdinalsFromReplicasAndDeleteSlots(replicas, tc.getDeleteSlots(label.TiDBLabelVal))
}

func (tc *TidbCluster) PDIsAvailable() bool {
	if tc.Spec.PD == nil {
		return true
	}
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

func (tc *TidbCluster) PumpIsAvailable() bool {
	var lowerLimit int32 = 1
	if tc.Status.Pump.StatefulSet == nil || tc.Status.Pump.StatefulSet.ReadyReplicas < lowerLimit {
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

func (tidbSvc *TiDBServiceSpec) GetMySQLNodePort() int32 {
	mysqlNodePort := tidbSvc.MySQLNodePort
	if mysqlNodePort == nil {
		return 0
	}
	return int32(*mysqlNodePort)
}

func (tidbSvc *TiDBServiceSpec) GetStatusNodePort() int32 {
	statusNodePort := tidbSvc.StatusNodePort
	if statusNodePort == nil {
		return 0
	}
	return int32(*statusNodePort)
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

func (tc *TidbCluster) SkipTLSWhenConnectTiDB() bool {
	_, ok := tc.Annotations[label.AnnSkipTLSWhenConnectTiDB]
	return ok
}

func (tc *TidbCluster) TiCDCTimezone() string {
	if tc.Spec.TiCDC != nil {
		v := tc.Spec.TiCDC.GenericConfig.Get("timezone")
		if v != nil {
			return v.AsString()
		}
	}

	return tc.Timezone()
}

func (tc *TidbCluster) TiCDCGCTTL() int32 {
	if tc.Spec.TiCDC != nil {
		v := tc.Spec.TiCDC.GenericConfig.Get("gcTTL")
		if v != nil {
			return int32(v.AsInt())
		}
	}

	return 86400
}

func (tc *TidbCluster) TiCDCLogFile() string {
	if tc.Spec.TiCDC != nil {
		v := tc.Spec.TiCDC.GenericConfig.Get("logFile")
		if v != nil {
			return v.AsString()
		}
	}

	return "/dev/stderr"
}

func (tc *TidbCluster) TiCDCLogLevel() string {
	if tc.Spec.TiCDC != nil {
		v := tc.Spec.TiCDC.GenericConfig.Get("logLevel")
		if v != nil {
			return v.AsString()
		}
	}

	return "info"
}

func (tc *TidbCluster) IsHeterogeneous() bool {
	return tc.Spec.Cluster != nil && len(tc.Spec.Cluster.Name) > 0 && tc.Spec.PD == nil
}
