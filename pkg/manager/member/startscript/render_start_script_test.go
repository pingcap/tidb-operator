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
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

	"github.com/onsi/gomega"
)

func TestRenderStartScriptRoute(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		name      string
		ver       v1alpha1.StartScriptVersion
		expectVer v1alpha1.StartScriptVersion
	}

	cases := []testcase{
		{
			name:      "v1",
			ver:       v1alpha1.StartScriptV1,
			expectVer: v1alpha1.StartScriptV1,
		},
		{
			name:      "v2",
			ver:       v1alpha1.StartScriptV2,
			expectVer: v1alpha1.StartScriptV2,
		},
		{
			name:      "empty version",
			ver:       v1alpha1.StartScriptVersion(""),
			expectVer: v1alpha1.StartScriptV1,
		},
		{
			name:      "unsupported version",
			ver:       v1alpha1.StartScriptVersion("v100"),
			expectVer: v1alpha1.StartScriptV1,
		},
	}

	for _, c := range cases {
		t.Logf("test case: %s", c.name)

		tc := &v1alpha1.TidbCluster{}
		tc.Spec.StartScriptVersion = c.ver
		mockRenderFunc(c.expectVer)

		for _, render := range []Render{
			RenderTiKVStartScript,
			RenderPDStartScript,
			RenderTiDBStartScript,
			RenderPumpStartScript,
			RenderTiCDCStartScript,
			RenderTiFlashStartScript,
			RenderTiFlashInitScript,
		} {
			_, err := render(tc)
			g.Expect(err).Should(gomega.Succeed())
		}

	}
}

func mockRenderFunc(expectedVer v1alpha1.StartScriptVersion) {
	for _, scriptMap := range []RenderMap{tikv, pd, ticdc, tidb, pump, tiflash, tiflashInit} {
		for ver := range scriptMap {
			var err error
			if ver != expectedVer {
				err = fmt.Errorf("should use render func for version %s", expectedVer)
			}
			scriptMap[ver] = func(tc *v1alpha1.TidbCluster) (string, error) {
				return "", err
			}
		}
	}
}
