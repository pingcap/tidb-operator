/*
Copyright 2016 The Kubernetes Authors.

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

// this file is copied (and modified) from k8s.io/kubernetes/test/e2e/framework/pods.go @v1.23.17

package k8s

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/onsi/gomega"

	"github.com/pingcap/tidb-operator/tests/third_party/k8s/log"
	e2epod "github.com/pingcap/tidb-operator/tests/third_party/k8s/pod"
)

const (
	// DefaultPodDeletionTimeout is the default timeout for deleting pod
	DefaultPodDeletionTimeout = 3 * time.Minute
)

// ImagePrePullList is the images used in the current test suite. It should be initialized in test suite and
// the images in the list should be pre-pulled in the test suite.  Currently, this is only used by
// node e2e test.
var ImagePrePullList sets.String

// PodClient is a convenience method for getting a pod client interface in the framework's namespace,
// possibly applying test-suite specific transformations to the pod spec, e.g. for
// node e2e pod scheduling.
func (f *Framework) PodClient() *PodClient {
	return &PodClient{
		f:            f,
		PodInterface: f.ClientSet.CoreV1().Pods(f.Namespace.Name),
	}
}

// PodClientNS is a convenience method for getting a pod client interface in an alternative namespace,
// possibly applying test-suite specific transformations to the pod spec, e.g. for
// node e2e pod scheduling.
func (f *Framework) PodClientNS(namespace string) *PodClient {
	return &PodClient{
		f:            f,
		PodInterface: f.ClientSet.CoreV1().Pods(namespace),
	}
}

// PodClient is a struct for pod client.
type PodClient struct {
	f *Framework
	v1core.PodInterface
}

// Create creates a new pod according to the framework specifications (don't wait for it to start).
func (c *PodClient) Create(pod *v1.Pod) *v1.Pod {
	c.mungeSpec(pod)
	p, err := c.PodInterface.Create(context.TODO(), pod, metav1.CreateOptions{})
	ExpectNoError(err, "Error creating Pod")
	return p
}

// DeleteSync deletes the pod and wait for the pod to disappear for `timeout`. If the pod doesn't
// disappear before the timeout, it will fail the test.
func (c *PodClient) DeleteSync(name string, options metav1.DeleteOptions, timeout time.Duration) {
	namespace := c.f.Namespace.Name
	err := c.Delete(context.TODO(), name, options)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Failf("Failed to delete pod %q: %v", name, err)
	}
	gomega.Expect(e2epod.WaitForPodToDisappear(c.f.ClientSet, namespace, name, labels.Everything(),
		2*time.Second, timeout)).To(gomega.Succeed(), "wait for pod %q to disappear", name)
}

// mungeSpec apply test-suite specific transformations to the pod spec.
func (c *PodClient) mungeSpec(pod *v1.Pod) {
	if !TestContext.NodeE2E {
		return
	}

	gomega.Expect(pod.Spec.NodeName).To(gomega.Or(gomega.BeZero(), gomega.Equal(TestContext.NodeName)), "Test misconfigured")
	pod.Spec.NodeName = TestContext.NodeName
	// Node e2e does not support the default DNSClusterFirst policy. Set
	// the policy to DNSDefault, which is configured per node.
	pod.Spec.DNSPolicy = v1.DNSDefault

	// PrepullImages only works for node e2e now. For cluster e2e, image prepull is not enforced,
	// we should not munge ImagePullPolicy for cluster e2e pods.
	if !TestContext.PrepullImages {
		return
	}
	// If prepull is enabled, munge the container spec to make sure the images are not pulled
	// during the test.
	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]
		if c.ImagePullPolicy == v1.PullAlways {
			// If the image pull policy is PullAlways, the image doesn't need to be in
			// the allow list or pre-pulled, because the image is expected to be pulled
			// in the test anyway.
			continue
		}
		// If the image policy is not PullAlways, the image must be in the pre-pull list and
		// pre-pulled.
		gomega.Expect(ImagePrePullList.Has(c.Image)).To(gomega.BeTrue(), "Image %q is not in the pre-pull list, consider adding it to PrePulledImages in test/e2e/common/util.go or NodePrePullImageList in test/e2e_node/image_list.go", c.Image)
		// Do not pull images during the tests because the images in pre-pull list should have
		// been prepulled.
		c.ImagePullPolicy = v1.PullNever
	}
}
