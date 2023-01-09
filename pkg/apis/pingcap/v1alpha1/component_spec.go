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

package v1alpha1

import (
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultHostNetwork = false
)

var (
	_ Cluster = &TidbCluster{}
	_ Cluster = &DMCluster{}
)

type Cluster interface {
	metav1.Object

	// AllComponentSpec return all component specs
	AllComponentSpec() []ComponentAccessor
	// AllComponentStatus return all component status
	AllComponentStatus() []ComponentStatus
	// ComponentSpec return a component spec, return nil if not exist
	ComponentSpec(typ MemberType) ComponentAccessor
	// ComponentStatus return a component status, return nil if not exist
	ComponentStatus(typ MemberType) ComponentStatus
	// ComponentIsSuspending return true if the component's phase is `Suspend`
	ComponentIsSuspending(typ MemberType) bool
	// ComponentIsSuspended return true if the component's phase is `Suspend` and all resources is suspended
	ComponentIsSuspended(typ MemberType) bool
	// ComponentIsNormal return true if the component's phase is `Normal`
	ComponentIsNormal(typ MemberType) bool
}

// ComponentAccessor is the interface to access component details, which respects the cluster-level properties
// and component-level overrides
type ComponentAccessor interface {
	MemberType() MemberType
	ImagePullPolicy() corev1.PullPolicy
	ImagePullSecrets() []corev1.LocalObjectReference
	HostNetwork() bool
	Affinity() *corev1.Affinity
	PriorityClassName() *string
	NodeSelector() map[string]string
	Labels() map[string]string
	Annotations() map[string]string
	Tolerations() []corev1.Toleration
	PodSecurityContext() *corev1.PodSecurityContext
	SchedulerName() string
	DnsPolicy() corev1.DNSPolicy
	ConfigUpdateStrategy() ConfigUpdateStrategy
	BuildPodSpec() corev1.PodSpec
	Env() []corev1.EnvVar
	EnvFrom() []corev1.EnvFromSource
	AdditionalContainers() []corev1.Container
	InitContainers() []corev1.Container
	AdditionalVolumes() []corev1.Volume
	AdditionalVolumeMounts() []corev1.VolumeMount
	TerminationGracePeriodSeconds() *int64
	StatefulSetUpdateStrategy() apps.StatefulSetUpdateStrategyType
	PodManagementPolicy() apps.PodManagementPolicyType
	TopologySpreadConstraints() []corev1.TopologySpreadConstraint
	SuspendAction() *SuspendAction
}

func (tc *TidbCluster) AllComponentSpec() []ComponentAccessor {
	components := []ComponentAccessor{}
	components = append(components, tc.BaseDiscoverySpec())
	if tc.Spec.PD != nil {
		components = append(components, tc.BasePDSpec())
	}
	if tc.Spec.TiDB != nil {
		components = append(components, tc.BaseTiDBSpec())
	}
	if tc.Spec.TiKV != nil {
		components = append(components, tc.BaseTiKVSpec())
	}
	if tc.Spec.TiFlash != nil {
		components = append(components, tc.BaseTiFlashSpec())
	}
	if tc.Spec.TiCDC != nil {
		components = append(components, tc.BaseTiCDCSpec())
	}
	if tc.Spec.Pump != nil {
		components = append(components, tc.BasePumpSpec())
	}
	if tc.Spec.TiProxy != nil {
		components = append(components, tc.BaseTiProxySpec())
	}
	return components
}

func (tc *TidbCluster) ComponentSpec(typ MemberType) ComponentAccessor {
	components := tc.AllComponentSpec()
	for _, component := range components {
		if component.MemberType() == typ {
			return component
		}
	}
	return nil
}

func (dc *DMCluster) AllComponentSpec() []ComponentAccessor {
	components := []ComponentAccessor{}
	components = append(components, dc.BaseDiscoverySpec())
	components = append(components, dc.BaseMasterSpec())
	if dc.Spec.Worker != nil {
		components = append(components, dc.BaseWorkerSpec())
	}
	return components
}

func (dc *DMCluster) ComponentSpec(typ MemberType) ComponentAccessor {
	components := dc.AllComponentSpec()
	for _, component := range components {
		if component.MemberType() == typ {
			return component
		}
	}
	return nil
}

func (ngm *TidbNGMonitoring) AllComponentSpec() []ComponentAccessor {
	components := []ComponentAccessor{}
	components = append(components, ngm.BaseNGMonitoringSpec())
	return components
}

func (ngm *TidbNGMonitoring) ComponentSpec(typ MemberType) ComponentAccessor {
	components := ngm.AllComponentSpec()
	for _, component := range components {
		if component.MemberType() == typ {
			return component
		}
	}
	return nil
}

type componentAccessorImpl struct {
	component MemberType
	name      string
	kind      string

	imagePullPolicy           corev1.PullPolicy
	imagePullSecrets          []corev1.LocalObjectReference
	hostNetwork               *bool
	affinity                  *corev1.Affinity
	priorityClassName         *string
	schedulerName             string
	clusterNodeSelector       map[string]string
	clusterAnnotations        map[string]string
	clusterLabels             map[string]string
	tolerations               []corev1.Toleration
	dnsConfig                 *corev1.PodDNSConfig
	dnsPolicy                 corev1.DNSPolicy
	configUpdateStrategy      ConfigUpdateStrategy
	statefulSetUpdateStrategy apps.StatefulSetUpdateStrategyType
	podManagementPolicy       apps.PodManagementPolicyType
	podSecurityContext        *corev1.PodSecurityContext
	topologySpreadConstraints []TopologySpreadConstraint
	suspendAction             *SuspendAction

	// CommonComponentSpec is the Component Spec
	CommonComponentSpec *CommonComponentSpec
}

func (a *componentAccessorImpl) MemberType() MemberType {
	return a.component
}

func (a *componentAccessorImpl) StatefulSetUpdateStrategy() apps.StatefulSetUpdateStrategyType {
	if a.CommonComponentSpec == nil || len(a.CommonComponentSpec.StatefulSetUpdateStrategy) == 0 {
		if len(a.statefulSetUpdateStrategy) == 0 {
			return apps.RollingUpdateStatefulSetStrategyType
		}
		return a.statefulSetUpdateStrategy
	}
	return a.CommonComponentSpec.StatefulSetUpdateStrategy
}

func (a *componentAccessorImpl) PodManagementPolicy() apps.PodManagementPolicyType {
	policy := apps.ParallelPodManagement
	if a.CommonComponentSpec != nil && len(a.CommonComponentSpec.PodManagementPolicy) != 0 {
		policy = a.CommonComponentSpec.PodManagementPolicy
	} else if len(a.podManagementPolicy) != 0 {
		policy = a.podManagementPolicy
	}

	// unified podManagementPolicy check to avoid check everywhere
	if policy == apps.OrderedReadyPodManagement {
		return apps.OrderedReadyPodManagement
	}
	return apps.ParallelPodManagement
}

func (a *componentAccessorImpl) PodSecurityContext() *corev1.PodSecurityContext {
	if a.CommonComponentSpec == nil || a.CommonComponentSpec.PodSecurityContext == nil {
		return a.podSecurityContext
	}
	return a.CommonComponentSpec.PodSecurityContext
}

func (a *componentAccessorImpl) ImagePullPolicy() corev1.PullPolicy {
	if a.CommonComponentSpec == nil || a.CommonComponentSpec.ImagePullPolicy == nil {
		return a.imagePullPolicy
	}
	return *a.CommonComponentSpec.ImagePullPolicy
}

func (a *componentAccessorImpl) ImagePullSecrets() []corev1.LocalObjectReference {
	if a.CommonComponentSpec == nil || len(a.CommonComponentSpec.ImagePullSecrets) == 0 {
		return a.imagePullSecrets
	}
	return a.CommonComponentSpec.ImagePullSecrets
}

func (a *componentAccessorImpl) HostNetwork() bool {
	if a.CommonComponentSpec == nil || a.CommonComponentSpec.HostNetwork == nil {
		if a.hostNetwork == nil {
			return defaultHostNetwork
		}
		return *a.hostNetwork
	}
	return *a.CommonComponentSpec.HostNetwork
}

func (a *componentAccessorImpl) Affinity() *corev1.Affinity {
	if a.CommonComponentSpec == nil || a.CommonComponentSpec.Affinity == nil {
		return a.affinity
	}
	return a.CommonComponentSpec.Affinity
}

func (a *componentAccessorImpl) PriorityClassName() *string {
	if a.CommonComponentSpec == nil || a.CommonComponentSpec.PriorityClassName == nil {
		return a.priorityClassName
	}
	return a.CommonComponentSpec.PriorityClassName
}

func (a *componentAccessorImpl) SchedulerName() string {
	if a.CommonComponentSpec == nil || a.CommonComponentSpec.SchedulerName == nil {
		return a.schedulerName
	}
	return *a.CommonComponentSpec.SchedulerName
}

func (a *componentAccessorImpl) NodeSelector() map[string]string {
	sel := map[string]string{}
	for k, v := range a.clusterNodeSelector {
		sel[k] = v
	}
	if a.CommonComponentSpec != nil {
		for k, v := range a.CommonComponentSpec.NodeSelector {
			sel[k] = v
		}
	}
	return sel
}

func (a *componentAccessorImpl) Labels() map[string]string {
	l := map[string]string{}
	for k, v := range a.clusterLabels {
		l[k] = v
	}
	if a.CommonComponentSpec != nil {
		for k, v := range a.CommonComponentSpec.Labels {
			l[k] = v
		}
	}
	return l
}

func (a *componentAccessorImpl) Annotations() map[string]string {
	anno := map[string]string{}
	for k, v := range a.clusterAnnotations {
		anno[k] = v
	}
	if a.CommonComponentSpec != nil {
		for k, v := range a.CommonComponentSpec.Annotations {
			anno[k] = v
		}
	}
	return anno
}

func (a *componentAccessorImpl) Tolerations() []corev1.Toleration {
	if a.CommonComponentSpec == nil || len(a.CommonComponentSpec.Tolerations) == 0 {
		return a.tolerations
	}
	return a.CommonComponentSpec.Tolerations
}

func (a *componentAccessorImpl) DnsPolicy() corev1.DNSPolicy {
	if a.CommonComponentSpec != nil && a.CommonComponentSpec.DNSPolicy != "" {
		return a.CommonComponentSpec.DNSPolicy
	}

	if a.dnsPolicy != "" {
		return a.dnsPolicy
	}

	if a.HostNetwork() {
		return corev1.DNSClusterFirstWithHostNet
	}
	return corev1.DNSClusterFirst // same as kubernetes default
}

func (a *componentAccessorImpl) DNSConfig() *corev1.PodDNSConfig {
	if a.CommonComponentSpec == nil || a.CommonComponentSpec.DNSConfig == nil {
		return a.dnsConfig
	}
	return a.CommonComponentSpec.DNSConfig
}

func (a *componentAccessorImpl) ConfigUpdateStrategy() ConfigUpdateStrategy {
	// defaulting logic will set a default value for configUpdateStrategy field, but if the
	// object is created in early version without this field being set, we should set a safe default
	if a.CommonComponentSpec == nil || a.CommonComponentSpec.ConfigUpdateStrategy == nil {
		if a.configUpdateStrategy != "" {
			return a.configUpdateStrategy
		}
		return ConfigUpdateStrategyInPlace
	}
	if *a.CommonComponentSpec.ConfigUpdateStrategy == "" {
		return ConfigUpdateStrategyInPlace
	}
	return *a.CommonComponentSpec.ConfigUpdateStrategy
}

func (a *componentAccessorImpl) BuildPodSpec() corev1.PodSpec {
	spec := corev1.PodSpec{
		SchedulerName:             a.SchedulerName(),
		Affinity:                  a.Affinity(),
		NodeSelector:              a.NodeSelector(),
		HostNetwork:               a.HostNetwork(),
		RestartPolicy:             corev1.RestartPolicyAlways,
		Tolerations:               a.Tolerations(),
		SecurityContext:           a.PodSecurityContext(),
		TopologySpreadConstraints: a.TopologySpreadConstraints(),
		DNSPolicy:                 a.DnsPolicy(),
		DNSConfig:                 a.DNSConfig(),
	}
	if a.PriorityClassName() != nil {
		spec.PriorityClassName = *a.PriorityClassName()
	}
	if a.ImagePullSecrets() != nil {
		spec.ImagePullSecrets = a.ImagePullSecrets()
	}
	if a.TerminationGracePeriodSeconds() != nil {
		spec.TerminationGracePeriodSeconds = a.TerminationGracePeriodSeconds()
	}
	return spec
}

func (a *componentAccessorImpl) Env() []corev1.EnvVar {
	if a.CommonComponentSpec == nil {
		return nil
	}
	return a.CommonComponentSpec.Env
}

func (a *componentAccessorImpl) EnvFrom() []corev1.EnvFromSource {
	if a.CommonComponentSpec == nil {
		return nil
	}
	return a.CommonComponentSpec.EnvFrom
}

func (a *componentAccessorImpl) InitContainers() []corev1.Container {
	if a.CommonComponentSpec == nil {
		return nil
	}
	return a.CommonComponentSpec.InitContainers
}

func (a *componentAccessorImpl) AdditionalContainers() []corev1.Container {
	if a.CommonComponentSpec == nil {
		return nil
	}
	return a.CommonComponentSpec.AdditionalContainers
}

func (a *componentAccessorImpl) AdditionalVolumes() []corev1.Volume {
	if a.CommonComponentSpec == nil {
		return nil
	}
	return a.CommonComponentSpec.AdditionalVolumes
}

func (a *componentAccessorImpl) AdditionalVolumeMounts() []corev1.VolumeMount {
	if a.CommonComponentSpec == nil {
		return nil
	}
	return a.CommonComponentSpec.AdditionalVolumeMounts
}

func (a *componentAccessorImpl) TerminationGracePeriodSeconds() *int64 {
	if a.CommonComponentSpec == nil {
		return nil
	}
	return a.CommonComponentSpec.TerminationGracePeriodSeconds
}

func (a *componentAccessorImpl) TopologySpreadConstraints() []corev1.TopologySpreadConstraint {
	tscs := a.topologySpreadConstraints
	if a.CommonComponentSpec != nil && len(a.CommonComponentSpec.TopologySpreadConstraints) > 0 {
		tscs = a.CommonComponentSpec.TopologySpreadConstraints
	}

	if len(tscs) == 0 {
		return nil
	}

	ptscs := make([]corev1.TopologySpreadConstraint, 0, len(tscs))
	for _, tsc := range tscs {
		ptsc := corev1.TopologySpreadConstraint{
			MaxSkew:           1,
			TopologyKey:       tsc.TopologyKey,
			WhenUnsatisfiable: corev1.DoNotSchedule,
		}
		componentLabelVal := getComponentLabelValue(a.component)
		var l label.Label
		switch a.kind {
		case TiDBClusterKind:
			l = label.New()
		case DMClusterKind:
			l = label.NewDM()
		}
		l[label.ComponentLabelKey] = componentLabelVal
		l[label.InstanceLabelKey] = a.name
		ptsc.LabelSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string(l),
		}
		ptscs = append(ptscs, ptsc)
	}
	return ptscs
}

func (a *componentAccessorImpl) SuspendAction() *SuspendAction {
	action := a.suspendAction
	if a.CommonComponentSpec != nil && a.CommonComponentSpec.SuspendAction != nil {
		action = a.CommonComponentSpec.SuspendAction
	}
	return action
}

func getComponentLabelValue(c MemberType) string {
	switch c {
	case PDMemberType:
		return label.PDLabelVal
	case TiDBMemberType:
		return label.TiDBLabelVal
	case TiKVMemberType:
		return label.TiKVLabelVal
	case TiFlashMemberType:
		return label.TiFlashLabelVal
	case TiCDCMemberType:
		return label.TiCDCLabelVal
	case PumpMemberType:
		return label.PumpLabelVal
	case DiscoveryMemberType:
		return label.DiscoveryLabelVal
	case DMDiscoveryMemberType:
		return label.DiscoveryLabelVal
	case DMMasterMemberType:
		return label.DMMasterLabelVal
	case DMWorkerMemberType:
		return label.DMWorkerLabelVal
	case NGMonitoringMemberType:
		return label.NGMonitorLabelVal
	}
	return ""
}

func buildTidbClusterComponentAccessor(c MemberType, tc *TidbCluster, componentSpec *CommonComponentSpec) ComponentAccessor {
	spec := &tc.Spec
	return &componentAccessorImpl{
		name:                      tc.Name,
		kind:                      TiDBClusterKind,
		component:                 c,
		imagePullPolicy:           spec.ImagePullPolicy,
		imagePullSecrets:          spec.ImagePullSecrets,
		hostNetwork:               spec.HostNetwork,
		affinity:                  spec.Affinity,
		priorityClassName:         spec.PriorityClassName,
		schedulerName:             spec.SchedulerName,
		clusterNodeSelector:       spec.NodeSelector,
		clusterLabels:             spec.Labels,
		clusterAnnotations:        spec.Annotations,
		tolerations:               spec.Tolerations,
		dnsConfig:                 spec.DNSConfig,
		dnsPolicy:                 spec.DNSPolicy,
		configUpdateStrategy:      spec.ConfigUpdateStrategy,
		statefulSetUpdateStrategy: spec.StatefulSetUpdateStrategy,
		podManagementPolicy:       spec.PodManagementPolicy,
		podSecurityContext:        spec.PodSecurityContext,
		topologySpreadConstraints: spec.TopologySpreadConstraints,
		suspendAction:             spec.SuspendAction,

		CommonComponentSpec: componentSpec,
	}
}

func buildDMClusterComponentAccessor(c MemberType, dc *DMCluster, componentSpec *CommonComponentSpec) ComponentAccessor {
	spec := &dc.Spec
	return &componentAccessorImpl{
		name:                      dc.Name,
		kind:                      DMClusterKind,
		component:                 c,
		imagePullPolicy:           spec.ImagePullPolicy,
		imagePullSecrets:          spec.ImagePullSecrets,
		hostNetwork:               spec.HostNetwork,
		affinity:                  spec.Affinity,
		priorityClassName:         spec.PriorityClassName,
		schedulerName:             spec.SchedulerName,
		clusterNodeSelector:       spec.NodeSelector,
		clusterLabels:             spec.Labels,
		clusterAnnotations:        spec.Annotations,
		tolerations:               spec.Tolerations,
		dnsConfig:                 spec.DNSConfig,
		dnsPolicy:                 spec.DNSPolicy,
		configUpdateStrategy:      spec.ConfigUpdateStrategy,
		statefulSetUpdateStrategy: spec.StatefulSetUpdateStrategy,
		podManagementPolicy:       spec.PodManagementPolicy,
		podSecurityContext:        spec.PodSecurityContext,
		topologySpreadConstraints: spec.TopologySpreadConstraints,
		suspendAction:             spec.SuspendAction,

		CommonComponentSpec: componentSpec,
	}
}

func buildTiDBNGMonitoringComponentAccessor(c MemberType, tngm *TidbNGMonitoring, componentSpec *CommonComponentSpec) ComponentAccessor {
	commonSpec := tngm.Spec.CommonComponentSpec
	impl := &componentAccessorImpl{
		name:                      tngm.GetName(),
		kind:                      TiDBNGMonitoringKind,
		component:                 c,
		imagePullSecrets:          commonSpec.ImagePullSecrets,
		hostNetwork:               commonSpec.HostNetwork,
		affinity:                  commonSpec.Affinity,
		priorityClassName:         commonSpec.PriorityClassName,
		clusterNodeSelector:       commonSpec.NodeSelector,
		clusterLabels:             commonSpec.Labels,
		clusterAnnotations:        commonSpec.Annotations,
		tolerations:               commonSpec.Tolerations,
		configUpdateStrategy:      ConfigUpdateStrategyRollingUpdate,
		podSecurityContext:        commonSpec.PodSecurityContext,
		topologySpreadConstraints: commonSpec.TopologySpreadConstraints,
		dnsConfig:                 commonSpec.DNSConfig,
		dnsPolicy:                 commonSpec.DNSPolicy,

		CommonComponentSpec: componentSpec,
	}
	if commonSpec.ImagePullPolicy != nil {
		impl.imagePullPolicy = *commonSpec.ImagePullPolicy
	}
	if commonSpec.SchedulerName != nil {
		impl.schedulerName = *commonSpec.SchedulerName
	}
	return impl
}

func buildTiDBDashboardComponentAccessor(c MemberType, td *TidbDashboard, componentSpec *CommonComponentSpec) ComponentAccessor {
	commonSpec := td.Spec.CommonComponentSpec
	impl := &componentAccessorImpl{
		name:                      td.GetName(),
		kind:                      TiDBDashboardKind,
		component:                 c,
		imagePullSecrets:          commonSpec.ImagePullSecrets,
		hostNetwork:               commonSpec.HostNetwork,
		affinity:                  commonSpec.Affinity,
		priorityClassName:         commonSpec.PriorityClassName,
		clusterNodeSelector:       commonSpec.NodeSelector,
		clusterLabels:             commonSpec.Labels,
		clusterAnnotations:        commonSpec.Annotations,
		tolerations:               commonSpec.Tolerations,
		configUpdateStrategy:      ConfigUpdateStrategyRollingUpdate,
		podSecurityContext:        commonSpec.PodSecurityContext,
		topologySpreadConstraints: commonSpec.TopologySpreadConstraints,
		dnsConfig:                 commonSpec.DNSConfig,
		dnsPolicy:                 commonSpec.DNSPolicy,

		CommonComponentSpec: componentSpec,
	}
	if commonSpec.ImagePullPolicy != nil {
		impl.imagePullPolicy = *commonSpec.ImagePullPolicy
	}
	if commonSpec.SchedulerName != nil {
		impl.schedulerName = *commonSpec.SchedulerName
	}
	return impl
}

// BaseDiscoverySpec returns the base spec of discovery component
func (tc *TidbCluster) BaseDiscoverySpec() ComponentAccessor {
	return buildTidbClusterComponentAccessor(DiscoveryMemberType, tc, tc.Spec.Discovery.CommonComponentSpec)
}

// BaseTiDBSpec returns the base spec of TiDB servers
func (tc *TidbCluster) BaseTiDBSpec() ComponentAccessor {
	var spec *CommonComponentSpec
	if tc.Spec.TiDB != nil {
		spec = &tc.Spec.TiDB.CommonComponentSpec
	}

	return buildTidbClusterComponentAccessor(TiDBMemberType, tc, spec)
}

// BaseTiKVSpec returns the base spec of TiKV servers
func (tc *TidbCluster) BaseTiKVSpec() ComponentAccessor {
	var spec *CommonComponentSpec
	if tc.Spec.TiKV != nil {
		spec = &tc.Spec.TiKV.CommonComponentSpec
	}

	return buildTidbClusterComponentAccessor(TiKVMemberType, tc, spec)
}

// BaseTiFlashSpec returns the base spec of TiFlash servers
func (tc *TidbCluster) BaseTiFlashSpec() ComponentAccessor {
	var spec *CommonComponentSpec
	if tc.Spec.TiFlash != nil {
		spec = &tc.Spec.TiFlash.CommonComponentSpec
	}

	return buildTidbClusterComponentAccessor(TiFlashMemberType, tc, spec)
}

// BaseTiProxySpec returns the base spec of TiProxy servers
func (tc *TidbCluster) BaseTiProxySpec() ComponentAccessor {
	var spec *CommonComponentSpec
	if tc.Spec.TiProxy != nil {
		spec = &tc.Spec.TiProxy.CommonComponentSpec
	}

	return buildTidbClusterComponentAccessor(TiProxyMemberType, tc, spec)
}

// BaseTiCDCSpec returns the base spec of TiCDC servers
func (tc *TidbCluster) BaseTiCDCSpec() ComponentAccessor {
	var spec *CommonComponentSpec
	if tc.Spec.TiCDC != nil {
		spec = &tc.Spec.TiCDC.CommonComponentSpec
	}

	return buildTidbClusterComponentAccessor(TiCDCMemberType, tc, spec)
}

// BasePDSpec returns the base spec of PD servers
func (tc *TidbCluster) BasePDSpec() ComponentAccessor {
	var spec *CommonComponentSpec
	if tc.Spec.PD != nil {
		spec = &tc.Spec.PD.CommonComponentSpec
	}

	return buildTidbClusterComponentAccessor(PDMemberType, tc, spec)
}

// BasePumpSpec returns the base spec of Pump:
func (tc *TidbCluster) BasePumpSpec() ComponentAccessor {
	var spec *CommonComponentSpec
	if tc.Spec.Pump != nil {
		spec = &tc.Spec.Pump.CommonComponentSpec
	}

	return buildTidbClusterComponentAccessor(PumpMemberType, tc, spec)
}

func (dc *DMCluster) BaseDiscoverySpec() ComponentAccessor {
	return buildDMClusterComponentAccessor(DMDiscoveryMemberType, dc, dc.Spec.Discovery.CommonComponentSpec)
}

func (dc *DMCluster) BaseMasterSpec() ComponentAccessor {
	return buildDMClusterComponentAccessor(DMMasterMemberType, dc, &dc.Spec.Master.CommonComponentSpec)
}

func (dc *DMCluster) BaseWorkerSpec() ComponentAccessor {
	var spec *CommonComponentSpec
	if dc.Spec.Worker != nil {
		spec = &dc.Spec.Worker.CommonComponentSpec
	}

	return buildDMClusterComponentAccessor(DMWorkerMemberType, dc, spec)
}

func (tngm *TidbNGMonitoring) BaseNGMonitoringSpec() ComponentAccessor {
	return buildTiDBNGMonitoringComponentAccessor(NGMonitoringMemberType, tngm, &tngm.Spec.NGMonitoring.CommonComponentSpec)
}

func (td *TidbDashboard) BaseTidbDashboardSpec() ComponentAccessor {
	return buildTiDBDashboardComponentAccessor(TiDBDashboardMemberType, td, &td.Spec.CommonComponentSpec)
}
