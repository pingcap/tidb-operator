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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	pkgclient "github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
)

const testUpdateRevision = "rev-current"

func newGracefulTiProxyAct(t *testing.T, cli pkgclient.Client, outdated ...*runtime.TiProxy) actor[runtime.TiProxyTuple, *v1alpha1.TiProxy, *runtime.TiProxy] {
	t.Helper()
	return actor[runtime.TiProxyTuple, *v1alpha1.TiProxy, *runtime.TiProxy]{
		c:            cli,
		converter:    runtime.TiProxyTuple{},
		beingOffline: NewState[*runtime.TiProxy](nil),
		outdated:     NewState(outdated),
		rev:          testUpdateRevision,
	}
}

func gracefulTiProxy(name, revision string, extraAnnotations map[string]string) *v1alpha1.TiProxy {
	annotations := map[string]string{
		v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
	}
	for k, v := range extraAnnotations {
		annotations[k] = v
	}
	return &v1alpha1.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       "ns",
			ResourceVersion: "1",
			UID:             types.UID(name),
			Labels: map[string]string{
				v1alpha1.LabelKeyInstanceRevisionHash: revision,
			},
			Annotations: annotations,
		},
	}
}

func TestDeleteInstanceGracefulTiProxyScaleInMarksOffline(t *testing.T) {
	t.Parallel()

	obj := gracefulTiProxy("tiproxy-a", testUpdateRevision, nil)
	cli := pkgclient.NewFakeClient(obj)
	act := newGracefulTiProxyAct(t, cli)

	err := act.deleteInstance(context.Background(), runtime.FromTiProxy(obj))
	require.NoError(t, err)

	actual := &v1alpha1.TiProxy{}
	require.NoError(t, cli.Get(context.Background(), ctrlclient.ObjectKeyFromObject(obj), actual))
	require.NotNil(t, actual.Spec.Offline)
	assert.True(t, *actual.Spec.Offline)
}

func TestDeleteInstanceGracefulTiProxyOutdatedRevisionDeletesDirectly(t *testing.T) {
	t.Parallel()

	obj := gracefulTiProxy("tiproxy-old", "rev-old", nil)
	cli := pkgclient.NewFakeClient(obj)
	act := newGracefulTiProxyAct(t, cli)

	err := act.deleteInstance(context.Background(), runtime.FromTiProxy(obj))
	require.NoError(t, err)

	actual := &v1alpha1.TiProxy{}
	err = cli.Get(context.Background(), ctrlclient.ObjectKeyFromObject(obj), actual)
	require.Error(t, err)
}

func TestDeleteInstanceGracefulTiProxyDeferDeleteDeletesDirectly(t *testing.T) {
	t.Parallel()

	obj := gracefulTiProxy("tiproxy-a", testUpdateRevision, map[string]string{
		v1alpha1.AnnoKeyDeferDelete: v1alpha1.AnnoValTrue,
	})
	cli := pkgclient.NewFakeClient(obj)
	act := newGracefulTiProxyAct(t, cli)

	err := act.deleteInstance(context.Background(), runtime.FromTiProxy(obj))
	require.NoError(t, err)

	actual := &v1alpha1.TiProxy{}
	err = cli.Get(context.Background(), ctrlclient.ObjectKeyFromObject(obj), actual)
	require.Error(t, err)
}

func TestDeleteInstanceGracefulTiProxySkipsOfflineWhileRollingReplaceInProgress(t *testing.T) {
	t.Parallel()

	obj := gracefulTiProxy("tiproxy-a", testUpdateRevision, nil)
	outdated := gracefulTiProxy("tiproxy-old", "rev-old", nil)
	cli := pkgclient.NewFakeClient(obj, outdated)
	act := newGracefulTiProxyAct(t, cli, runtime.FromTiProxy(outdated))

	err := act.deleteInstance(context.Background(), runtime.FromTiProxy(obj))
	require.NoError(t, err)

	actual := &v1alpha1.TiProxy{}
	err = cli.Get(context.Background(), ctrlclient.ObjectKeyFromObject(obj), actual)
	require.Error(t, err)
	assert.Equal(t, 0, act.beingOffline.Len())
}
