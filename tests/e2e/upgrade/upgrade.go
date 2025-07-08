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

//go:build upgrade_e2e

package upgrade

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/k8s"
)

const (
	operatorNs              = "tidb-admin"
	operatorDeployName      = "tidb-operator"
	newVersionOperatorImage = "pingcap/tidb-operator:latest"

	createClusterTimeout = 10 * time.Minute
	createClusterPolling = 10 * time.Second
)

// RunCmd runs a command and returns its output.
func runCmd(cmd string) (string, error) {
	// Find project root using similar method as hack/lib/e2e.sh
	// This file is at tests/e2e/upgrade/upgrade.go, so project root is ../../../
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}

	// Get the directory containing this file (tests/e2e/upgrade)
	currentDir := filepath.Dir(currentFile)
	// Go up 3 levels to reach project root
	projectRoot := filepath.Join(currentDir, "..", "..", "..")

	// Convert to absolute path
	absProjectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path of project root: %w", err)
	}

	finalCmd := cmd
	if strings.HasPrefix(cmd, "kubectl") {
		kubectlPath := filepath.Join(absProjectRoot, "_output", "bin", "kubectl")
		finalCmd = strings.Replace(cmd, "kubectl", kubectlPath, 1)
	}

	fullCmd := fmt.Sprintf("cd %s && %s", absProjectRoot, finalCmd)
	output, err := exec.Command("bash", "-c", fullCmd).CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("failed to run command: %s, output: %s, error: %w", fullCmd, string(output), err)
	}
	return string(output), nil
}

var _ = ginkgo.PDescribe("Upgrade TiDB Operator", label.P0, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("should not restart pods after upgrade", label.P0, func() {
		ginkgo.It("with basic spec", func(ctx context.Context) {
			ginkgo.By("Check if the old version operator is running")
			deploy := &appsv1.Deployment{}
			err := f.Client.Get(ctx, client.ObjectKey{
				Namespace: operatorNs,
				Name:      operatorDeployName,
			}, deploy)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(deploy.Status.ReadyReplicas).To(gomega.BeNumerically(">=", 1))
			gomega.Expect(deploy.Spec.Template.Spec.Containers[0].Image).NotTo(gomega.Equal(newVersionOperatorImage))

			ginkgo.By("Deploy a tidb cluster with old version operator")
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)
			// proxyg := f.MustCreateTiProxy(ctx)
			// flashg := f.MustCreateTiFlash(ctx)
			// cdcg := f.MustCreateTiCDC(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			// f.WaitForTiProxyGroupReady(ctx, proxyg)
			// f.WaitForTiFlashGroupReady(ctx, flashg)
			// f.WaitForTiCDCGroupReady(ctx, cdcg)

			ginkgo.By("Recording pod UIDs before upgrade")
			podList := &corev1.PodList{}
			err = f.Client.List(ctx, podList, client.InNamespace(pdg.Namespace))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			originalPodUIDs := k8s.GetPodUIDMapFromPodList(ctx, podList)

			ginkgo.By("Upgrading operator and CRDs")
			_, err = runCmd("kubectl apply --server-side=true -f manifests/crd/")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			deploy.Spec.Template.Spec.Containers[0].Image = newVersionOperatorImage
			gomega.Expect(f.Client.Update(ctx, deploy)).NotTo(gomega.HaveOccurred())

			ginkgo.By("Waiting for new operator to be ready")
			gomega.Eventually(func(g gomega.Gomega) {
				err := f.Client.Get(ctx, client.ObjectKey{
					Namespace: operatorNs,
					Name:      operatorDeployName,
				}, deploy)
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(deploy.Status.UpdatedReplicas).To(gomega.BeNumerically(">=", 1))
				g.Expect(deploy.Status.ReadyReplicas).To(gomega.Equal(deploy.Status.Replicas))
			}).WithTimeout(3 * time.Minute).WithPolling(createClusterPolling).Should(gomega.Succeed())

			ginkgo.By("Verifying pods are not restarted")
			gomega.Consistently(func(g gomega.Gomega) {
				err := f.Client.List(ctx, podList, client.InNamespace(pdg.Namespace))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				currentPodUIDs := k8s.GetPodUIDMapFromPodList(ctx, podList)
				g.Expect(currentPodUIDs).To(gomega.Equal(originalPodUIDs))
			}).WithTimeout(3 * time.Minute).WithPolling(createClusterPolling).Should(gomega.Succeed())
		})
	})
})
