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

package k8s

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	hashutil "github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/util/hash"
)

// CalculateHashAndSetLabels calculate the hash of pod spec and set it to the pod labels.
func CalculateHashAndSetLabels(pod *corev1.Pod) {
	spec := pod.Spec.DeepCopy()
	for i := range spec.InitContainers {
		c := &spec.InitContainers[i]
		// ignores init containers image change to support hot reload image for sidecar
		c.Image = ""
	}

	// This prevents the hash from being changed when new fields are added to the `PodSpec` due to K8s version upgrades.
	data, _ := json.Marshal(spec)
	hasher := fnv.New32a()
	hashutil.DeepHashObject(hasher, data)

	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[v1alpha1.LabelKeyPodSpecHash] = rand.SafeEncodeString(fmt.Sprint(hasher.Sum32()))
}

type CompareResult int

const (
	CompareResultEqual CompareResult = iota
	CompareResultRecreate
	CompareResultUpdate
)

func (r CompareResult) String() string {
	switch r {
	case CompareResultEqual:
		return "Equal"
	case CompareResultRecreate:
		return "Recreate"
	case CompareResultUpdate:
		return "Update"
	default:
		return "Unknown"
	}
}

// ComparePods compares two pods and returns the result of comparison.
// TODO: add check for changes that can be updated without recreating the pod.
func ComparePods(current, expected *corev1.Pod) CompareResult {
	if current.GetLabels()[v1alpha1.LabelKeyPodSpecHash] == expected.GetLabels()[v1alpha1.LabelKeyPodSpecHash] {
		// The revision hash will always be different when there is a change, so ignore it.
		p1, p2 := current.DeepCopy(), expected.DeepCopy()
		// We also should update labels of pods even if revisions are not equal
		// delete(p1.Labels, v1alpha1.LabelKeyInstanceRevisionHash)
		// delete(p2.Labels, v1alpha1.LabelKeyInstanceRevisionHash)
		if !maputil.AreEqual(p1.Labels, p2.Labels) || !maputil.AreEqual(p1.Annotations, p2.Annotations) {
			// Labels or annotations are different, need to update the pod.
			return CompareResultUpdate
		}
		// No difference found, no need to update the pod.
		return CompareResultEqual
	}
	// Pod spec hash is different, need to recreate the pod.
	return CompareResultRecreate
}

func GetResourceRequirements(req v1alpha1.ResourceRequirements) corev1.ResourceRequirements {
	if req.CPU == nil && req.Memory == nil {
		return corev1.ResourceRequirements{}
	}
	ret := corev1.ResourceRequirements{
		Limits:   map[corev1.ResourceName]resource.Quantity{},
		Requests: map[corev1.ResourceName]resource.Quantity{},
	}
	if req.CPU != nil {
		ret.Requests[corev1.ResourceCPU] = *req.CPU
		ret.Limits[corev1.ResourceCPU] = *req.CPU
	}
	if req.Memory != nil {
		ret.Requests[corev1.ResourceMemory] = *req.Memory
		ret.Limits[corev1.ResourceMemory] = *req.Memory
	}
	return ret
}
