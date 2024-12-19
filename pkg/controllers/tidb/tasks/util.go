// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tasks

import (
	"fmt"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

func ConfigMapName(tidbName string) string {
	return tidbName
}

func PersistentVolumeClaimName(podName, volName string) string {
	// ref: https://github.com/pingcap/tidb-operator/blob/v1.6.0/pkg/apis/pingcap/v1alpha1/helpers.go#L92
	if volName == "" {
		return "tidb-" + podName
	}
	return "tidb-" + podName + "-" + volName
}

// TiDBServiceURL returns the service URL of a tidb member.
func TiDBServiceURL(tidb *v1alpha1.TiDB, scheme string) string {
	return fmt.Sprintf("%s://%s.%s.%s.svc:%d", scheme, tidb.PodName(), tidb.Spec.Subdomain, tidb.Namespace, tidb.GetStatusPort())
}