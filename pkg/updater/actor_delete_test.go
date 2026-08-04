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

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	pkgclient "github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
)

type deleteOptionRecorderClient struct {
	pkgclient.Client
	lastDeleteOptions ctrlclient.DeleteOptions
	deleteCalls       int
}

func (c *deleteOptionRecorderClient) Delete(ctx context.Context, obj ctrlclient.Object, opts ...ctrlclient.DeleteOption) error {
	c.deleteCalls++
	c.lastDeleteOptions = ctrlclient.DeleteOptions{}
	c.lastDeleteOptions.ApplyOptions(opts)
	return c.Client.Delete(ctx, obj, opts...)
}

func TestDeleteInstanceDoesNotOrphanTiProxyDependents(t *testing.T) {
	t.Parallel()

	obj := &v1alpha1.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "tiproxy-a",
			Namespace:       "ns",
			ResourceVersion: "1",
			UID:             types.UID("tiproxy-a"),
		},
	}
	cli := &deleteOptionRecorderClient{Client: pkgclient.NewFakeClient(obj)}
	act := actor[runtime.TiProxyTuple, *v1alpha1.TiProxy, *runtime.TiProxy]{
		c:         cli,
		converter: runtime.TiProxyTuple{},
	}

	err := act.deleteInstance(context.Background(), runtime.FromTiProxy(obj), false)
	require.NoError(t, err)
	require.Equal(t, 1, cli.deleteCalls)
	require.Nil(t, cli.lastDeleteOptions.PropagationPolicy)
}

func TestDeleteInstanceGracefulTiProxyDeferDeleteDeletesCR(t *testing.T) {
	t.Parallel()

	obj := &v1alpha1.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "tiproxy-old",
			Namespace:       "ns",
			ResourceVersion: "1",
			UID:             types.UID("tiproxy-old"),
			Annotations: map[string]string{
				v1alpha1.AnnoKeyDeferDelete:                               v1alpha1.AnnoValTrue,
				v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
			},
		},
	}
	cli := &deleteOptionRecorderClient{Client: pkgclient.NewFakeClient(obj)}
	act := actor[runtime.TiProxyTuple, *v1alpha1.TiProxy, *runtime.TiProxy]{
		c:                    cli,
		converter:            runtime.TiProxyTuple{},
		directDeleteOutdated: true,
	}

	err := act.deleteInstance(context.Background(), runtime.FromTiProxy(obj), true)
	require.NoError(t, err)
	require.Equal(t, 1, cli.deleteCalls)
	require.Nil(t, cli.lastDeleteOptions.PropagationPolicy)
}

func TestDeleteInstanceGracefulTiProxyOutdatedDirectDeletesCR(t *testing.T) {
	t.Parallel()

	// Rolling replace via scaleInOutdated(..., deferDel=false) never writes
	// defer-delete, but WithDirectDeleteOutdated still forces a hard delete.
	obj := &v1alpha1.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "tiproxy-old",
			Namespace:       "ns",
			ResourceVersion: "1",
			UID:             types.UID("tiproxy-old"),
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
			},
		},
	}
	cli := &deleteOptionRecorderClient{Client: pkgclient.NewFakeClient(obj)}
	act := actor[runtime.TiProxyTuple, *v1alpha1.TiProxy, *runtime.TiProxy]{
		c:                    cli,
		converter:            runtime.TiProxyTuple{},
		directDeleteOutdated: true,
	}

	err := act.deleteInstance(context.Background(), runtime.FromTiProxy(obj), true)
	require.NoError(t, err)
	require.Equal(t, 1, cli.deleteCalls)
	require.Nil(t, cli.lastDeleteOptions.PropagationPolicy)

	got := &v1alpha1.TiProxy{}
	err = cli.Get(context.Background(), ctrlclient.ObjectKeyFromObject(obj), got)
	require.True(t, apierrors.IsNotFound(err))
}

func TestCleanupDirectDeletesGracefulTiProxyWithDeferDelete(t *testing.T) {
	t.Parallel()

	obj := &v1alpha1.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "tiproxy-old",
			Namespace:       "ns",
			ResourceVersion: "1",
			UID:             types.UID("tiproxy-old"),
			Annotations: map[string]string{
				v1alpha1.AnnoKeyDeferDelete:                               v1alpha1.AnnoValTrue,
				v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
			},
		},
	}
	cli := &deleteOptionRecorderClient{Client: pkgclient.NewFakeClient(obj)}
	inst := runtime.FromTiProxy(obj)
	act := actor[runtime.TiProxyTuple, *v1alpha1.TiProxy, *runtime.TiProxy]{
		c:                    cli,
		converter:            runtime.TiProxyTuple{},
		directDeleteOutdated: true,
		deleted:              NewState([]*runtime.TiProxy{inst}),
	}

	require.NoError(t, act.Cleanup(context.Background()))
	require.Equal(t, 1, cli.deleteCalls)

	got := &v1alpha1.TiProxy{}
	err := cli.Get(context.Background(), ctrlclient.ObjectKeyFromObject(obj), got)
	require.True(t, apierrors.IsNotFound(err))
}

func TestScaleInOutdatedDirectDeletesGracefulTiProxyWithoutDeferDelete(t *testing.T) {
	t.Parallel()

	obj := &v1alpha1.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "tiproxy-old",
			Namespace:       "ns",
			ResourceVersion: "1",
			UID:             types.UID("tiproxy-old"),
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
			},
		},
	}
	cli := &deleteOptionRecorderClient{Client: pkgclient.NewFakeClient(obj)}
	inst := runtime.FromTiProxy(obj)
	act := actor[runtime.TiProxyTuple, *v1alpha1.TiProxy, *runtime.TiProxy]{
		c:                    cli,
		converter:            runtime.TiProxyTuple{},
		directDeleteOutdated: true,
		outdated:             NewState([]*runtime.TiProxy{inst}),
	}

	_, err := act.scaleInOutdated(context.Background(), obj.Name, false)
	require.NoError(t, err)
	require.Equal(t, 1, cli.deleteCalls)

	got := &v1alpha1.TiProxy{}
	err = cli.Get(context.Background(), ctrlclient.ObjectKeyFromObject(obj), got)
	require.True(t, apierrors.IsNotFound(err))
}

func TestDeleteInstanceGracefulTiProxyScaleInMarksOffline(t *testing.T) {
	t.Parallel()

	obj := &v1alpha1.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "tiproxy-a",
			Namespace:       "ns",
			ResourceVersion: "1",
			UID:             types.UID("tiproxy-a"),
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds: "3600",
			},
		},
	}
	cli := &deleteOptionRecorderClient{Client: pkgclient.NewFakeClient(obj)}
	act := actor[runtime.TiProxyTuple, *v1alpha1.TiProxy, *runtime.TiProxy]{
		c:                    cli,
		converter:            runtime.TiProxyTuple{},
		directDeleteOutdated: true,
		beingOffline:         NewState([]*runtime.TiProxy{}),
	}

	err := act.deleteInstance(context.Background(), runtime.FromTiProxy(obj), false)
	require.NoError(t, err)
	require.Equal(t, 0, cli.deleteCalls)

	got := &v1alpha1.TiProxy{}
	require.NoError(t, cli.Get(context.Background(), ctrlclient.ObjectKeyFromObject(obj), got))
	require.NotNil(t, got.Spec.Offline)
	require.True(t, *got.Spec.Offline)
}

func TestDeleteInstanceDoesNotOrphanNonTiProxyDependents(t *testing.T) {
	t.Parallel()

	obj := &v1alpha1.TiKV{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "tikv-a",
			Namespace:       "ns",
			ResourceVersion: "1",
			UID:             types.UID("tikv-a"),
		},
		Spec: v1alpha1.TiKVSpec{
			Offline: ptr.To(true),
		},
	}
	cli := &deleteOptionRecorderClient{Client: pkgclient.NewFakeClient(obj)}
	act := actor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
		c:         cli,
		converter: runtime.TiKVTuple{},
	}

	err := act.deleteInstance(context.Background(), runtime.FromTiKV(obj), false)
	require.NoError(t, err)
	require.Equal(t, 1, cli.deleteCalls)
	require.Nil(t, cli.lastDeleteOptions.PropagationPolicy)
}
