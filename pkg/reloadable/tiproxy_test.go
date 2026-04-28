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

package reloadable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestCheckTiProxy(t *testing.T) {
	cases := []struct {
		desc       string
		proxyg     *v1alpha1.TiProxyGroup
		proxy      *v1alpha1.TiProxy
		reloadable bool
	}{
		{
			desc: "delete delay seconds is reloadable",
			proxyg: &v1alpha1.TiProxyGroup{
				Spec: v1alpha1.TiProxyGroupSpec{
					Template: v1alpha1.TiProxyTemplate{
						Spec: v1alpha1.TiProxyTemplateSpec{
							GracefulShutdownDeleteDelaySeconds: ptr.To[int32](20),
						},
					},
				},
			},
			proxy: &v1alpha1.TiProxy{
				Spec: v1alpha1.TiProxySpec{
					TiProxyTemplateSpec: v1alpha1.TiProxyTemplateSpec{},
				},
			},
			reloadable: true,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert.Equal(t, c.reloadable, CheckTiProxy(c.proxyg, c.proxy), c.desc)
		})
	}
}
