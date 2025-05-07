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

package apicall

import (
	"cmp"
	"context"
	"slices"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

func ListInstances[
	GS scope.GroupInstance[GF, GT, IS],
	IS scope.List[IL, I],
	GF client.Object,
	GT runtime.Group,
	IL client.ObjectList,
	I client.Object,
](ctx context.Context, c client.Client, g GF) ([]I, error) {
	l := scope.NewList[IS]()
	if err := c.List(ctx, l, client.InNamespace(g.GetNamespace()), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   coreutil.Cluster[GS](g),
		v1alpha1.LabelKeyGroup:     g.GetName(),
		v1alpha1.LabelKeyComponent: scope.Component[GS](),
	}); err != nil {
		return nil, err
	}

	objs := scope.GetItems[IS](l)

	// always sort instances
	slices.SortFunc(objs, func(a, b I) int {
		return cmp.Compare(a.GetName(), b.GetName())
	})

	return objs, nil
}
