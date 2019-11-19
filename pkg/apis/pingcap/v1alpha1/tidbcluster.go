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
	tols := a.ClusterSpec.Tolerations
	tols = append(tols, a.ComponentSpec.Tolerations...)
	return tols
}

func (tc *TidbCluster) BaseTiDBSpec() ComponentAccessor {
	return &componentAccessorImpl{&tc.Spec, &tc.Spec.TiDB.ComponentSpec}
}

func (tc *TidbCluster) BaseTiKVSpec() ComponentAccessor {
	return &componentAccessorImpl{&tc.Spec, &tc.Spec.TiKV.ComponentSpec}
}

func (tc *TidbCluster) BasePDSpec() ComponentAccessor {
	return &componentAccessorImpl{&tc.Spec, &tc.Spec.PD.ComponentSpec}
}

func (tc *TidbCluster) BasePumpSpec() ComponentAccessor {
	return &componentAccessorImpl{&tc.Spec, &tc.Spec.Pump.ComponentSpec}
}

func (tc *TidbCluster) HelperImage() string {
	image := tc.Spec.Helper.Image
	if image == "" {
		// for backward compatiability
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
		// for backward compatiability
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
	return tc.PDRealReplicas() == tc.Status.PD.StatefulSet.Replicas
}

func (tc *TidbCluster) PDAllMembersReady() bool {
	if int(tc.PDRealReplicas()) != len(tc.Status.PD.Members) {
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

func (tc *TidbCluster) PDRealReplicas() int32 {
	return tc.Spec.PD.Replicas + int32(len(tc.Status.PD.FailureMembers))
}

func (tc *TidbCluster) TiKVAllPodsStarted() bool {
	return tc.TiKVRealReplicas() == tc.Status.TiKV.StatefulSet.Replicas
}

func (tc *TidbCluster) TiKVAllStoresReady() bool {
	if int(tc.TiKVRealReplicas()) != len(tc.Status.TiKV.Stores) {
		return false
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.State != TiKVStateUp {
			return false
		}
	}

	return true
}

func (tc *TidbCluster) TiKVRealReplicas() int32 {
	return tc.Spec.TiKV.Replicas + int32(len(tc.Status.TiKV.FailureStores))
}

func (tc *TidbCluster) TiDBAllPodsStarted() bool {
	return tc.TiDBRealReplicas() == tc.Status.TiDB.StatefulSet.Replicas
}

func (tc *TidbCluster) TiDBAllMembersReady() bool {
	if int(tc.TiDBRealReplicas()) != len(tc.Status.TiDB.Members) {
		return false
	}

	for _, member := range tc.Status.TiDB.Members {
		if !member.Health {
			return false
		}
	}

	return true
}

func (tc *TidbCluster) TiDBRealReplicas() int32 {
	return tc.Spec.TiDB.Replicas + int32(len(tc.Status.TiDB.FailureMembers))
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
