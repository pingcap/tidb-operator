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
	"github.com/pingcap/tidb-operator/pkg/client"
)

func WaitForTSOsHealthy(ctx context.Context, c client.Client, tg *v1alpha1.TSOGroup, timeout time.Duration) error {
	list := v1alpha1.TSOList{}
	return WaitForList(ctx, c, &list, func() error {
		errs := []error{}
		if len(list.Items) != int(*tg.Spec.Replicas) {
			errs = append(errs, fmt.Errorf("tso %s/%s replicas %d not equal to %d", tg.Namespace, tg.Name, len(list.Items), *tg.Spec.Replicas))
		}
		for i := range list.Items {
			tso := &list.Items[i]
			if err := checkInstanceStatus(v1alpha1.LabelValComponentTSO, tso.Name, tso.Namespace, tso.Generation, tso.Status.CommonStatus); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.NewAggregate(errs)
	}, timeout, client.InNamespace(tg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   tg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     tg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTSO,
	})
}
