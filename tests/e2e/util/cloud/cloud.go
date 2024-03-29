// Copyright 2020 PingCAP, Inc.
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

package cloud

import (
	"os/exec"
	"strings"

	framework "github.com/pingcap/tidb-operator/tests/third_party/k8s"
	"github.com/pingcap/tidb-operator/tests/third_party/k8s/log"
)

func getClusterLocation() string {
	if isRegionalCluster() {
		return "--region=" + framework.TestContext.CloudConfig.Region
	}
	return "--zone=" + framework.TestContext.CloudConfig.Zone
}

func getGcloudCommandFromTrack(commandTrack string, args []string) []string {
	command := []string{"gcloud"}
	if commandTrack == "beta" || commandTrack == "alpha" {
		command = append(command, commandTrack)
	}
	command = append(command, args...)
	command = append(command, getClusterLocation())
	command = append(command, "--project="+framework.TestContext.CloudConfig.ProjectID)
	return command
}

func getGcloudCommand(args []string) []string {
	track := ""
	if isRegionalCluster() {
		track = "beta"
	}
	return getGcloudCommandFromTrack(track, args)
}

func isRegionalCluster() bool {
	return framework.TestContext.CloudConfig.MultiZone
}

func execCmd(args ...string) *exec.Cmd {
	log.Logf("Executing: %s", strings.Join(args, " "))
	return exec.Command(args[0], args[1:]...)
}

func DisableNodeAutoRepair() {
	if framework.TestContext.Provider == "gke" {
		// https://cloud.google.com/kubernetes-engine/docs/how-to/node-auto-repair
		nodePool := "default-pool"
		log.Logf("Using gcloud to disable auto-repair for pool %s", nodePool)
		args := []string{"container", "node-pools", "update", "default-pool", "--cluster", framework.TestContext.CloudConfig.Cluster,
			"--no-enable-autorepair"}
		gcloudCommand := getGcloudCommand(args)
		output, err := execCmd(gcloudCommand...).CombinedOutput()
		log.Logf("Config update result: %s", output)
		framework.ExpectNoError(err, "failed to get gcloud command: %q", gcloudCommand)
	} else {
		// TODO support AWS (EKS)
		log.Failf("unsupported provider %q", framework.TestContext.Provider)
	}
}

func EnableNodeAutoRepair() {
	if framework.TestContext.Provider == "gke" {
		// https://cloud.google.com/kubernetes-engine/docs/how-to/node-auto-repair
		nodePool := "default-pool"
		log.Logf("Using gcloud to disable auto-repair for pool %s", nodePool)
		args := []string{"container", "node-pools", "update", "default-pool", "--cluster", framework.TestContext.CloudConfig.Cluster,
			"--enable-autorepair"}
		gcloudCommand := getGcloudCommand(args)
		output, err := execCmd(gcloudCommand...).CombinedOutput()
		log.Logf("Config update result: %s", output)
		framework.ExpectNoError(err, "failed to get gcloud command: %q", gcloudCommand)
	} else {
		// TODO support AWS (EKS)
		log.Failf("unsupported provider %q", framework.TestContext.Provider)
	}
}
