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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1 "github.com/pingcap/tidb-operator/pkg/manager/member/startscript/v1"
	v2 "github.com/pingcap/tidb-operator/pkg/manager/member/startscript/v2"
)

var (
	tikv = map[v1alpha1.StartScriptVersion]func(*v1alpha1.TidbCluster, string) (string, error){
		v1alpha1.StartScriptV1: v1.RenderTiKVStartScript,
		v1alpha1.StartScriptV2: v2.RenderTiKVStartScript,
	}
	pd = map[v1alpha1.StartScriptVersion]func(*v1alpha1.TidbCluster, string) (string, error){
		v1alpha1.StartScriptV1: v1.RenderPDStartScript,
		v1alpha1.StartScriptV2: v2.RenderPDStartScript,
	}
	tidb = map[v1alpha1.StartScriptVersion]func(*v1alpha1.TidbCluster) (string, error){
		v1alpha1.StartScriptV1: v1.RenderTiDBStartScript,
		v1alpha1.StartScriptV2: v2.RenderTiDBStartScript,
	}
	pump = map[v1alpha1.StartScriptVersion]func(*v1alpha1.TidbCluster) (string, error){
		v1alpha1.StartScriptV1: v1.RenderPumpStartScript,
		v1alpha1.StartScriptV2: v2.RenderPumpStartScript,
	}
	ticdc = map[v1alpha1.StartScriptVersion]func(*v1alpha1.TidbCluster, string) (string, error){
		v1alpha1.StartScriptV1: v1.RenderTiCDCStartScript,
		v1alpha1.StartScriptV2: v2.RenderTiCDCStartScript,
	}
	tiflash = map[v1alpha1.StartScriptVersion]func(*v1alpha1.TidbCluster) (string, error){
		v1alpha1.StartScriptV1: v1.RenderTiFlashStartScript,
		v1alpha1.StartScriptV2: v2.RenderTiFlashStartScript,
	}
)

func RenderTiKVStartScript(tc *v1alpha1.TidbCluster, tikvDataVolumeMountPath string) (string, error) {
	return tikv[tc.StartScriptVersion()](tc, tikvDataVolumeMountPath)
}

func RenderPDStartScript(tc *v1alpha1.TidbCluster, pdDataVolumeMountPath string) (string, error) {
	return pd[tc.StartScriptVersion()](tc, pdDataVolumeMountPath)
}

func RenderTiDBStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	return tidb[tc.StartScriptVersion()](tc)
}

func RenderPumpStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	return pump[tc.StartScriptVersion()](tc)
}

func RenderTiCDCStartScript(tc *v1alpha1.TidbCluster, ticdcCertPath string) (string, error) {
	return ticdc[tc.StartScriptVersion()](tc, ticdcCertPath)
}

func RenderTiFlashStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	return tiflash[tc.StartScriptVersion()](tc)
}
