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

package tasks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/updater"
)

const testUpdateRevision = "rev-current"

func gracefulTiProxy(name, revision string) *v1alpha1.TiProxy {
	return &v1alpha1.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       "ns",
			ResourceVersion: "1",
			UID:             types.UID(name),
			Labels: map[string]string{
				v1alpha1.LabelKeyInstanceRevisionHash: revision,
			},
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
			},
		},
	}
}

func newGracefulExecutor(
	t *testing.T,
	cli client.Client,
	desired int,
	instances ...*runtime.TiProxy,
) updater.Executor {
	t.Helper()
	return updater.New[runtime.TiProxyTuple]().
		WithInstances(instances...).
		WithDesired(desired).
		WithClient(cli).
		WithRevision(testUpdateRevision).
		WithCancelOfflineFilterPolicy(
			updater.FilterOutdated[*runtime.TiProxy](testUpdateRevision),
			updater.FilterReviveAbandoned[*runtime.TiProxy](),
		).
		WithNewFactory(updater.NewFunc[*runtime.TiProxy](func() *runtime.TiProxy {
			return &runtime.TiProxy{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns"},
				Spec:       v1alpha1.TiProxySpec{Offline: ptr.To(false)},
			}
		})).
		Build()
}

func TestExecutorScaleInUpdateMarksOffline(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	obj := gracefulTiProxy("tiproxy-a", testUpdateRevision)
	cli := client.NewFakeClient(obj)

	_, err := newGracefulExecutor(t, cli, 0, runtime.FromTiProxy(obj)).Do(ctx)
	require.NoError(t, err)

	actual := &v1alpha1.TiProxy{}
	require.NoError(t, cli.Get(ctx, ctrlclient.ObjectKeyFromObject(obj), actual))
	require.NotNil(t, actual.Spec.Offline)
	assert.True(t, *actual.Spec.Offline)
}

func TestExecutorScaleInUpdateDeletesDuringRollingReplace(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	currentA := gracefulTiProxy("tiproxy-a", testUpdateRevision)
	currentB := gracefulTiProxy("tiproxy-b", testUpdateRevision)
	outdated := gracefulTiProxy("tiproxy-old", "rev-old")
	cli := client.NewFakeClient(currentA, currentB, outdated)

	_, err := newGracefulExecutor(t, cli, 1,
		runtime.FromTiProxy(currentA),
		runtime.FromTiProxy(currentB),
		runtime.FromTiProxy(outdated),
	).Do(ctx)
	require.NoError(t, err)

	remaining := 0
	for _, name := range []string{"tiproxy-a", "tiproxy-b", "tiproxy-old"} {
		err = cli.Get(ctx, ctrlclient.ObjectKey{Namespace: "ns", Name: name}, &v1alpha1.TiProxy{})
		if err == nil {
			remaining++
		}
	}
	assert.Equal(t, 2, remaining)
}

func TestExecutorScaleOutRevivesGracefulOfflineInstance(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	obj := gracefulTiProxy("tiproxy-a", testUpdateRevision)
	obj.Spec.Offline = ptr.To(true)
	cli := client.NewFakeClient(obj)

	_, err := newGracefulExecutor(t, cli, 2, runtime.FromTiProxy(obj)).Do(ctx)
	require.NoError(t, err)

	actual := &v1alpha1.TiProxy{}
	require.NoError(t, cli.Get(ctx, ctrlclient.ObjectKeyFromObject(obj), actual))
	assert.False(t, coreutil.IsOffline[scope.TiProxy](actual))
}

func TestExecutorScaleOutSkipsOutdatedOfflineInstance(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	outdated := gracefulTiProxy("tiproxy-old", "rev-old")
	outdated.Spec.Offline = ptr.To(true)
	current := gracefulTiProxy("tiproxy-a", testUpdateRevision)
	cli := client.NewFakeClient(outdated, current)

	_, err := newGracefulExecutor(t, cli, 3,
		runtime.FromTiProxy(outdated),
		runtime.FromTiProxy(current),
	).Do(ctx)
	require.NoError(t, err)

	revived := &v1alpha1.TiProxy{}
	require.NoError(t, cli.Get(ctx, ctrlclient.ObjectKeyFromObject(current), revived))
	assert.False(t, coreutil.IsOffline[scope.TiProxy](revived))

	stillOffline := &v1alpha1.TiProxy{}
	require.NoError(t, cli.Get(ctx, ctrlclient.ObjectKeyFromObject(outdated), stillOffline))
	assert.True(t, coreutil.IsOffline[scope.TiProxy](stillOffline))
}

func TestExecutorScaleOutSkipsReviveAbandonedInstance(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	abandoned := gracefulTiProxy("tiproxy-a", testUpdateRevision)
	abandoned.Spec.Offline = ptr.To(true)
	abandoned.Annotations[v1alpha1.AnnoKeyTiProxyReviveAbandoned] = v1alpha1.AnnoValTrue
	current := gracefulTiProxy("tiproxy-b", testUpdateRevision)
	cli := client.NewFakeClient(abandoned, current)

	_, err := newGracefulExecutor(t, cli, 3,
		runtime.FromTiProxy(abandoned),
		runtime.FromTiProxy(current),
	).Do(ctx)
	require.NoError(t, err)

	stillAbandoned := &v1alpha1.TiProxy{}
	require.NoError(t, cli.Get(ctx, ctrlclient.ObjectKeyFromObject(abandoned), stillAbandoned))
	assert.True(t, coreutil.IsOffline[scope.TiProxy](stillAbandoned))
	assert.Equal(t, v1alpha1.AnnoValTrue, stillAbandoned.Annotations[v1alpha1.AnnoKeyTiProxyReviveAbandoned])

	unchanged := &v1alpha1.TiProxy{}
	require.NoError(t, cli.Get(ctx, ctrlclient.ObjectKeyFromObject(current), unchanged))
	assert.False(t, coreutil.IsOffline[scope.TiProxy](unchanged))

	tiproxies := &v1alpha1.TiProxyList{}
	require.NoError(t, cli.List(ctx, tiproxies, ctrlclient.InNamespace("ns")))
	assert.Len(t, tiproxies.Items, 3)

	created := 0
	for i := range tiproxies.Items {
		tp := &tiproxies.Items[i]
		if tp.Name == "tiproxy-a" || tp.Name == "tiproxy-b" {
			continue
		}
		created++
		assert.False(t, coreutil.IsOffline[scope.TiProxy](tp))
	}
	assert.Equal(t, 1, created)
}
