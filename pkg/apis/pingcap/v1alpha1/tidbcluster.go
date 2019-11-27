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
	defaultHelperImage = "busybox:1.26.2"
)

// ComponentAccessor is the interface to access component details, which respects the cluster-level properties
// and component-level overrides
type ComponentAccessor interface {
	Image() string
	ImagePullPolicy() corev1.PullPolicy
	HostNetwork() bool
	Affinity() *corev1.Affinity
	PriorityClassName() string
	NodeSelector() map[string]string
	Annotations() map[string]string
	Tolerations() []corev1.Toleration
	PodSecurityContext() *corev1.PodSecurityContext
	SchedulerName() string
}

type componentAccessorImpl struct {
	// Cluster is the TidbCluster Spec
	ClusterSpec *TidbClusterSpec

	// Cluster is the Component Spec
	ComponentSpec *ComponentSpec
}

func (a *componentAccessorImpl) Image() string {
	image := a.ComponentSpec.Image
	baseImage := a.ComponentSpec.BaseImage
	// base image takes higher priority
	if baseImage != "" {
		version := a.ComponentSpec.Version
		if version == "" {
			version = a.ClusterSpec.Version
		}
		image = fmt.Sprintf("%s:%s", baseImage, version)
	}
	return image
}

func (a *componentAccessorImpl) PodSecurityContext() *corev1.PodSecurityContext {
	return a.ComponentSpec.PodSecurityContext
}

func (a *componentAccessorImpl) ImagePullPolicy() corev1.PullPolicy {
	pp := a.ComponentSpec.ImagePullPolicy
	if pp == nil {
		pp = &a.ClusterSpec.ImagePullPolicy
	}
	return *pp
}

func (a *componentAccessorImpl) HostNetwork() bool {
	hostNetwork := a.ComponentSpec.HostNetwork
	if hostNetwork == nil {
		hostNetwork = &a.ClusterSpec.HostNetwork
	}
	return *hostNetwork
}

func (a *componentAccessorImpl) Affinity() *corev1.Affinity {
	affi := a.ComponentSpec.Affinity
	if affi == nil {
		affi = a.ClusterSpec.Affinity
	}
	return affi
}

func (a *componentAccessorImpl) PriorityClassName() string {
	pcn := a.ComponentSpec.PriorityClassName
	if pcn == "" {
		pcn = a.ClusterSpec.PriorityClassName
	}
	return pcn
}

func (a *componentAccessorImpl) SchedulerName() string {
	pcn := a.ComponentSpec.SchedulerName
	if pcn == "" {
		pcn = a.ClusterSpec.SchedulerName
	}
	return pcn
}

func (a *componentAccessorImpl) NodeSelector() map[string]string {
	sel := map[string]string{}
	for k, v := range a.ClusterSpec.NodeSelector {
		sel[k] = v
	}
	for k, v := range a.ComponentSpec.NodeSelector {
		sel[k] = v
	}
	return sel
}

func (a *componentAccessorImpl) Annotations() map[string]string {
	anno := map[string]string{}
	for k, v := range a.ClusterSpec.Annotations {
		anno[k] = v
	}
	for k, v := range a.ComponentSpec.Annotations {
		anno[k] = v
	}
	return anno
}

func (a *componentAccessorImpl) Tolerations() []corev1.Toleration {
	tols := a.ComponentSpec.Tolerations
	if len(tols) == 0 {
		tols = a.ClusterSpec.Tolerations
	}
	return tols
}

// BaseTiDBSpec returns the base spec of TiDB servers
func (tc *TidbCluster) BaseTiDBSpec() ComponentAccessor {
	return &componentAccessorImpl{&tc.Spec, &tc.Spec.TiDB.ComponentSpec}
}

// BaseTiKVSpec returns the base spec of TiKV servers
func (tc *TidbCluster) BaseTiKVSpec() ComponentAccessor {
	return &componentAccessorImpl{&tc.Spec, &tc.Spec.TiKV.ComponentSpec}
}

// BasePDSpec returns the base spec of PD servers
func (tc *TidbCluster) BasePDSpec() ComponentAccessor {
	return &componentAccessorImpl{&tc.Spec, &tc.Spec.PD.ComponentSpec}
}

// BasePumpSpec returns two results:
// 1. the base pump spec, if exists.
// 2. whether the base pump spec exists.
func (tc *TidbCluster) BasePumpSpec() (ComponentAccessor, bool) {
	if tc.Spec.Pump == nil {
		return nil, false
	}
	return &componentAccessorImpl{&tc.Spec, &tc.Spec.Pump.ComponentSpec}, true
}

func (tc *TidbCluster) HelperImage() string {
	image := tc.Spec.Helper.Image
	if image == "" {
		// for backward compatibility
		image = tc.Spec.TiDB.SlowLogTailer.Image
	}
	if image == "" {
		image = defaultHelperImage
	}
	return image
}

func (tc *TidbCluster) HelperImagePullPolicy() corev1.PullPolicy {
	pp := tc.Spec.Helper.ImagePullPolicy
	if pp == nil {
		// for backward compatibility
		pp = tc.Spec.TiDB.SlowLogTailer.ImagePullPolicy
	}
	if pp == nil {
		pp = &tc.Spec.ImagePullPolicy
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

func (tc *TidbCluster) Scheme() string {
	if tc.Spec.EnableTLSCluster {
		return "https"
	}
	return "http"
}
