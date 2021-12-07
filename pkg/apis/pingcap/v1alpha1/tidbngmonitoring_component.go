// Copyright 2021 PingCAP, Inc.
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

func (tngm *TidbNGMonitoring) BaseNGMonitoringSpec() ComponentAccessor {
	return buildTiDBNGMonitoringComponentAccessor(ComponentNGMonitoring, tngm, &tngm.Spec.NGMonitoring.ComponentSpec)
}

func buildTiDBNGMonitoringComponentAccessor(c Component, tngm *TidbNGMonitoring, componentSpec *ComponentSpec) ComponentAccessor {
	commonSpec := tngm.Spec.ComponentSpec
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

		ComponentSpec: componentSpec,
	}
	if commonSpec.ImagePullPolicy != nil {
		impl.imagePullPolicy = *commonSpec.ImagePullPolicy
	}
	if commonSpec.SchedulerName != nil {
		impl.schedulerName = *commonSpec.SchedulerName
	}
	return impl
}
