// Copyright 2018 PingCAP, Inc.
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

package member

import (
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func timezoneMountVolume() (corev1.VolumeMount, corev1.Volume) {
	return corev1.VolumeMount{Name: "timezone", MountPath: "/etc/localtime", ReadOnly: true},
		corev1.Volume{
			Name:         "timezone",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/etc/localtime"}},
		}
}

func annotationsMountVolume() (corev1.VolumeMount, corev1.Volume) {
	m := corev1.VolumeMount{Name: "annotations", ReadOnly: true, MountPath: "/etc/podinfo"}
	v := corev1.Volume{
		Name: "annotations",
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path:     "annotations",
						FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations"},
					},
				},
			},
		},
	}
	return m, v
}

// statefulSetInNormal confirm whether the statefulSet is normal state
func statefulSetInNormal(set *apps.StatefulSet) bool {
	return set.Status.CurrentRevision == set.Status.UpdateRevision && set.Generation <= *set.Status.ObservedGeneration
}
