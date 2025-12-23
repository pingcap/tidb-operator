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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

func LongestHealthPeer(pd *v1alpha1.PD, peers []*v1alpha1.PD) string {
	var p string
	lastTime := time.Now()
	for _, peer := range peers {
		if peer.Name == pd.Name {
			continue
		}
		cond := meta.FindStatusCondition(peer.Status.Conditions, v1alpha1.CondReady)
		if cond == nil || cond.Status != metav1.ConditionTrue {
			continue
		}
		if cond.LastTransitionTime.Time.Before(lastTime) {
			lastTime = cond.LastTransitionTime.Time
			p = peer.Name
		}
	}

	return p
}

// VolumeName returns the real spec.volumes[*].name of pod
// TODO(liubo02): extract to namer pkg
func VolumeName(volName string) string {
	return metav1alpha1.VolNamePrefix + volName
}

func VolumeMount(name string, mount *v1alpha1.VolumeMount) *corev1.VolumeMount {
	vm := &corev1.VolumeMount{
		Name:      name,
		MountPath: mount.MountPath,
		SubPath:   mount.SubPath,
	}

	if mount.Type == v1alpha1.VolumeMountTypePDData {
		if vm.MountPath == "" {
			vm.MountPath = v1alpha1.VolumeMountPDDataDefaultPath
		}
	}

	return vm
}
