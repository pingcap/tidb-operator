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

package coreutil

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestTiProxyClientPort(t *testing.T) {
	tiproxy := &v1alpha1.TiProxy{}
	require.Equal(t, int32(v1alpha1.DefaultTiProxyPortClient), TiProxyClientPort(tiproxy))

	tiproxy.Spec.Server.Ports.Client = &v1alpha1.TiProxyPortOrRange{
		Range: &v1alpha1.TiProxyPortRange{
			Start: 10000,
			End:   10002,
		},
	}
	require.Equal(t, int32(v1alpha1.DefaultTiProxyPortClient), TiProxyClientPort(tiproxy))
	require.Equal(t, []int{10000, 10002}, TiProxyClientPortRange(tiproxy))

	tiproxy.Spec.Server.Ports.Client.Port = ptr.To[int32](7000)
	require.Equal(t, int32(7000), TiProxyClientPort(tiproxy))
}

func TestTiProxyGroupClientPort(t *testing.T) {
	proxyg := &v1alpha1.TiProxyGroup{}
	require.Equal(t, int32(v1alpha1.DefaultTiProxyPortClient), TiProxyGroupClientPort(proxyg))

	proxyg.Spec.Template.Spec.Server.Ports.Client = &v1alpha1.TiProxyPortOrRange{
		Range: &v1alpha1.TiProxyPortRange{
			Start: 10000,
			End:   10002,
		},
	}
	require.Equal(t, int32(v1alpha1.DefaultTiProxyPortClient), TiProxyGroupClientPort(proxyg))

	proxyg.Spec.Template.Spec.Server.Ports.Client.Port = ptr.To[int32](7000)
	require.Equal(t, int32(7000), TiProxyGroupClientPort(proxyg))
}
