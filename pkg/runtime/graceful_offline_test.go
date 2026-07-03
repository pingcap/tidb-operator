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

package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestGracefulOfflineScaleInEnabled(t *testing.T) {
	t.Parallel()

	assert.False(t, GracefulOfflineScaleInEnabled(nil))
	assert.False(t, GracefulOfflineScaleInEnabled(map[string]string{}))
	assert.True(t, GracefulOfflineScaleInEnabled(map[string]string{
		v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
	}))
	assert.False(t, GracefulOfflineScaleInEnabled(map[string]string{
		v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "0",
	}))
}

func TestTiProxySupportsOffline(t *testing.T) {
	t.Parallel()

	proxy := &TiProxy{}
	assert.False(t, proxy.SupportsOffline())

	proxy.SetAnnotations(map[string]string{
		v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
	})
	assert.True(t, proxy.SupportsOffline())
}
