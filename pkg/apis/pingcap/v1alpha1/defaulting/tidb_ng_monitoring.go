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

package defaulting

import "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

const (
	defaultNGMonitoringImage = "pingcap/ng-monitoring"
)

func SetTidbNGMonitoringDefault(tngm *v1alpha1.TidbNGMonitoring) {
	setTidbNGMonitoringSpecDefault(tngm)
}

func setTidbNGMonitoringSpecDefault(tngm *v1alpha1.TidbNGMonitoring) {
	for id := range tngm.Spec.Clusters {
		if tngm.Spec.Clusters[id].Namespace == "" {
			tngm.Spec.Clusters[id].Namespace = tngm.Namespace
		}
	}
}
