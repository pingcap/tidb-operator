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

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
)

const (
	operatorNs              = "tidb-admin"
	operatorDeployName      = "tidb-operator"
	newVersionOperatorImage = "pingcap/tidb-operator:latest"

	createClusterTimeout = 10 * time.Minute
	createClusterPolling = 10 * time.Second
)

// RunCmd runs a command and returns its output.
func runCmd(ctx context.Context, cmd string) (string, error) {
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
	output, err := exec.CommandContext(ctx, "bash", "-c", fullCmd).CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("failed to run command: %s, output: %s, error: %w", fullCmd, string(output), err)
	}
	return string(output), nil
}

var _ = ginkgo.Describe("Upgrade TiDB Operator", label.P0, func() {
	f := framework.New()
	f.Setup()

	// JustAfterEach runs before the framework deletes the cluster and namespace.
	ginkgo.JustAfterEach(func(ctx context.Context) {
		if !ginkgo.CurrentSpecReport().Failed() {
			return
		}
		commands := []string{
			"source hack/lib/e2e.sh && e2e::dump_operator",
			"kubectl --request-timeout=30s get crd clusters.core.pingcap.com -o yaml",
		}
		if f.Namespace != nil {
			ns := f.Namespace.Name
			commands = append(commands,
				fmt.Sprintf("kubectl --request-timeout=30s -n %s get clusters,pdgroups,pds,tikvgroups,tidbgroups -o yaml", ns),
				fmt.Sprintf("kubectl --request-timeout=30s -n %s describe pods", ns),
				fmt.Sprintf("kubectl --request-timeout=30s -n %s get events --sort-by=.metadata.creationTimestamp", ns),
			)
		}
		for _, cmd := range commands {
			cmdCtx, cancel := context.WithTimeout(ctx, time.Minute)
			output, err := runCmd(cmdCtx, cmd)
			cancel()
			ginkgo.GinkgoWriter.Printf("Diagnostics: %s\n%s\n", cmd, output)
			if err != nil {
				ginkgo.GinkgoWriter.Printf("Cannot collect diagnostics: %v\n", err)
			}
		}
	})

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

			nctx, cancel := context.WithCancel(ctx)
			pdDone := framework.AsyncWaitPodsRollingUpdateOnce[scope.PDGroup](nctx, f, pdg, int(*pdg.Spec.Replicas), true)
			defer func() { cancel(); <-pdDone }()
			kvDone := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiKVGroup](nctx, f, kvg, int(*kvg.Spec.Replicas), true)
			defer func() { cancel(); <-kvDone }()
			dbDone := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiDBGroup](nctx, f, dbg, int(*dbg.Spec.Replicas), true)
			defer func() { cancel(); <-dbDone }()
			ginkgo.By("Upgrading operator")
			patch := client.MergeFrom(deploy.DeepCopy())
			deploy.Spec.Template.Spec.Containers[0].Image = newVersionOperatorImage
			gomega.Expect(f.Client.Patch(ctx, deploy, patch)).NotTo(gomega.HaveOccurred())

			ginkgo.By("Waiting for new operator to be ready")
			generation := deploy.Generation
			gomega.Eventually(func(g gomega.Gomega) {
				err := f.Client.Get(ctx, client.ObjectKey{
					Namespace: operatorNs,
					Name:      operatorDeployName,
				}, deploy)
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(deploy.Status.ObservedGeneration).To(gomega.BeNumerically(">=", generation))
				g.Expect(deploy.Spec.Replicas).NotTo(gomega.BeNil())
				g.Expect(*deploy.Spec.Replicas).To(gomega.BeNumerically(">=", 1))
				g.Expect(deploy.Status.UpdatedReplicas).To(gomega.Equal(*deploy.Spec.Replicas))
				g.Expect(deploy.Status.Replicas).To(gomega.Equal(*deploy.Spec.Replicas))
				g.Expect(deploy.Status.AvailableReplicas).To(gomega.Equal(*deploy.Spec.Replicas))
			}).WithTimeout(3 * time.Minute).WithPolling(createClusterPolling).Should(gomega.Succeed())

			ginkgo.By("Verifying pods are not restarted")
			timer := time.NewTimer(3 * time.Minute)
			defer timer.Stop()
			select {
			case <-timer.C:
			case <-ctx.Done():
				f.Must(ctx.Err())
			}
			cancel()
			<-pdDone
			<-kvDone
			<-dbDone

		})
	})
})
