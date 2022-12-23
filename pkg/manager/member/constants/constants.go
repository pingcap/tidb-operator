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

package constants

const (
	// TiProxyVolumeMountPath is the path for tiproxy data volume
	TiProxyVolumeMountPath = "/var/lib/tiproxy"

	// TiKVDataVolumeMountPath is the mount path for tikv data volume
	TiKVDataVolumeMountPath = "/var/lib/tikv"

	// PDDataVolumeMountPath is the mount path for pd data volume
	PDDataVolumeMountPath = "/var/lib/pd"

	// TiCDCCertPath is the path for ticdc cert in container
	TiCDCCertPath = "/var/lib/ticdc-tls"
)
