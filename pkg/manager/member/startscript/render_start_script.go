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

func RenderTiKVStartScript(tc *v1alpha1.TidbCluster, tikvDataVolumeMountPath string) (string, error) {
	switch tc.Spec.StartScriptVersion {
	case v1alpha1.StartScriptV2:
		return v2.RenderTiKVStartScript(tc, tikvDataVolumeMountPath)
	case v1alpha1.StartScriptV1:
		return v1.RenderTiKVStartScript(tc, tikvDataVolumeMountPath)
	}

	// use v1 by default
	return v1.RenderTiKVStartScript(tc, tikvDataVolumeMountPath)
}

func RenderPDStartScript(tc *v1alpha1.TidbCluster, pdDataVolumeMountPath string) (string, error) {
	switch tc.Spec.StartScriptVersion {
	case v1alpha1.StartScriptV2:
		return v2.RenderPDStartScript(tc, pdDataVolumeMountPath)
	case v1alpha1.StartScriptV1:
		return v1.RenderPDStartScript(tc, pdDataVolumeMountPath)
	}

	// use v1 by default
	return v1.RenderPDStartScript(tc, pdDataVolumeMountPath)
}

func RenderTiDBStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	switch tc.Spec.StartScriptVersion {
	case v1alpha1.StartScriptV2:
		return v2.RenderTiDBStartScript(tc)
	case v1alpha1.StartScriptV1:
		return v1.RenderTiDBStartScript(tc)
	}

	// use v1 by default
	return v1.RenderTiDBStartScript(tc)
}

func RenderPumpStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	switch tc.Spec.StartScriptVersion {
	case v1alpha1.StartScriptV2:
		return v2.RenderPumpStartScript(tc)
	case v1alpha1.StartScriptV1:
		return v1.RenderPumpStartScript(tc)
	}

	// use v1 by default
	return v1.RenderPumpStartScript(tc)
}

func RenderTiCDCStartScript(tc *v1alpha1.TidbCluster, ticdcCertPath string) (string, error) {
	switch tc.Spec.StartScriptVersion {
	case v1alpha1.StartScriptV2:
		return v2.RenderTiCDCStartScript(tc, ticdcCertPath)
	case v1alpha1.StartScriptV1:
		return v1.RenderTiCDCStartScript(tc, ticdcCertPath)
	}

	// use v1 by default
	return v1.RenderTiCDCStartScript(tc, ticdcCertPath)
}

func RenderTiFlashStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	switch tc.Spec.StartScriptVersion {
	case v1alpha1.StartScriptV2:
		return v2.RenderTiFlashStartScript(tc)
	case v1alpha1.StartScriptV1:
		return v1.RenderTiFlashStartScript(tc)
	}

	// use v1 by default
	return v1.RenderTiFlashStartScript(tc)
}
