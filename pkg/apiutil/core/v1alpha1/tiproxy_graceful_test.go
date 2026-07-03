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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestGracefulShutdownHelpers(t *testing.T) {
	t.Parallel()

	proxy := &v1alpha1.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "120",
			},
		},
	}

	seconds, ok, err := GracefulShutdownDeleteDelaySeconds(proxy)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, int32(120), seconds)
	assert.True(t, GracefulScaleInEnabled(proxy))

	startAt := time.Now().Add(-30 * time.Second)
	remaining, enabled, err := GracefulShutdownRemainingFromSources(time.Now(), proxy.Annotations, map[string]string{
		v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime: startAt.Format(time.RFC3339Nano),
	})
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Greater(t, remaining, 80*time.Second)
}

func TestGracefulShutdownBeginTimeFromSources(t *testing.T) {
	t.Parallel()

	now := time.Now()
	earlier := now.Add(-2 * time.Hour).Format(time.RFC3339Nano)
	later := now.Add(-10 * time.Minute).Format(time.RFC3339Nano)

	startAt := GracefulShutdownBeginTimeFromSources(
		map[string]string{v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime: later},
		map[string]string{v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime: earlier},
	)
	require.False(t, startAt.IsZero())
	assert.Equal(t, earlier, startAt.Format(time.RFC3339Nano))
}

func TestHasGracefulDrainState(t *testing.T) {
	t.Parallel()

	assert.False(t, HasGracefulDrainState(nil))
	assert.True(t, HasGracefulDrainState(map[string]string{
		v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime: time.Now().Format(time.RFC3339Nano),
	}))
	assert.True(t, HasGracefulDrainState(map[string]string{
		v1alpha1.AnnoKeyTiProxyGracefulShutdownConnectionsDrained: v1alpha1.AnnoValTrue,
	}))
}
