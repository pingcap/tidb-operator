package apicall

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

func ListInstances[
	S scope.Group[F, T],
	I client.Object,
	F client.Object,
	T runtime.Group,
](ctx context.Context, c client.Client, g F) ([]I, error) {
	l := scope.NewInstanceList[S]()
	if err := c.List(ctx, l, client.InNamespace(g.GetNamespace()), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   coreutil.Cluster[S](g),
		v1alpha1.LabelKeyGroup:     g.GetName(),
		v1alpha1.LabelKeyComponent: scope.Component[S](),
	}); err != nil {
		return nil, err
	}

	objs := make([]I, 0, meta.LenList(l))
	if err := meta.EachListItem(l, func(item kuberuntime.Object) error {
		obj, ok := item.(I)
		if !ok {
			// unreachable
			return fmt.Errorf("cannot convert item")
		}
		objs = append(objs, obj)
		return nil
	}); err != nil {
		// unreachable
		return nil, err
	}

	return objs, nil
}
