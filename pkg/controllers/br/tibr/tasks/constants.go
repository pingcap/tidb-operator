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

import corev1 "k8s.io/api/core/v1"

const (
	ConfigFileName     = "tibr-config.yaml"
	StatefulSetReplica = int32(1)
	APIServerPort      = 19500
	SecretAccessMode   = int32(420)
	ConfigVolumeName   = "config-volume"
	TLSVolumeName      = "tibr-tls"
	TLSMountPath       = "/var/lib/tibr-tls"
	ConfigMountPath    = "/etc/config"
	DataMountPath      = "/data"
)

var (
	TLSCmdArgs = []string{
		"--cacert",
		TLSMountPath + "/ca.crt",
		"--cert",
		TLSMountPath + "/tls.crt",
		"--key",
		TLSMountPath + "/tls.key",
	}
	configVolumeMount = corev1.VolumeMount{
		Name: ConfigVolumeName, MountPath: ConfigMountPath,
	}
	tlsVolumeMount = corev1.VolumeMount{
		Name: TLSVolumeName, MountPath: TLSMountPath,
	}
)
