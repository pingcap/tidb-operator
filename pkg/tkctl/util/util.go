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

package util

import v1 "k8s.io/api/core/v1"

const (
	DockerSocket = "/var/run/docker.sock"
)

// MakeDockerSocketMount create the volume and corresponding mount for docker socket
func MakeDockerSocketMount(hostDockerSocket string, readOnly bool) (volume v1.Volume, mount v1.VolumeMount) {
	mount = v1.VolumeMount{
		Name:      "docker",
		ReadOnly:  readOnly,
		MountPath: DockerSocket,
	}
	volume = v1.Volume{
		Name: "docker",
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: hostDockerSocket,
			},
		},
	}
	return
}

// TODO: fix unsafe name infer after resources being managed by CRD
// GetTidbServiceName infers tidb service name from tidb cluster name
func GetTidbServiceName(tc string) string {
	return tc + "-tidb"
}
