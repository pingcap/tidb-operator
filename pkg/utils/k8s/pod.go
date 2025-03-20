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
	"bytes"
	"fmt"
	"hash/fnv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	serializerjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	hashutil "github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/util/hash"
)

var podEncoder = scheme.Codecs.EncoderForVersion(
	serializerjson.NewSerializerWithOptions(
		serializerjson.DefaultMetaFactory,
		scheme.Scheme,
		scheme.Scheme,
		serializerjson.SerializerOptions{
			Yaml:   false,
			Pretty: false,
			Strict: true,
		}),
	corev1.SchemeGroupVersion,
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
	buf := bytes.Buffer{}
	if err := podEncoder.Encode(&corev1.Pod{Spec: *spec}, &buf); err != nil {
		panic(fmt.Errorf("failed to encode pod spec, %w", err))
	}
	hasher := fnv.New32a()
	hashutil.DeepHashObject(hasher, buf.Bytes())

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
	if current.Labels[v1alpha1.LabelKeyPodSpecHash] == expected.Labels[v1alpha1.LabelKeyPodSpecHash] {
		if current.Annotations[metav1alpha1.RestartAnnotationKey] != expected.Annotations[metav1alpha1.RestartAnnotationKey] {
			// Restart annotation is different, need to recreate the pod.
			return CompareResultRecreate
		}
		if !maputil.AreEqual(current.Labels, expected.Labels) || !maputil.AreEqual(current.Annotations, expected.Annotations) {
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

// AnnoProm returns the prometheus annotations for a pod.
func AnnoProm(port int32, path string) map[string]string {
	return map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   fmt.Sprintf("%d", port),
		"prometheus.io/path":   path,
	}
}

// AnnoAdditionalProm returns the additional prometheus annotations for a pod.
// Some pods may have multiple prometheus endpoints.
// We assume the same path is used for all endpoints.
func AnnoAdditionalProm(name string, port int32) map[string]string {
	return map[string]string{
		fmt.Sprintf("%s.prometheus.io/port", name): fmt.Sprintf("%d", port),
	}
}

// LabelsForTidbCluster returns some labels with "app.kubernetes.io" prefix.
// NOTE: these labels are deprecated and they are used for TiDB Operator v1 compatibility.
// If you are developing a new feature, please use labels with "pingcap.com" prefix instead.
func LabelsK8sApp(cluster, component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/component":  component,
		"app.kubernetes.io/instance":   cluster,
		"app.kubernetes.io/managed-by": "tidb-operator",
		"app.kubernetes.io/name":       "tidb-cluster",
	}
}
