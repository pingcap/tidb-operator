// Copyright 2019. PingCAP, Inc.
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

package check

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
	"github.com/fatih/color"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	config_v2_1_16 "github.com/pingcap/tidb-operator/pkg/config/v2_1_16"
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/engine"
	helm_env "k8s.io/helm/pkg/helm/environment"
	chartproto "k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/timeconv"
	tversion "k8s.io/helm/pkg/version"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"sigs.k8s.io/yaml"
)

const (
	kubeVersion   = "v1.12.8"
	checkLongDesc = `
                  Preflight check for TiDB cluster configuration.
`
	checkExample = `
      # check config from local chart
      tkctl check -t demo --chart charts/tidb-cluster --values charts/tidb-cluster/values.yaml

      # check config from remote chart
      tkctl check -t demo --chart pingcap/tidb-cluster --values charts/tidb-cluster/values.yaml --version v1.1.0-alpha.2
`
	// AWS load balancer annotations
	awsILBAnnotationKey    = "service.beta.kubernetes.io/aws-load-balancer-internal"
	awsILBAnnotationVal    = "0.0.0.0/0"
	awsLBTypeAnnotationKey = "service.beta.kubernetes.io/aws-load-balancer-type"
	awsLBTypeAnnotationVal = "nlb"
	// GCP load balancer annotations
	gcpLBTypeAnnotationKey = "cloud.google.com/load-balancer-type"
	gcpLBTypeAnnotationVal = "Internal"
	// Aliyun load balancer annotations
	aliyunLBAddressTypeAnnotationKey  = "service.beta.kubernetes.io/alicloud-loadbalancer-address-type"
	aliyunLBAddressTypeAnnotationVal  = "intranet"
	aliyunSLBNetworkTypeAnnotationKey = "service.beta.kubernetes.io/alicloud-loadbalancer-slb-network-type"
	aliyunSLBNetworkTypeAnnotationVal = "vpc"
)

var (
	errorPrintf  = color.New(color.FgRed).PrintfFunc()
	errorPrintln = color.New(color.FgRed).PrintlnFunc()
	warnPrintf   = color.New(color.FgYellow).PrintfFunc()
	warnPrintln  = color.New(color.FgYellow).PrintlnFunc()
	infoPrintf   = color.New(color.FgGreen).PrintfFunc()
	infoPrintln  = color.New(color.FgGreen).PrintlnFunc()
	settings     helm_env.EnvSettings
)

type CheckOptions struct {
	TidbClusterName string
	Namespace       string
	Chart           string
	ChartVersion    string
	ValuesFile      string
	Cloud           string
	DryRun          bool

	genericclioptions.IOStreams
}

// NewCheckOptions return a CheckOptions
func NewCheckOptions(streams genericclioptions.IOStreams) *CheckOptions {
	return &CheckOptions{
		IOStreams: streams,
	}
}

// NewCmdCheck creates the check command which checks the tidb cluster configuration
func NewCmdCheck(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	options := NewCheckOptions(streams)
	cmd := &cobra.Command{
		Use:     "check",
		Short:   "check TiDB cluster configuration",
		Long:    checkLongDesc,
		Example: checkExample,
		Run: func(_ *cobra.Command, _ []string) {
			cmdutil.CheckErr(options.Run())
		},
	}
	cmd.Flags().BoolVarP(&options.DryRun, "dry-run", "", true, "whether dry-run without cluster")
	cmd.Flags().StringVarP(&options.Cloud, "cloud", "", "", "cloud provider")
	cmd.Flags().StringVarP(&options.ValuesFile, "values", "f", "tidb-cluster.yaml", "tidb-cluster values file")
	cmd.Flags().StringVarP(&options.Chart, "chart", "", "pingcap/tidb-cluster", "tidb cluster chart path, local path or remote repo path")
	cmd.Flags().StringVarP(&options.ChartVersion, "chart-version", "", "v1.0.0", "tidb cluster chart version")
	flags := cmd.PersistentFlags()
	settings.AddFlags(flags)
	settings.Init(flags)
	return cmd
}

func (o *CheckOptions) Run() error {
	var tidbcluster v1alpha1.TidbCluster
	var tidbConfigMap corev1.ConfigMap
	var tikvConfigMap corev1.ConfigMap
	var pdConfigMap corev1.ConfigMap
	var pumpConfigMap corev1.ConfigMap
	var drainerConfigMap corev1.ConfigMap
	var pumpSts *appsv1.StatefulSet
	var drainerSts *appsv1.StatefulSet
	var tidbSvc corev1.Service
	out, err := o.renderChart()
	if err != nil {
		return err
	}
	for k, v := range out {
		filename := filepath.Base(k)
		switch filename {
		case "tidb-cluster.yaml":
			if err := yaml.Unmarshal([]byte(v), &tidbcluster); err != nil {
				return err
			}
		case "tidb-configmap.yaml":
			if err := yaml.Unmarshal([]byte(v), &tidbConfigMap); err != nil {
				return err
			}
		case "tikv-configmap.yaml":
			if err := yaml.Unmarshal([]byte(v), &tikvConfigMap); err != nil {
				return err
			}
		case "pd-configmap.yaml":
			if err := yaml.Unmarshal([]byte(v), &pdConfigMap); err != nil {
				return err
			}
		case "pump-configmap-rollout.yaml":
			if err := yaml.Unmarshal([]byte(v), &pumpConfigMap); err != nil {
				return err
			}
		case "drainer-configmap-rollout.yaml":
			if err := yaml.Unmarshal([]byte(v), &drainerConfigMap); err != nil {
				return err
			}
		case "tidb-service.yaml":
			if err := yaml.Unmarshal([]byte(v), &tidbSvc); err != nil {
				return err
			}
		case "pump-statefulset.yaml":
			if err := yaml.Unmarshal([]byte(v), &pumpSts); err != nil {
				return err
			}
		case "drainer-statefulset.yaml":
			if err := yaml.Unmarshal([]byte(v), &drainerSts); err != nil {
				return err
			}
		}
	}
	if !componentVersionMatches(&tidbcluster, pumpSts, drainerSts) {
		errorPrintln("component version mismatch")
		return errors.New("cluster component version mismatch")
	}
	infoPrintln("component version matches, all versions are ", getImageVersion(tidbcluster.Spec.PD.Image))

	o.checkTidbService(tidbSvc)
	checkResourceSettings(&tidbcluster, pumpSts, drainerSts)

	var tidbcfg *config_v2_1_16.TidbConfig
	var pdcfg *config_v2_1_16.PdConfig
	var tikvcfg *config_v2_1_16.TikvConfig
	if cfg, exist := tidbConfigMap.Data["config-file"]; exist {
		infoPrintln("#### TiDB Config ####")
		fmt.Println(cfg)
		infoPrintln("#### TiDB Config ####\n")
		_, err = toml.Decode(cfg, &tidbcfg)
		if err != nil {
			return fmt.Errorf("failed to decode tidb config: %v", err)
		}
		if tidbcfg.Log.Level != "info" {
			warnPrintln("TiDB log-level:", tidbcfg.Log.Level)
		}
		if tidbcfg.Binlog.IgnoreError {
			warnPrintln("TiDB binlog.ignore-error is enabled")
		}
		if tidbcfg.PreparedPlanCache.Enabled {
			warnPrintln("TiDB prepared-plan-cache is experimental feature")
		}
		if tidbcfg.TxnLocalLatches.Enabled {
			warnPrintln("TiDB txn-local-latches is for write conflict heavy business")
		}
	}
	if cfg, exist := tikvConfigMap.Data["config-file"]; exist {
		infoPrintln("#### TiKV Config ####")
		fmt.Println(cfg)
		infoPrintln("#### TiKV Config ####\n")
		_, err = toml.Decode(cfg, &tikvcfg)
		if err != nil {
			return fmt.Errorf("failed to decode tikv config: %v", err)
		}
		tikvResources := util.ResourceRequirement(tidbcluster.Spec.TiKV.ContainerSpec)
		memoryLimit := tikvResources.Limits[corev1.ResourceMemory]
		m, b1 := memoryLimit.AsInt64()
		if b1 {
			// rocksdb.writecf.block-cache-size 10%~30% of memory limit
			// rocksdb.defaultcf.block-cache-size 30%~50% of memory limit
			if int64(tikvcfg.Rocksdb.Writecf.BlockCacheSize) < m/10 || int64(tikvcfg.Rocksdb.Writecf.BlockCacheSize) > 3*m/10 {
				errorPrintf("TiKV rocksdb.writecf.block-cache-size should be 10%% ~ 30%% of memory limit, but got %d/%d\n", tikvcfg.Rocksdb.Writecf.BlockCacheSize, m)
			}
			if int64(tikvcfg.Rocksdb.Defaultcf.BlockCacheSize) < 3*m/10 || int64(tikvcfg.Rocksdb.Defaultcf.BlockCacheSize) > m/2 {
				errorPrintf("TiKV rocksdb.writecf.block-cache-size should be 30%% ~ 50%% of memory limit, but got %d/%d\n", tikvcfg.Rocksdb.Defaultcf.BlockCacheSize, m)
			}
			infoPrintf("TiKV rocksdb.lockcf.block-cache-size: %dMiB\n", tikvcfg.Rocksdb.Lockcf.BlockCacheSize/1024/1024)
		}
		if !tikvcfg.RaftStore.SyncLog {
			errorPrintln("Raftstore sync-log is disable")
		}
		if tikvcfg.LogLevel != "info" {
			warnPrintln("TiKV log-level:", tikvcfg.LogLevel)
		}
	}
	if cfg, exist := pumpConfigMap.Data["pump-config"]; exist {
		infoPrintln("#### Pump Config ####")
		fmt.Println(cfg)
		infoPrintln("#### Pump Config ####\n")
	}
	if cfg, exist := pdConfigMap.Data["config-file"]; exist {
		infoPrintln("#### PD Config ####")
		fmt.Println(cfg)
		infoPrintln("#### PD Config ####\n")
		_, err = toml.Decode(cfg, &pdcfg)
		if err != nil {
			return fmt.Errorf("failed to decode pd config: %v", err)
		}
		if pdcfg.Log.Level != "info" {
			warnPrintln("PD log-level:", pdcfg.Log.Level)
		}
		if pdcfg.Replication.MaxReplicas != 3 {
			warnPrintln("TiKV data replication factor is", pdcfg.Replication.MaxReplicas)
		}
	}
	return nil
}

func checkResourceSettings(tc *v1alpha1.TidbCluster, pump *appsv1.StatefulSet, drainer *appsv1.StatefulSet) {
	pdResources := util.ResourceRequirement(tc.Spec.PD.ContainerSpec)
	tikvResources := util.ResourceRequirement(tc.Spec.TiKV.ContainerSpec)
	tidbResources := util.ResourceRequirement(tc.Spec.TiDB.ContainerSpec)
	if err := compareResources(pdResources); err != nil {
		errorPrintf("PD resources settings (%v) error: %v\n", pdResources, err)
	}
	if err := compareResources(tikvResources); err != nil {
		errorPrintf("TiKV resources settings (%v) error: %v\n", tikvResources, err)
	}
	if err := compareResources(tidbResources); err != nil {
		errorPrintf("TiDB resources settings (%v) error: %v\n", tidbResources, err)
	}

	if container := getContainerFromSts(pump, "pump"); container != nil {
		if err := compareResources(container.Resources); err != nil {
			errorPrintf("Pump resources settings (%v) error: %v\n", container.Resources, err)
		}
	}
	if container := getContainerFromSts(drainer, "drainer"); container != nil {
		if err := compareResources(container.Resources); err != nil {
			errorPrintf("Drainer resources settings (%v) error: %v\n", container.Resources, err)
		}
	}
}

// check requests and limits cpu/memory not zero, requests <= limits, (warn if cpu is not integer, or not Guaranteed QoS)
func compareResources(requirement corev1.ResourceRequirements) error {
	cpuRequest := requirement.Requests[corev1.ResourceCPU]
	cpuLimit := requirement.Limits[corev1.ResourceCPU]
	memoryRequest := requirement.Requests[corev1.ResourceMemory]
	memoryLimit := requirement.Limits[corev1.ResourceMemory]

	if cpuRequest.IsZero() || cpuLimit.IsZero() {
		return fmt.Errorf("cpu request or limit is zero")
	}
	switch cpuRequest.Cmp(cpuLimit) {
	case -1:
		warnPrintln("CPU request < CPU limit, not Guaranteed QoS")
	case 0:
	case 1:
		return fmt.Errorf("CPU request > CPU limit")
	}
	_, b1 := cpuRequest.AsInt64()
	_, b2 := cpuLimit.AsInt64()
	if !b1 || !b2 {
		fmt.Errorf("CPU request or limit cannot convert as integer, taskset will not set")
	}

	if memoryRequest.IsZero() || memoryLimit.IsZero() {
		return fmt.Errorf("memory request or limit is zero")
	}
	switch memoryRequest.Cmp(memoryLimit) {
	case -1:
		warnPrintln("memory request < memory limit, not Guaranteed QoS")
	case 0:
	case 1:
		fmt.Errorf("memory request > memory limit")
	}
	return nil
}

func componentVersionMatches(tc *v1alpha1.TidbCluster, pump *appsv1.StatefulSet, drainer *appsv1.StatefulSet) bool {
	pdVersion := getImageVersion(tc.Spec.PD.Image)
	tikvVersion := getImageVersion(tc.Spec.TiKV.Image)
	tidbVersion := getImageVersion(tc.Spec.TiDB.Image)
	var pumpVersion, drainerVersion string
	if container := getContainerFromSts(drainer, "drainer"); container != nil {
		drainerVersion = getImageVersion(container.Image)
	}
	if container := getContainerFromSts(pump, "pump"); container != nil {
		pumpVersion = getImageVersion(container.Image)
	}
	if pdVersion != "" && pdVersion == tikvVersion &&
		pdVersion == tidbVersion {
		if pump == nil || pdVersion == pumpVersion {
			return true
		}
		if drainer == nil || pdVersion == drainerVersion {
			return true
		}
	}
	errorPrintf("cluster component version mismatch: (pd, %s), (tikv, %s), (tidb, %s), (pump, %s)\n", pdVersion, tikvVersion, tidbVersion, pumpVersion)
	return false
}

func getContainerFromSts(sts *appsv1.StatefulSet, name string) *corev1.Container {
	if sts == nil {
		return nil
	}
	for _, container := range sts.Spec.Template.Spec.Containers {
		if container.Name == name {
			return &container
		}
	}
	return nil
}

func getImageVersion(image string) string {
	str := strings.Split(image, ":")
	if len(str) != 2 && len(str) != 3 {
		return ""
	}
	return str[len(str)-1]
}

func (o *CheckOptions) renderChart() (map[string]string, error) {
	name := o.Chart
	version := o.ChartVersion
	var verify bool
	var repoURL, username, password, keyring, certFile, keyFile, caFile string
	cp, err := locateChartPath(repoURL, username, password, name, version, verify, keyring, certFile, keyFile, caFile)
	if err != nil {
		return nil, err
	}
	infoPrintln("chart path:", cp)

	c, err := chartutil.Load(cp)
	if err != nil {
		return nil, err
	}
	renderer := engine.New()
	caps := &chartutil.Capabilities{
		APIVersions:   chartutil.DefaultVersionSet,
		KubeVersion:   chartutil.DefaultKubeVersion,
		TillerVersion: tversion.GetVersionProto(),
	}

	valuesFile, err := os.Open(o.ValuesFile)
	defer valuesFile.Close()
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(valuesFile)
	if err != nil {
		return nil, err
	}

	config := &chartproto.Config{Raw: string(bytes), Values: map[string]*chartproto.Value{}}
	kv, err := semver.NewVersion(kubeVersion)
	if err != nil {
		return nil, err
	}
	caps.KubeVersion.Major = fmt.Sprint(kv.Major())
	caps.KubeVersion.Minor = fmt.Sprint(kv.Minor())
	caps.KubeVersion.GitVersion = fmt.Sprintf("v%d.%d.0", kv.Major(), kv.Minor())

	options := chartutil.ReleaseOptions{
		Name:      o.TidbClusterName,
		Time:      timeconv.Now(),
		Namespace: o.Namespace,
	}
	vals, err := chartutil.ToRenderValuesCaps(c, config, options, caps)
	if err != nil {
		return nil, err
	}

	return renderer.Render(c, vals)
}

func (o *CheckOptions) checkTidbService(svc corev1.Service) {
	if svc.Spec.Type == "LoadBalancer" || svc.Spec.Type == "NodePort" {
		policy := "Cluster"
		if svc.Spec.ExternalTrafficPolicy == "Local" {
			policy = "Local"
		}
		warnPrintf("External Traffic Policy: %s\n", policy)
	}
	annotations := svc.Annotations
	if o.Cloud != "" {
		if svc.Spec.Type != "LoadBalancer" || annotations == nil {
			errorPrintln("TiDB Service not exposed as load balancer on", o.Cloud)
		}
		switch strings.ToLower(o.Cloud) {
		case "aws":
			if val, exist := annotations[awsILBAnnotationKey]; !exist || val != awsILBAnnotationVal {
				errorPrintf("TiDB Service not using %s internal load balancer\n", o.Cloud)
			}
			if val, exist := annotations[awsLBTypeAnnotationKey]; !exist || val != awsLBTypeAnnotationVal {
				errorPrintf("TiDB Service not using %s network load balancer\n", o.Cloud)
			}
		case "gcp":
			if val, exist := annotations[gcpLBTypeAnnotationKey]; !exist || val != gcpLBTypeAnnotationVal {
				errorPrintf("TiDB Service not using %s internal load balancer\n", o.Cloud)
			}
		case "aliyun":
			if val, exist := annotations[aliyunLBAddressTypeAnnotationKey]; !exist || val != aliyunLBAddressTypeAnnotationVal {
				errorPrintf("TiDB Service not using %s internal load balancer\n", o.Cloud)
			}
			if val, exist := annotations[aliyunSLBNetworkTypeAnnotationKey]; !exist || val != aliyunSLBNetworkTypeAnnotationVal {
				errorPrintf("TiDB Service not using %s vpc load balancer\n", o.Cloud)
			}
		default:
			errorPrintln("Unknown cloud:", o.Cloud)
		}
	}
}
