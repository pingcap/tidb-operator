// Copyright 2021 PingCAP, Inc.
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

package startscript

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	startscriptv1 "github.com/pingcap/tidb-operator/pkg/manager/member/startscript/v1"
	startscriptv2 "github.com/pingcap/tidb-operator/pkg/manager/member/startscript/v2"
	"k8s.io/klog/v2"
)

func RenderTiKVStartScript(tc *v1alpha1.TidbCluster, tikvDataVolumeMountPath string) (string, error) {
	switch tc.Spec.StartScriptVersion {
	case v1alpha1.StartScriptV2:
		return startscriptv2.RenderTiKVStartScript(tc, tikvDataVolumeMountPath)
	case v1alpha1.StartScriptV1:
		return renderTiKVStartScriptV1(tc, tikvDataVolumeMountPath)
	}

	// use v1 by default
	return renderTiKVStartScriptV1(tc, tikvDataVolumeMountPath)
}

func RenderPDStartScript(tc *v1alpha1.TidbCluster, pdDataVolumeMountPath string) (string, error) {
	switch tc.Spec.StartScriptVersion {
	case v1alpha1.StartScriptV2:
		return startscriptv2.RenderPDStartScript(tc, pdDataVolumeMountPath)
	case v1alpha1.StartScriptV1:
		return renderPDStartScriptV1(tc, pdDataVolumeMountPath)
	}

	// use v1 by default
	return renderPDStartScriptV1(tc, pdDataVolumeMountPath)
}

func RenderTiDBStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	switch tc.Spec.StartScriptVersion {
	case v1alpha1.StartScriptV2:
		return startscriptv2.RenderTiDBStartScript(tc)
	case v1alpha1.StartScriptV1:
		return renderTiDBStartScriptV1(tc)
	}

	// use v1 by default
	return renderTiDBStartScriptV1(tc)
}

func RenderPumpStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	switch tc.Spec.StartScriptVersion {
	case v1alpha1.StartScriptV2:
		return startscriptv2.RenderPumpStartScript(tc)
	case v1alpha1.StartScriptV1:
		return renderPumpStartScriptV1(tc)
	}

	// use v1 by default
	return renderPumpStartScriptV1(tc)
}

func renderTiKVStartScriptV1(tc *v1alpha1.TidbCluster, tikvDataVolumeMountPath string) (string, error) {
	model := &startscriptv1.TiKVStartScriptModel{
		CommonModel: startscriptv1.CommonModel{
			AcrossK8s:     tc.AcrossK8s(),
			ClusterDomain: tc.Spec.ClusterDomain,
		},
		EnableAdvertiseStatusAddr: false,
		DataDir:                   filepath.Join(tikvDataVolumeMountPath, tc.Spec.TiKV.DataSubDir),
	}
	if tc.Spec.EnableDynamicConfiguration != nil && *tc.Spec.EnableDynamicConfiguration {
		model.AdvertiseStatusAddr = "${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc" + controller.FormatClusterDomain(tc.Spec.ClusterDomain)
		model.EnableAdvertiseStatusAddr = true
	}

	model.PDAddress = tc.Scheme() + "://${CLUSTER_NAME}-pd:2379"
	if tc.AcrossK8s() {
		model.PDAddress = tc.Scheme() + "://${CLUSTER_NAME}-pd:2379" // get pd addr from discovery in startup script
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		model.PDAddress = tc.Scheme() + "://" + controller.PDMemberName(tc.Spec.Cluster.Name) + ":2379" // use pd of reference cluster
	}

	return startscriptv1.RenderTiKVStartScript(model)
}

func renderPDStartScriptV1(tc *v1alpha1.TidbCluster, pdDataVolumeMountPath string) (string, error) {
	model := &startscriptv1.PDStartScriptModel{
		CommonModel: startscriptv1.CommonModel{
			AcrossK8s:     tc.AcrossK8s(),
			ClusterDomain: tc.Spec.ClusterDomain,
		},
		Scheme:  tc.Scheme(),
		DataDir: filepath.Join(pdDataVolumeMountPath, tc.Spec.PD.DataSubDir),
	}
	if tc.Spec.PD.StartUpScriptVersion == "v1" {
		model.CheckDomainScript = startscriptv1.CheckDNSV1
	}
	return startscriptv1.RenderPDStartScript(model)
}

func renderTiDBStartScriptV1(tc *v1alpha1.TidbCluster) (string, error) {
	plugins := tc.Spec.TiDB.Plugins
	model := &startscriptv1.TidbStartScriptModel{
		CommonModel: startscriptv1.CommonModel{
			AcrossK8s:     tc.AcrossK8s(),
			ClusterDomain: tc.Spec.ClusterDomain,
		},
		EnablePlugin:    len(plugins) > 0,
		PluginDirectory: "/plugins",
		PluginList:      strings.Join(plugins, ","),
	}

	model.Path = "${CLUSTER_NAME}-pd:2379"
	if tc.AcrossK8s() {
		model.Path = "${CLUSTER_NAME}-pd:2379" // get pd addr from discovery in startup script
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		model.Path = controller.PDMemberName(tc.Spec.Cluster.Name) + ":2379" // use pd of reference cluster
	}

	return startscriptv1.RenderTiDBStartScript(model)
}

func renderPumpStartScriptV1(tc *v1alpha1.TidbCluster) (string, error) {
	scheme := "http"
	if tc.IsTLSClusterEnabled() {
		scheme = "https"
	}

	pdDomain := controller.PDMemberName(tc.Name)
	if tc.AcrossK8s() {
		pdDomain = controller.PDMemberName(tc.Name) // get pd addr from discovery in startup script
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		pdDomain = controller.PDMemberName(tc.Spec.Cluster.Name) // use pd of reference cluster
	}

	pdAddr := fmt.Sprintf("%s://%s:2379", scheme, pdDomain)
	return startscriptv1.RenderPumpStartScript(&startscriptv1.PumpStartScriptModel{
		CommonModel: startscriptv1.CommonModel{
			AcrossK8s:     tc.AcrossK8s(),
			ClusterDomain: tc.Spec.ClusterDomain,
		},
		Scheme:      scheme,
		ClusterName: tc.Name,
		PDAddr:      pdAddr,
		LogLevel:    getPumpLogLevel(tc),
		Namespace:   tc.GetNamespace(),
	})
}

func getPumpLogLevel(tc *v1alpha1.TidbCluster) string {
	defaultPumpLogLevel := "info"

	cfg := tc.Spec.Pump.Config
	if cfg == nil {
		return defaultPumpLogLevel
	}

	v := cfg.Get("log-level")
	if v == nil {
		return defaultPumpLogLevel
	}

	logLevel, err := v.AsString()
	if err != nil {
		klog.Warning("error log-level for pump: ", err)
		return defaultPumpLogLevel
	}

	return logLevel
}
