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

package e2e

import (
	"context"
	"fmt"
	"time"

	//nolint: stylecheck // too many changes, refactor later
	. "github.com/onsi/ginkgo/v2"
	//nolint: stylecheck // too many changes, refactor later
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/k8s"
)

const (
	operatorNamespace      = "tidb-admin"
	operatorDeploymentName = "tidb-operator"
)

var _ = SynchronizedBeforeSuite(func() []byte {
	// TODO(csuzhangxc): options to install operator for upgrade test

	// load kubeconfig
	restConfig, err := k8s.LoadConfig()
	Expect(err).NotTo(HaveOccurred())

	// check if CRD exists
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	Expect(err).NotTo(HaveOccurred())
	crdList, err := discoveryClient.ServerPreferredResources()
	Expect(err).NotTo(HaveOccurred())
	Eventually(func() error {
		for _, resourceList := range crdList {
			if resourceList.GroupVersion == v1alpha1.GroupVersion.String() {
				for _, resource := range resourceList.APIResources {
					if resource.Group == v1alpha1.GroupName && resource.Kind == "Cluster" {
						return nil
					}
				}
			}
		}
		return fmt.Errorf("CRD not found")
	}).WithTimeout(5 * time.Minute).WithPolling(5 * time.Second).Should(Succeed()) //nolint:mnd // refactor to use constant

	// check if operator deployment exists and is running
	clientset, err := kubernetes.NewForConfig(restConfig)
	Expect(err).NotTo(HaveOccurred())
	Eventually(func() error {
		deployment, err2 := clientset.AppsV1().Deployments(operatorNamespace).Get(
			context.Background(), operatorDeploymentName, metav1.GetOptions{})
		if err2 != nil {
			return err2
		}
		if deployment.Status.ReadyReplicas > 0 {
			return nil
		}
		return fmt.Errorf("operator deployment not ready")
	}).WithTimeout(5 * time.Minute).WithPolling(5 * time.Second).Should(Succeed()) //nolint:mnd // refactor to use constant

	// set zone labels for nodes if not exists
	nodeList, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())
	for _, node := range nodeList.Items {
		if _, ok := node.Labels[corev1.LabelTopologyZone]; !ok {
			node.Labels[corev1.LabelTopologyZone] = "az-1" // just for test
			_, err = clientset.CoreV1().Nodes().Update(context.Background(), &node, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())
		}
	}

	return nil
}, func([]byte) {
	// This will run on all nodes
})
