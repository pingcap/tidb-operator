// Copyright 2022 PingCAP, Inc.
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

package tidbdashboard

import (
	"fmt"
)

const (
	clusterTLSMountPath  = "/var/lib/cluster-client-tls"
	clusterTLSVolumeName = "cluster-client-tls"

	mysqlTLSMountPath  = "/var/lib/mysql-client-tls"
	mysqlTLSVolumeName = "mysql-client-tls"

	dataPVCMountPath  = "/var/lib/tidb-dashboard-data"
	dataPVCVolumeName = "tidb-dashboard-data"

	port = 12333
)

// StatefulSetName return dashboard name.
func StatefulSetName(td string) string {
	return fmt.Sprintf("%s-tidb-dashboard", td)
}

// ServiceName return a service name for dashboard.
func ServiceName(td string) string {
	return fmt.Sprintf("%s-tidb-dashboard-exposed", td)
}

// TCClusterClientTLSSecretName return name of secret which contains cluster client certs borrowed from tc.
func TCClusterClientTLSSecretName(td string) string {
	return fmt.Sprintf("%s-tc-cluster-client-tls", td)
}

// TCMySQLClientTLSSecretName return name of secret which contains mysql client certs borrowed from tc.
func TCMySQLClientTLSSecretName(td string) string {
	return fmt.Sprintf("%s-tc-mysql-client-tls", td)
}
