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

package tidbngmonitoring

import "fmt"

// NGMonitoringName return ng monitoring name
func NGMonitoringName(monitorName string) string {
	return fmt.Sprintf("%s-ng-monitoring", monitorName)
}

// NGMonitoringHeadlessServiceName return headless service name
func NGMonitoringHeadlessServiceName(tngm string) string {
	return fmt.Sprintf("%s-ng-monitoring", tngm)
}
