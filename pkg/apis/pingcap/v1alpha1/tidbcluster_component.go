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
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultHostNetwork = false
)

// ComponentAccessor is the interface to access component details, which respects the cluster-level properties
// and component-level overrides
type ComponentAccessor interface {
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
	AdditionalContainers() []corev1.Container
	InitContainers() []corev1.Container
	AdditionalVolumes() []corev1.Volume
	AdditionalVolumeMounts() []corev1.VolumeMount
	TerminationGracePeriodSeconds() *int64
	StatefulSetUpdateStrategy() apps.StatefulSetUpdateStrategyType
	TopologySpreadConstraints() []corev1.TopologySpreadConstraint
}

// Component defines component identity of all components
type Component int

const (
	ComponentPD Component = iota
	ComponentTiDB
	ComponentTiKV
	ComponentTiFlash
	ComponentTiCDC
	ComponentPump
	ComponentDiscovery
	ComponentDMDiscovery
	ComponentDMMaster
	ComponentDMWorker
)

type componentAccessorImpl struct {
	component Component
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
	configUpdateStrategy      ConfigUpdateStrategy
	statefulSetUpdateStrategy apps.StatefulSetUpdateStrategyType
	podSecurityContext        *corev1.PodSecurityContext
	topologySpreadConstraints []TopologySpreadConstraint

	// ComponentSpec is the Component Spec
	ComponentSpec *ComponentSpec
}

func (a *componentAccessorImpl) StatefulSetUpdateStrategy() apps.StatefulSetUpdateStrategyType {
	if a.ComponentSpec == nil || len(a.ComponentSpec.StatefulSetUpdateStrategy) == 0 {
		if len(a.statefulSetUpdateStrategy) == 0 {
			return apps.RollingUpdateStatefulSetStrategyType
		}
		return a.statefulSetUpdateStrategy
	}
	return a.ComponentSpec.StatefulSetUpdateStrategy
}

func (a *componentAccessorImpl) PodSecurityContext() *corev1.PodSecurityContext {
	if a.ComponentSpec == nil || a.ComponentSpec.PodSecurityContext == nil {
		return a.podSecurityContext
	}
	return a.ComponentSpec.PodSecurityContext
}

func (a *componentAccessorImpl) ImagePullPolicy() corev1.PullPolicy {
	if a.ComponentSpec == nil || a.ComponentSpec.ImagePullPolicy == nil {
		return a.imagePullPolicy
	}
	return *a.ComponentSpec.ImagePullPolicy
}

func (a *componentAccessorImpl) ImagePullSecrets() []corev1.LocalObjectReference {
	if a.ComponentSpec == nil || len(a.ComponentSpec.ImagePullSecrets) == 0 {
		return a.imagePullSecrets
	}
	return a.ComponentSpec.ImagePullSecrets
}

func (a *componentAccessorImpl) HostNetwork() bool {
	if a.ComponentSpec == nil || a.ComponentSpec.HostNetwork == nil {
		if a.hostNetwork == nil {
			return defaultHostNetwork
		}
		return *a.hostNetwork
	}
	return *a.ComponentSpec.HostNetwork
}

func (a *componentAccessorImpl) Affinity() *corev1.Affinity {
	if a.ComponentSpec == nil || a.ComponentSpec.Affinity == nil {
		return a.affinity
	}
	return a.ComponentSpec.Affinity
}

func (a *componentAccessorImpl) PriorityClassName() *string {
	if a.ComponentSpec == nil || a.ComponentSpec.PriorityClassName == nil {
		return a.priorityClassName
	}
	return a.ComponentSpec.PriorityClassName
}

func (a *componentAccessorImpl) SchedulerName() string {
	if a.ComponentSpec == nil || a.ComponentSpec.SchedulerName == nil {
		return a.schedulerName
	}
	return *a.ComponentSpec.SchedulerName
}

func (a *componentAccessorImpl) NodeSelector() map[string]string {
	sel := map[string]string{}
	for k, v := range a.clusterNodeSelector {
		sel[k] = v
	}
	if a.ComponentSpec != nil {
		for k, v := range a.ComponentSpec.NodeSelector {
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
	if a.ComponentSpec != nil {
		for k, v := range a.ComponentSpec.Labels {
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
	if a.ComponentSpec != nil {
		for k, v := range a.ComponentSpec.Annotations {
			anno[k] = v
		}
	}
	return anno
}

func (a *componentAccessorImpl) Tolerations() []corev1.Toleration {
	if a.ComponentSpec == nil || len(a.ComponentSpec.Tolerations) == 0 {
		return a.tolerations
	}
	return a.ComponentSpec.Tolerations
}

func (a *componentAccessorImpl) DnsPolicy() corev1.DNSPolicy {
	dnsPolicy := corev1.DNSClusterFirst // same as kubernetes default
	if a.HostNetwork() {
		dnsPolicy = corev1.DNSClusterFirstWithHostNet
	}
	return dnsPolicy
}

func (a *componentAccessorImpl) ConfigUpdateStrategy() ConfigUpdateStrategy {
	// defaulting logic will set a default value for configUpdateStrategy field, but if the
	// object is created in early version without this field being set, we should set a safe default
	if a.ComponentSpec == nil || a.ComponentSpec.ConfigUpdateStrategy == nil {
		if a.configUpdateStrategy != "" {
			return a.configUpdateStrategy
		}
		return ConfigUpdateStrategyInPlace
	}
	if *a.ComponentSpec.ConfigUpdateStrategy == "" {
		return ConfigUpdateStrategyInPlace
	}
	return *a.ComponentSpec.ConfigUpdateStrategy
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
	if a.ComponentSpec == nil {
		return nil
	}
	return a.ComponentSpec.Env
}

func (a *componentAccessorImpl) InitContainers() []corev1.Container {
	if a.ComponentSpec == nil {
		return nil
	}
	return a.ComponentSpec.InitContainers
}

func (a *componentAccessorImpl) AdditionalContainers() []corev1.Container {
	if a.ComponentSpec == nil {
		return nil
	}
	return a.ComponentSpec.AdditionalContainers
}

func (a *componentAccessorImpl) AdditionalVolumes() []corev1.Volume {
	if a.ComponentSpec == nil {
		return nil
	}
	return a.ComponentSpec.AdditionalVolumes
}

func (a *componentAccessorImpl) AdditionalVolumeMounts() []corev1.VolumeMount {
	if a.ComponentSpec == nil {
		return nil
	}
	return a.ComponentSpec.AdditionalVolumeMounts
}

func (a *componentAccessorImpl) TerminationGracePeriodSeconds() *int64 {
	if a.ComponentSpec == nil {
		return nil
	}
	return a.ComponentSpec.TerminationGracePeriodSeconds
}

func (a *componentAccessorImpl) TopologySpreadConstraints() []corev1.TopologySpreadConstraint {
	tscs := a.topologySpreadConstraints
	if a.ComponentSpec != nil && len(a.ComponentSpec.TopologySpreadConstraints) > 0 {
		tscs = a.ComponentSpec.TopologySpreadConstraints
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

func getComponentLabelValue(c Component) string {
	switch c {
	case ComponentPD:
		return label.PDLabelVal
	case ComponentTiDB:
		return label.TiDBLabelVal
	case ComponentTiKV:
		return label.TiKVLabelVal
	case ComponentTiFlash:
		return label.TiFlashLabelVal
	case ComponentTiCDC:
		return label.TiCDCLabelVal
	case ComponentPump:
		return label.PumpLabelVal
	case ComponentDiscovery:
		return label.DiscoveryLabelVal
	case ComponentDMDiscovery:
		return label.DiscoveryLabelVal
	case ComponentDMMaster:
		return label.DMMasterLabelVal
	case ComponentDMWorker:
		return label.DMWorkerLabelVal
	}
	return ""
}

func buildTidbClusterComponentAccessor(c Component, tc *TidbCluster, componentSpec *ComponentSpec) ComponentAccessor {
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
		configUpdateStrategy:      spec.ConfigUpdateStrategy,
		statefulSetUpdateStrategy: spec.StatefulSetUpdateStrategy,
		podSecurityContext:        spec.PodSecurityContext,
		topologySpreadConstraints: spec.TopologySpreadConstraints,

		ComponentSpec: componentSpec,
	}
}

func buildDMClusterComponentAccessor(c Component, dc *DMCluster, componentSpec *ComponentSpec) ComponentAccessor {
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
		configUpdateStrategy:      ConfigUpdateStrategyRollingUpdate,
		podSecurityContext:        spec.PodSecurityContext,
		topologySpreadConstraints: spec.TopologySpreadConstraints,

		ComponentSpec: componentSpec,
	}
}

// BaseDiscoverySpec returns the base spec of discovery component
func (tc *TidbCluster) BaseDiscoverySpec() ComponentAccessor {
	// all configs follow global one
	return buildTidbClusterComponentAccessor(ComponentDiscovery, tc, nil)
}

// BaseTiDBSpec returns the base spec of TiDB servers
func (tc *TidbCluster) BaseTiDBSpec() ComponentAccessor {
	var spec *ComponentSpec
	if tc.Spec.TiDB != nil {
		spec = &tc.Spec.TiDB.ComponentSpec
	}

	return buildTidbClusterComponentAccessor(ComponentTiDB, tc, spec)
}

// BaseTiKVSpec returns the base spec of TiKV servers
func (tc *TidbCluster) BaseTiKVSpec() ComponentAccessor {
	var spec *ComponentSpec
	if tc.Spec.TiKV != nil {
		spec = &tc.Spec.TiKV.ComponentSpec
	}

	return buildTidbClusterComponentAccessor(ComponentTiKV, tc, spec)
}

// BaseTiFlashSpec returns the base spec of TiFlash servers
func (tc *TidbCluster) BaseTiFlashSpec() ComponentAccessor {
	var spec *ComponentSpec
	if tc.Spec.TiFlash != nil {
		spec = &tc.Spec.TiFlash.ComponentSpec
	}

	return buildTidbClusterComponentAccessor(ComponentTiFlash, tc, spec)
}

// BaseTiCDCSpec returns the base spec of TiCDC servers
func (tc *TidbCluster) BaseTiCDCSpec() ComponentAccessor {
	var spec *ComponentSpec
	if tc.Spec.TiCDC != nil {
		spec = &tc.Spec.TiCDC.ComponentSpec
	}

	return buildTidbClusterComponentAccessor(ComponentTiCDC, tc, spec)
}

// BasePDSpec returns the base spec of PD servers
func (tc *TidbCluster) BasePDSpec() ComponentAccessor {
	var spec *ComponentSpec
	if tc.Spec.PD != nil {
		spec = &tc.Spec.PD.ComponentSpec
	}

	return buildTidbClusterComponentAccessor(ComponentPD, tc, spec)
}

// BasePumpSpec returns the base spec of Pump:
func (tc *TidbCluster) BasePumpSpec() ComponentAccessor {
	var spec *ComponentSpec
	if tc.Spec.Pump != nil {
		spec = &tc.Spec.Pump.ComponentSpec
	}

	return buildTidbClusterComponentAccessor(ComponentPump, tc, spec)
}

func (dc *DMCluster) BaseDiscoverySpec() ComponentAccessor {
	return buildDMClusterComponentAccessor(ComponentDMDiscovery, dc, nil)
}

func (dc *DMCluster) BaseMasterSpec() ComponentAccessor {
	return buildDMClusterComponentAccessor(ComponentDMMaster, dc, &dc.Spec.Master.ComponentSpec)
}

func (dc *DMCluster) BaseWorkerSpec() ComponentAccessor {
	var spec *ComponentSpec
	if dc.Spec.Worker != nil {
		spec = &dc.Spec.Worker.ComponentSpec
	}

	return buildDMClusterComponentAccessor(ComponentDMWorker, dc, spec)
}
