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

package monitor

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

func getMonitorObjectName(monitor *v1alpha1.TidbMonitor) string {
	return fmt.Sprintf("%s-monitor", monitor.Name)
}

func getMonitorTargetRegex(monitor *v1alpha1.TidbMonitor) string {
	clusters := monitor.Spec.Clusters
	regex := clusters[0].Name
	for _, cluster := range clusters[1:] {
		regex = fmt.Sprintf("%s|%s", regex, cluster.Name)
	}
	return regex
}
