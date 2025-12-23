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

func WaitForTiKVWorkersHealthy(ctx context.Context, c client.Client, wg *v1alpha1.TiKVWorkerGroup, timeout time.Duration) error {
	list := v1alpha1.TiKVWorkerList{}
	return WaitForList(ctx, c, &list, func() error {
		errs := []error{}
		if len(list.Items) != int(*wg.Spec.Replicas) {
			errs = append(errs, fmt.Errorf("tikv worker %s/%s replicas %d not equal to %d", wg.Namespace, wg.Name, len(list.Items), *wg.Spec.Replicas))
		}
		for i := range list.Items {
			w := &list.Items[i]
			if err := checkInstanceStatus(v1alpha1.LabelValComponentTiKVWorker, w.Name, w.Namespace, w.Generation, w.Status.CommonStatus); err != nil {
				errs = append(errs, err)
			}
		}

		return errors.NewAggregate(errs)
	}, timeout, client.InNamespace(wg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   wg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     wg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKVWorker,
	})
}
