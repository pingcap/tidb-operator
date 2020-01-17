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

func (tm *TidbMonitor) PrometheusPortName() *string {
	return tm.Spec.Prometheus.Service.PortName
}

func (tm *TidbMonitor) GrafanaPortName() *string {
	if tm.Spec.Grafana != nil {
		return tm.Spec.Grafana.Service.PortName
	}
	return nil
}

func (tm *TidbMonitor) ReloaderPortName() *string {
	return tm.Spec.Reloader.Service.PortName
}
