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

import corev1 "k8s.io/api/core/v1"

type MonitorComponentAccessor interface {
	PortName() *string
	ServiceType() corev1.ServiceType
	ImagePullPolicy() *corev1.PullPolicy
}

type monitorComponentAccessorImpl struct {
	// Monitor is the TidbMonitor Spec
	MonitorSpec *TidbMonitorSpec

	// Container is the Component Spec
	MonitorComponentSpec *MonitorContainer

	// Service is Component Service Spec
	MonitorServiceSpec *ServiceSpec
}

func (m *monitorComponentAccessorImpl) PortName() *string {
	return m.MonitorServiceSpec.PortName
}

func (m *monitorComponentAccessorImpl) ServiceType() corev1.ServiceType {
	return m.MonitorServiceSpec.Type
}

func (m *monitorComponentAccessorImpl) ImagePullPolicy() *corev1.PullPolicy {
	if m.MonitorComponentSpec.ImagePullPolicy == nil {
		return &m.MonitorSpec.ImagePullPolicy
	}
	return m.MonitorComponentSpec.ImagePullPolicy
}

// BasePrometheusSpec return the base spec of Prometheus service
func (tm *TidbMonitor) BasePrometheusSpec() MonitorComponentAccessor {
	return &monitorComponentAccessorImpl{
		MonitorSpec:          &tm.Spec,
		MonitorComponentSpec: &tm.Spec.Prometheus.MonitorContainer,
		MonitorServiceSpec:   &tm.Spec.Prometheus.Service,
	}
}

func (tm *TidbMonitor) BaseGrafanaSpec() MonitorComponentAccessor {
	if tm.Spec.Grafana != nil {
		return &monitorComponentAccessorImpl{
			MonitorSpec:          &tm.Spec,
			MonitorComponentSpec: &tm.Spec.Grafana.MonitorContainer,
			MonitorServiceSpec:   &tm.Spec.Grafana.Service,
		}
	}
	return nil
}

func (tm *TidbMonitor) BaseReloaderSpec() MonitorComponentAccessor {
	return &monitorComponentAccessorImpl{
		MonitorSpec:          &tm.Spec,
		MonitorComponentSpec: &tm.Spec.Reloader.MonitorContainer,
		MonitorServiceSpec:   &tm.Spec.Reloader.Service,
	}
}
