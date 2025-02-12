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

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

func ConfigMapName(tidbName string) string {
	return tidbName
}

func PersistentVolumeClaimName(podName, volName string) string {
	// ref: https://github.com/pingcap/tidb-operator/blob/v1.6.0/pkg/apis/pingcap/v1alpha1/helpers.go#L92
	// NOTE: for v1, should use component as volName of data, e.g. tidb
	return volName + "-" + podName
}

// TiDBServiceURL returns the service URL of a tidb member.
func TiDBServiceURL(tidb *v1alpha1.TiDB, scheme string) string {
	return fmt.Sprintf("%s://%s.%s.%s.svc:%d",
		scheme,
		coreutil.PodName[scope.TiDB](tidb),
		tidb.Spec.Subdomain,
		tidb.Namespace,
		coreutil.TiDBStatusPort(tidb),
	)
}

// Real spec.volumes[*].name of pod
// TODO(liubo02): extract to namer pkg
func VolumeName(volName string) string {
	return meta.VolNamePrefix + volName
}

func VolumeMount(name string, mount *v1alpha1.VolumeMount) *corev1.VolumeMount {
	vm := &corev1.VolumeMount{
		Name:      name,
		MountPath: mount.MountPath,
		SubPath:   mount.SubPath,
	}
	if mount.Type == v1alpha1.VolumeMountTypeTiDBSlowLog {
		if vm.MountPath == "" {
			vm.MountPath = v1alpha1.VolumeMountTiDBSlowLogDefaultPath
		}
	}

	return vm
}
