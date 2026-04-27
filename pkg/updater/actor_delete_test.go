package updater

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
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
}

func (c *deleteOptionRecorderClient) Delete(ctx context.Context, obj ctrlclient.Object, opts ...ctrlclient.DeleteOption) error {
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

	err := act.deleteInstance(context.Background(), runtime.FromTiProxy(obj))
	require.NoError(t, err)
	require.Nil(t, cli.lastDeleteOptions.PropagationPolicy)
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

	err := act.deleteInstance(context.Background(), runtime.FromTiKV(obj))
	require.NoError(t, err)
	require.Nil(t, cli.lastDeleteOptions.PropagationPolicy)
}
