/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// this file is copied from k8s.io/kubernetes/test/e2e/framework/statefulset/rest.go @v1.23.17

package statefulset

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	framework "github.com/pingcap/tidb-operator/tests/third_party/k8s"
)

// GetPodList gets the current Pods in ss.
func GetPodList(c clientset.Interface, ss *appsv1.StatefulSet) *v1.PodList {
	selector, err := metav1.LabelSelectorAsSelector(ss.Spec.Selector)
	framework.ExpectNoError(err)
	podList, err := c.CoreV1().Pods(ss.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	framework.ExpectNoError(err)
	return podList
}
