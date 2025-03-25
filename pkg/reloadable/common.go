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

package reloadable

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func convertOverlay(o *v1alpha1.Overlay) *v1alpha1.Overlay {
	if o == nil {
		o = &v1alpha1.Overlay{}
	}
	if o.Pod == nil {
		o.Pod = &v1alpha1.PodOverlay{}
	}
	if o.Pod.Spec == nil {
		o.Pod.Spec = &corev1.PodSpec{}
	}

	o.Pod.Labels = nil
	o.Pod.Annotations = nil

	// ignore all pvc overlay
	o.PersistentVolumeClaims = nil

	return o
}

func convertVolumes(vols []v1alpha1.Volume) []v1alpha1.Volume {
	for i := range vols {
		vol := &vols[i]
		vol.Storage = resource.Quantity{}
		vol.StorageClassName = nil
		vol.VolumeAttributesClassName = nil
	}

	return vols
}

func convertLabels(ls map[string]string) map[string]string {
	// Ignore additional labels applied to the instance
	// See apiutil.InstanceLabels
	delete(ls, v1alpha1.LabelKeyManagedBy)
	delete(ls, v1alpha1.LabelKeyComponent)
	delete(ls, v1alpha1.LabelKeyCluster)
	delete(ls, v1alpha1.LabelKeyGroup)
	delete(ls, v1alpha1.LabelKeyInstanceRevisionHash)

	return ls
}

func convertAnnotations(ls map[string]string) map[string]string {
	// ignore boot annotation of pd
	delete(ls, v1alpha1.AnnoKeyInitialClusterNum)

	return ls
}
