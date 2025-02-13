package topology

import (
	"context"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"k8s.io/apimachinery/pkg/api/meta"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
)

func ListGroups[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](ctx context.Context, c client.Client, ns, cluster string) ([]F, error) {
	l := scope.NewList[S]()
	if err := c.List(ctx, l, client.InNamespace(ns), client.MatchingFields{
		"spec.cluster.name": cluster,
	}); err != nil {
		return nil, err
	}

	objs := make([]F, 0, meta.LenList(l))
	if err := meta.EachListItem(l, func(item kuberuntime.Object) error {
		obj, ok := item.(F)
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
