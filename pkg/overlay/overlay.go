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

package overlay

import (
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

func OverlayPod(pod *corev1.Pod, overlay *v1alpha1.PodOverlay) {
	if overlay == nil {
		return
	}
	src := overlay.DeepCopy()
	// Field spec.nodeSelector is an atomic map
	// But we hope to overlay it as a granular map
	// Because we may inject topology selector into node selector
	//
	// TODO: validate that conflict keys cannot be added into overlay
	// But now, just overwrite the conflict keys.
	for k, v := range pod.Spec.NodeSelector {
		src.Spec.NodeSelector[k] = v
	}

	overlayObjectMeta(&pod.ObjectMeta, convertObjectMeta(&src.ObjectMeta))
	overlayPodSpec(&pod.Spec, src.Spec)
}

func convertObjectMeta(meta *v1alpha1.ObjectMeta) *metav1.ObjectMeta {
	return &metav1.ObjectMeta{
		Annotations: meta.Annotations,
		Labels:      meta.Labels,
	}
}

// Predefined overlay quantity function
func overlayQuantity(dst, src *resource.Quantity) {
	*dst = *src
}

// Predefined overlay objectmeta function
func overlayObjectMeta(dst, src *metav1.ObjectMeta) {
	if src.Labels != nil {
		if dst.Labels == nil {
			dst.Labels = map[string]string{}
		}
		for k, v := range src.Labels {
			dst.Labels[k] = v
		}
	}

	if src.Annotations != nil {
		if dst.Annotations == nil {
			dst.Annotations = map[string]string{}
		}
		for k, v := range src.Annotations {
			dst.Annotations[k] = v
		}
	}
}
