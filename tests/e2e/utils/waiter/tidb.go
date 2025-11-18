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

package waiter

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/errors"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func WaitForTiDBsHealthy(ctx context.Context, c client.Client, dbg *v1alpha1.TiDBGroup, timeout time.Duration) error {
	list := v1alpha1.TiDBList{}
	return WaitForList(ctx, c, &list, func() error {
		errs := []error{}
		if len(list.Items) != int(*dbg.Spec.Replicas) {
			errs = append(errs, fmt.Errorf("db %s/%s replicas %d not equal to %d", dbg.Namespace, dbg.Name, len(list.Items), *dbg.Spec.Replicas))
		}
		for i := range list.Items {
			db := &list.Items[i]
			if err := checkInstanceStatus(v1alpha1.LabelValComponentTiDB, db.Name, db.Namespace, db.Generation, db.Status.CommonStatus); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.NewAggregate(errs)
	}, timeout, client.InNamespace(dbg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   dbg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     dbg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
	})
}
