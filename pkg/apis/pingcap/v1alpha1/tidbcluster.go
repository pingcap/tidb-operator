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
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
)

const (
	// defaultHelperImage is default image of helper
	defaultHelperImage        = "busybox:1.26.2"
	defaultTimeZone           = "UTC"
	defaultExposeStatus       = true
	defaultSeparateSlowLog    = true
	defaultSeparateRocksDBLog = false
	defaultSeparateRaftLog    = false
	defaultEnablePVReclaim    = false
	// defaultEvictLeaderTimeout is the timeout limit of evict leader
	defaultEvictLeaderTimeout = 1500 * time.Minute
)

var (
	defaultSlowLogTailerSpec = TiDBSlowLogTailerSpec{
		ResourceRequirements: corev1.ResourceRequirements{},
	}
	defaultLogTailerSpec = LogTailerSpec{
		ResourceRequirements: corev1.ResourceRequirements{},
	}
	defaultHelperSpec = HelperSpec{}
)

// PDImage return the image used by PD.
//
// If PD isn't specified, return empty string.
func (tc *TidbCluster) PDImage() string {
	if tc.Spec.PD == nil {
		return ""
	}

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

// PDVersion return the image version used by PD.
//
// If PD isn't specified, return empty string.
func (tc *TidbCluster) PDVersion() string {
	if tc.Spec.PD == nil {
		return ""
	}

	image := tc.PDImage()
	colonIdx := strings.LastIndexByte(image, ':')
	if colonIdx >= 0 {
		return image[colonIdx+1:]
	}

	return "latest"
}

// TiKVImage return the image used by TiKV.
//
// If TiKV isn't specified, return empty string.
func (tc *TidbCluster) TiKVImage() string {
	if tc.Spec.TiKV == nil {
		return ""
	}

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

// TiKVVersion return the image version used by TiKV.
//
// If TiKV isn't specified, return empty string.
func (tc *TidbCluster) TiKVVersion() string {
	if tc.Spec.TiKV == nil {
		return ""
	}

	image := tc.TiKVImage()
	colonIdx := strings.LastIndexByte(image, ':')
	if colonIdx >= 0 {
		return image[colonIdx+1:]
	}

	return "latest"
}

func (tc *TidbCluster) TiKVContainerPrivilege() *bool {
	if tc.Spec.TiKV == nil || tc.Spec.TiKV.Privileged == nil {
		pri := false
		return &pri
	}
	return tc.Spec.TiKV.Privileged
}

func (tc *TidbCluster) TiKVEvictLeaderTimeout() time.Duration {
	if tc.Spec.TiKV != nil && tc.Spec.TiKV.EvictLeaderTimeout != nil {
		d, err := time.ParseDuration(*tc.Spec.TiKV.EvictLeaderTimeout)
		if err == nil {
			return d
		}
	}
	return defaultEvictLeaderTimeout
}

// TiFlashImage return the image used by TiFlash.
//
// If TiFlash isn't specified, return empty string.
func (tc *TidbCluster) TiFlashImage() string {
	if tc.Spec.TiFlash == nil {
		return ""
	}

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

// TiFlashVersion returns the image version used by TiFlash.
//
// If TiFlash isn't specified, return empty string.
func (tc *TidbCluster) TiFlashVersion() string {
	if tc.Spec.TiFlash == nil {
		return ""
	}

	image := tc.TiFlashImage()
	colonIdx := strings.LastIndexByte(image, ':')
	if colonIdx >= 0 {
		return image[colonIdx+1:]
	}

	return "latest"
}

// TiCDCImage return the image used by TiCDC.
//
// If TiCDC isn't specified, return empty string.
func (tc *TidbCluster) TiCDCImage() string {
	if tc.Spec.TiCDC == nil {
		return ""
	}

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
	if tc.Spec.TiFlash == nil || tc.Spec.TiFlash.Privileged == nil {
		pri := false
		return &pri
	}
	return tc.Spec.TiFlash.Privileged
}

// TiDBImage return the image used by TiDB.
//
// If TiDB isn't specified, return empty string.
func (tc *TidbCluster) TiDBImage() string {
	if tc.Spec.TiDB == nil {
		return ""
	}

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

// PumpImage return the image used by Pump.
//
// If Pump isn't specified, return nil.
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
	if image == nil && tc.Spec.TiDB != nil {
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
	if pp == nil && tc.Spec.TiDB != nil {
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

func (tc *TidbCluster) TiKVBootStrapped() bool {
	return tc.Status.TiKV.BootStrapped
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

func (tc *TidbCluster) TiFlashScaling() bool {
	return tc.Status.TiFlash.Phase == ScalePhase
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

// PDAllPodsStarted return whether all pods of PD are started.
//
// If PD isn't specified, return false.
func (tc *TidbCluster) PDAllPodsStarted() bool {
	if tc.Spec.PD == nil {
		return false
	}
	return tc.PDStsDesiredReplicas() == tc.PDStsActualReplicas()
}

// PDAllMembersReady return whether all members of PD are ready.
//
// If PD isn't specified, return false.
func (tc *TidbCluster) PDAllMembersReady() bool {
	if tc.Spec.PD == nil {
		return false
	}

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

func (tc *TidbCluster) GetPDDeletedFailureReplicas() int32 {
	var deteledReplicas int32 = 0
	for _, failureMember := range tc.Status.PD.FailureMembers {
		if failureMember.MemberDeleted {
			deteledReplicas++
		}
	}
	return deteledReplicas
}

func (tc *TidbCluster) PDStsDesiredReplicas() int32 {
	if tc.Spec.PD == nil {
		return 0
	}
	return tc.Spec.PD.Replicas + tc.GetPDDeletedFailureReplicas()
}

func (tc *TidbCluster) PDStsActualReplicas() int32 {
	stsStatus := tc.Status.PD.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.Replicas
}

func (tc *TidbCluster) PDStsDesiredOrdinals(excludeFailover bool) sets.Int32 {
	if tc.Spec.PD == nil {
		return sets.Int32{}
	}
	replicas := tc.Spec.PD.Replicas
	if !excludeFailover {
		replicas = tc.PDStsDesiredReplicas()
	}
	return GetPodOrdinalsFromReplicasAndDeleteSlots(replicas, tc.getDeleteSlots(label.PDLabelVal))
}

// TiKVAllPodsStarted return whether all pods of TiKV are started.
//
// If TiKV isn't specified, return false.
func (tc *TidbCluster) TiKVAllPodsStarted() bool {
	if tc.Spec.TiKV == nil {
		return false
	}
	return tc.TiKVStsDesiredReplicas() == tc.TiKVStsActualReplicas()
}

// TiKVAllStoresReady return whether all stores of TiKV are ready.
//
// If TiKV isn't specified, return false.
func (tc *TidbCluster) TiKVAllStoresReady() bool {
	if tc.Spec.TiKV == nil {
		return false
	}

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
	if tc.Spec.TiKV == nil {
		return 0
	}
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
	if tc.Spec.TiKV == nil {
		return sets.Int32{}
	}
	replicas := tc.Spec.TiKV.Replicas
	if !excludeFailover {
		replicas = tc.TiKVStsDesiredReplicas()
	}
	return GetPodOrdinalsFromReplicasAndDeleteSlots(replicas, tc.getDeleteSlots(label.TiKVLabelVal))
}

// TiFlashAllPodsStarted return whether all pods of TiFlash are started.
//
// If TiFlash isn't specified, return false.
func (tc *TidbCluster) TiFlashAllPodsStarted() bool {
	if tc.Spec.TiFlash == nil {
		return false
	}
	return tc.TiFlashStsDesiredReplicas() == tc.TiFlashStsActualReplicas()
}

// TiFlashAllPodsStarted return whether all stores of TiFlash are ready.
//
// If TiFlash isn't specified, return false.
func (tc *TidbCluster) TiFlashAllStoresReady() bool {
	if tc.Spec.TiFlash == nil {
		return false
	}

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
	return GetPodOrdinalsFromReplicasAndDeleteSlots(replicas, tc.getDeleteSlots(label.TiFlashLabelVal))
}

// TiDBAllPodsStarted return whether all pods of TiDB are started.
//
// If TiDB isn't specified, return false.
func (tc *TidbCluster) TiDBAllPodsStarted() bool {
	if tc.Spec.TiDB == nil {
		return false
	}
	return tc.TiDBStsDesiredReplicas() == tc.TiDBStsActualReplicas()
}

// TiDBAllMembersReady return whether all members of TiDB are ready.
//
// If TiDB isn't specified, return false.
func (tc *TidbCluster) TiDBAllMembersReady() bool {
	if tc.Spec.TiDB == nil {
		return false
	}

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
	if tc.Spec.TiDB == nil {
		return 0
	}
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
	if tc.Spec.TiDB == nil {
		return sets.Int32{}
	}
	replicas := tc.Spec.TiDB.Replicas
	if !excludeFailover {
		replicas = tc.TiDBStsDesiredReplicas()
	}
	return GetPodOrdinalsFromReplicasAndDeleteSlots(replicas, tc.getDeleteSlots(label.TiDBLabelVal))
}

// PDIsAvailable return whether PD is available.
//
// If PD isn't specified, return true.
func (tc *TidbCluster) PDIsAvailable() bool {
	if tc.Spec.PD == nil {
		return true
	}

	lowerLimit := (tc.Spec.PD.Replicas+int32(len(tc.Status.PD.PeerMembers)))/2 + 1
	if int32(len(tc.Status.PD.Members)+len(tc.Status.PD.PeerMembers)) < lowerLimit {
		return false
	}

	var availableNum int32
	for _, pdMember := range tc.Status.PD.Members {
		if pdMember.Health {
			availableNum++
		}
	}

	var peerAvailableNum int32
	for _, pdMember := range tc.Status.PD.PeerMembers {
		if pdMember.Health {
			peerAvailableNum++
		}
	}

	availableNum += peerAvailableNum
	if availableNum < lowerLimit {
		return false
	}

	if tc.Status.PD.StatefulSet == nil || tc.Status.PD.StatefulSet.ReadyReplicas+peerAvailableNum < lowerLimit {
		return false
	}

	return true
}

func (tc *TidbCluster) TiKVIsAvailable() bool {
	var lowerLimit int32 = 1
	if int32(len(tc.Status.TiKV.Stores)+len(tc.Status.TiKV.PeerStores)) < lowerLimit {
		return false
	}

	var availableNum int32
	for _, store := range tc.Status.TiKV.Stores {
		if store.State == TiKVStateUp {
			availableNum++
		}
	}

	var peerAvailableNum int32
	for _, store := range tc.Status.TiKV.PeerStores {
		if store.State == TiKVStateUp {
			peerAvailableNum++
		}
	}

	availableNum += peerAvailableNum

	if availableNum < lowerLimit {
		return false
	}

	if tc.Status.TiKV.StatefulSet == nil || tc.Status.TiKV.StatefulSet.ReadyReplicas+peerAvailableNum < lowerLimit {
		return false
	}

	return true
}

func (tc *TidbCluster) PumpIsAvailable() bool {
	lowerLimit := 1
	if len(tc.Status.Pump.Members) < lowerLimit {
		return false
	}

	availableNum := 0
	for _, member := range tc.Status.Pump.Members {
		if member.State == PumpStateOnline {
			availableNum++
		}
	}

	return availableNum >= lowerLimit
}

func (tc *TidbCluster) GetClusterID() string {
	return tc.Status.ClusterID
}

func (tc *TidbCluster) IsTLSClusterEnabled() bool {
	return tc.Spec.TLSCluster != nil && tc.Spec.TLSCluster.Enabled
}

func (tc *TidbCluster) NeedToSyncTiDBInitializer() bool {
	return tc.Spec.TiDB != nil && tc.Spec.TiDB.Initializer != nil && tc.Spec.TiDB.Initializer.CreatePassword && tc.Status.TiDB.PasswordInitialized == nil
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
	var binlogEnabled *bool
	if tc.Spec.TiDB != nil {
		binlogEnabled = tc.Spec.TiDB.BinlogEnabled
	}

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
		return defaultSlowLogTailerSpec
	}
	return *tidb.SlowLogTailer
}

// GetServicePort returns the service port for tidb
func (tidb *TiDBSpec) GetServicePort() int32 {
	port := DefaultTiDBServicePort
	if tidb.Service != nil && tidb.Service.Port != nil {
		port = *tidb.Service.Port
	}
	return port
}

func (tikv *TiKVSpec) ShouldSeparateRocksDBLog() bool {
	separateRocksDBLog := tikv.SeparateRocksDBLog
	if separateRocksDBLog == nil {
		return defaultSeparateRocksDBLog
	}
	return *separateRocksDBLog
}

func (tikv *TiKVSpec) ShouldSeparateRaftLog() bool {
	separateRaftLog := tikv.SeparateRaftLog
	if separateRaftLog == nil {
		return defaultSeparateRaftLog
	}
	return *separateRaftLog
}

func (tikv *TiKVSpec) GetLogTailerSpec() LogTailerSpec {
	if tikv.LogTailer == nil {
		return defaultLogTailerSpec
	}
	return *tikv.LogTailer
}

func (tikv *TiKVSpec) GetRecoverByUID() types.UID {
	if tikv.Failover == nil {
		return ""
	}
	return tikv.Failover.RecoverByUID
}

func (tiflash *TiFlashSpec) GetRecoverByUID() types.UID {
	if tiflash.Failover == nil {
		return ""
	}
	return tiflash.Failover.RecoverByUID
}

func (tidbSvc *TiDBServiceSpec) ShouldExposeStatus() bool {
	exposeStatus := tidbSvc.ExposeStatus
	if exposeStatus == nil {
		return defaultExposeStatus
	}
	return *exposeStatus
}

// GetMySQLNodePort returns the mysqlNodePort config in spec.tidb.service
func (tidbSvc *TiDBServiceSpec) GetMySQLNodePort() int32 {
	mysqlNodePort := tidbSvc.MySQLNodePort
	if mysqlNodePort == nil {
		return 0
	}
	return int32(*mysqlNodePort)
}

// GetStatusNodePort returns the statusNodePort config in spec.tidb.service
func (tidbSvc *TiDBServiceSpec) GetStatusNodePort() int32 {
	statusNodePort := tidbSvc.StatusNodePort
	if statusNodePort == nil {
		return 0
	}
	return int32(*statusNodePort)
}

// GetPort returns the service port name in spec.tidb.service
func (tidbSvc *TiDBServiceSpec) GetPortName() string {
	portName := "mysql-client"
	if tidbSvc.PortName != nil {
		portName = *tidbSvc.PortName
	}
	return portName
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

// TODO: We Should better do not specified the default value ourself if user not specified the item.
func (tc *TidbCluster) TiCDCTimezone() string {
	if tc.Spec.TiCDC != nil && tc.Spec.TiCDC.Config != nil {
		if v := tc.Spec.TiCDC.Config.Get("tz"); v != nil {
			tz, err := v.AsString()
			if err != nil {
				klog.Warningf("'%s/%s' incorrect tz type %v", tc.Namespace, tc.Name, v.Interface())
			} else {
				return tz
			}
		}

	}
	return tc.Timezone()
}

func (tc *TidbCluster) TiCDCGCTTL() int32 {
	if tc.Spec.TiCDC != nil && tc.Spec.TiCDC.Config != nil {
		if v := tc.Spec.TiCDC.Config.Get("gc-ttl"); v != nil {
			ttl, err := v.AsInt()
			if err != nil {
				klog.Warningf("'%s/%s' incorrect gc-ttl type %v", tc.Namespace, tc.Name, v.Interface())
			} else {
				return int32(ttl)
			}
		}
	}

	return 86400
}

func (tc *TidbCluster) TiCDCLogFile() string {
	if tc.Spec.TiCDC != nil && tc.Spec.TiCDC.Config != nil {
		if v := tc.Spec.TiCDC.Config.Get("log-file"); v != nil {
			file, err := v.AsString()
			if err != nil {
				klog.Warningf("'%s/%s' incorrect log-file type %v", tc.Namespace, tc.Name, v.Interface())
			} else {
				return file
			}
		}
	}

	return ""
}

func (tc *TidbCluster) TiCDCLogLevel() string {
	if tc.Spec.TiCDC != nil && tc.Spec.TiCDC.Config != nil {
		if v := tc.Spec.TiCDC.Config.Get("log-level"); v != nil {
			level, err := v.AsString()
			if err != nil {
				klog.Warningf("'%s/%s' incorrect log-level type %v", tc.Namespace, tc.Name, v.Interface())
			} else {
				return level
			}
		}
	}

	return "info"
}

func (tc *TidbCluster) Heterogeneous() bool {
	return tc.Spec.Cluster != nil && len(tc.Spec.Cluster.Name) > 0
}

func (tc *TidbCluster) WithoutLocalPD() bool {
	return tc.Spec.PD == nil
}

func (tc *TidbCluster) WithoutLocalTiDB() bool {
	return tc.Spec.TiDB == nil
}

func (tc *TidbCluster) AcrossK8s() bool {
	return tc.Spec.AcrossK8s
}

// IsComponentVolumeResizing returns true if any volume of component is resizing.
func (tc *TidbCluster) IsComponentVolumeResizing(compType MemberType) bool {
	comps := ComponentStatusFromTC(tc)
	for _, comp := range comps {
		if comp.GetMemberType() == compType {
			conds := comp.GetConditions()
			return meta.IsStatusConditionTrue(conds, ComponentVolumeResizing)
		}
	}
	return false
}
