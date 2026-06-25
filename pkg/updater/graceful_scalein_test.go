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

package updater

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

func TestNeedsGracefulOfflineScaleIn(t *testing.T) {
	t.Parallel()

	proxy := &runtime.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "60",
			},
		},
	}
	assert.True(t, needsGracefulOfflineScaleIn(proxy))

	proxy.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds] = "0"
	assert.False(t, needsGracefulOfflineScaleIn(proxy))
}

func TestChooseBeingOfflineToReviveLIFO(t *testing.T) {
	t.Parallel()

	now := time.Now()
	older := &runtime.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "older",
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
				v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime:          now.Add(-2 * time.Hour).Format(time.RFC3339Nano),
			},
		},
	}
	newer := &runtime.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "newer",
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
				v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime:          now.Add(-10 * time.Minute).Format(time.RFC3339Nano),
			},
		},
	}
	justOffline := &runtime.TiProxy{
		ObjectMeta: metav1.ObjectMeta{Name: "just-offline"},
		Spec: v1alpha1.TiProxySpec{
			Offline: ptr.To(true),
		},
	}

	chosen, ok := chooseBeingOfflineToRevive([]*runtime.TiProxy{older, newer, justOffline}, now)
	assert.True(t, ok)
	assert.Equal(t, "just-offline", chosen.GetName())

	chosen, ok = chooseBeingOfflineToRevive([]*runtime.TiProxy{older, newer}, now)
	assert.True(t, ok)
	assert.Equal(t, "newer", chosen.GetName())
}

func TestChooseBeingOfflineToReviveSkipsNearDeletion(t *testing.T) {
	t.Parallel()

	now := time.Now()
	delay := 10 * time.Minute
	almostDone := &runtime.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "almost-done",
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: strconv.Itoa(int(delay.Seconds())),
				v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime:          now.Add(-delay + time.Minute).Format(time.RFC3339Nano),
			},
		},
	}
	revivable := &runtime.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "revivable",
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: strconv.Itoa(int(delay.Seconds())),
				v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime:          now.Add(-2 * time.Minute).Format(time.RFC3339Nano),
			},
		},
	}

	chosen, ok := chooseBeingOfflineToRevive([]*runtime.TiProxy{almostDone, revivable}, now)
	assert.True(t, ok)
	assert.Equal(t, "revivable", chosen.GetName())

	_, ok = chooseBeingOfflineToRevive([]*runtime.TiProxy{almostDone}, now)
	assert.False(t, ok)
}

func TestCancelOneOffliningSkipsUpdateAtCurrentRevision(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	rev := "2"
	proxy := &runtime.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "proxy-0",
			Namespace:       metav1.NamespaceDefault,
			ResourceVersion: "10",
			Labels:          map[string]string{v1alpha1.LabelKeyInstanceRevisionHash: rev},
			Generation:      2,
		},
		Spec: v1alpha1.TiProxySpec{
			Offline: ptr.To(true),
		},
		Status: v1alpha1.TiProxyStatus{
			CommonStatus: v1alpha1.CommonStatus{
				ObservedGeneration: 2,
				CurrentRevision:    rev,
			},
		},
	}

	fakeCli := client.NewFakeClient(proxy.To())
	act := &actor[runtime.TiProxyTuple, *v1alpha1.TiProxy, *runtime.TiProxy]{
		c:            fakeCli,
		f:            NewFunc[*runtime.TiProxy](func() *runtime.TiProxy { return &runtime.TiProxy{} }),
		beingOffline: NewState([]*runtime.TiProxy{proxy}),
		update:       NewState[*runtime.TiProxy](nil),
		rev:          rev,
		actions:      make([]action, 0),
	}

	require.NoError(t, act.cancelOneOfflining(ctx, proxy))
	assert.Equal(t, []action{actionCancelOffline}, act.RecordedActions())
	assert.Equal(t, 1, act.update.Len())
	assert.Equal(t, 0, act.beingOffline.Len())

	actual := &v1alpha1.TiProxy{}
	require.NoError(t, fakeCli.Get(ctx, client.ObjectKeyFromObject(proxy.To()), actual))
	assert.False(t, coreutil.IsOffline[scope.TiProxy](actual))
}

func TestRevivableForScaleOutUsesMinRemainingConstant(t *testing.T) {
	t.Parallel()

	now := time.Now()
	delay := time.Hour
	annotations := map[string]string{
		v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: strconv.Itoa(int(delay.Seconds())),
	}

	annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime] = now.Add(-delay + time.Minute).Format(time.RFC3339Nano)
	ok, err := coreutil.RevivableForGracefulScaleOutFromSources(now, annotations)
	require.NoError(t, err)
	assert.False(t, ok)

	annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime] = now.Add(-delay + coreutil.MinRemainingToReviveBeforeDelete).Format(time.RFC3339Nano)
	ok, err = coreutil.RevivableForGracefulScaleOutFromSources(now, annotations)
	require.NoError(t, err)
	assert.True(t, ok)
}
