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

package image

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	"k8s.io/kubernetes/test/e2e/framework"
)

const (
	TiDBV3Version                 = "v3.1.1"
	TiDBV3UpgradeVersion          = "v3.1.2"
	TiDBV4Version                 = "v4.0.4"
	TiDBV4UpgradeVersion          = "v4.0.5"
	PrometheusImage               = "prom/prometheus"
	PrometheusVersion             = "v2.18.1"
	TiDBMonitorReloaderImage      = "pingcap/tidb-monitor-reloader"
	TiDBMonitorReloaderVersion    = "v1.0.1"
	TiDBMonitorInitializerImage   = "pingcap/tidb-monitor-initializer"
	TiDBMonitorInitializerVersion = "v3.0.8"
	GrafanaImage                  = "grafana/grafana"
	GrafanaVersion                = "6.1.6"
)

func ListImages() []string {
	images := []string{}
	versions := make([]string, 0)
	versions = append(versions, TiDBV3Version)
	versions = append(versions, TiDBV3UpgradeVersion)
	versions = append(versions, TiDBV4Version)
	versions = append(versions, TiDBV4UpgradeVersion)
	for _, v := range versions {
		images = append(images, fmt.Sprintf("pingcap/pd:%s", v))
		images = append(images, fmt.Sprintf("pingcap/tidb:%s", v))
		images = append(images, fmt.Sprintf("pingcap/tikv:%s", v))
	}
	images = append(images, fmt.Sprintf("%s:%s", PrometheusImage, PrometheusVersion))
	images = append(images, fmt.Sprintf("%s:%s", TiDBMonitorReloaderImage, TiDBMonitorReloaderVersion))
	images = append(images, fmt.Sprintf("%s:%s", TiDBMonitorInitializerImage, TiDBMonitorInitializerVersion))
	images = append(images, fmt.Sprintf("%s:%s", GrafanaImage, GrafanaVersion))
	imagesFromOperator, err := readImagesFromValues(filepath.Join(framework.TestContext.RepoRoot, "charts/tidb-operator/values.yaml"), sets.NewString(".advancedStatefulset.image", ".admissionWebhook.jobImage"))
	if err != nil {
		framework.ExpectNoError(err)
	}
	images = append(images, imagesFromOperator...)
	imageKeysFromTiDBCluster := sets.NewString(".pd.image", ".tikv.image", ".tidb.image")
	imagesFromTiDBCluster, err := readImagesFromValues(filepath.Join(framework.TestContext.RepoRoot, "charts/tidb-cluster/values.yaml"), imageKeysFromTiDBCluster)
	if err != nil {
		framework.ExpectNoError(err)
	}
	images = append(images, imagesFromTiDBCluster...)
	return sets.NewString(images...).List()
}

// values represents a collection of chart values.
type values map[string]interface{}

func walkValues(vals values, parentKey string, fn func(k string, v interface{})) {
	for k, v := range vals {
		fn(parentKey+"."+k, v)
		valsMap, ok := v.(map[string]interface{})
		if ok {
			walkValues(valsMap, parentKey+"."+k, fn)
		}
	}
}

func readImagesFromValues(f string, keys sets.String) ([]string, error) {
	var vals values
	data, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &vals)
	if err != nil {
		return nil, err
	}
	if len(vals) == 0 {
		vals = values{}
	}
	images := []string{}
	walkValues(vals, "", func(k string, v interface{}) {
		if keys != nil && !keys.Has(k) {
			return
		}
		if image, ok := v.(string); ok {
			images = append(images, image)
		}
	})
	return images, nil
}

func nsenter(args ...string) ([]byte, error) {
	nsenter_args := []string{
		"--mount=/rootfs/proc/1/ns/mnt",
		fmt.Sprintf("--wd=%s", framework.TestContext.RepoRoot),
		"--",
	}
	nsenter_args = append(nsenter_args, args...)
	klog.Infof("run nsenter command: %s %s", "nsenter", strings.Join(nsenter_args, " "))
	return exec.Command("nsenter", nsenter_args...).CombinedOutput()
}

// PreloadImages pre-loads images into the e2e cluster.
// This is used to speed up the e2e process.
// NOTE: it supports kind only right now
func PreloadImages() error {
	images := ListImages()
	// TODO: make it configurable
	cluster := "tidb-operator"
	kindBin := "./output/bin/kind"
	output, err := nsenter(kindBin, "get", "nodes", "--name", cluster)
	if err != nil {
		return err
	}
	nodes := []string{}
	for _, l := range strings.Split(string(output), "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if strings.HasSuffix(l, "-control-plane") {
			continue
		}
		nodes = append(nodes, l)
	}
	for _, image := range images {
		if _, err := nsenter("docker", "pull", image); err != nil {
			klog.Errorf("preloadImages, error pulling image %s", image)
			continue
		}
		if output, err := nsenter(kindBin, "load", "docker-image", "--name", cluster, "--nodes", strings.Join(nodes, ","), image); err != nil {
			klog.Errorf("error preloading image %s", output)
			return err
		}
	}
	for _, image := range images {
		if output, err := nsenter("docker", "rmi", image); err != nil {
			klog.Errorf("error cleaning up image %s", output)
			return err
		}
	}
	return nil
}
