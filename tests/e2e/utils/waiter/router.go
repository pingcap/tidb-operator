// Copyright 2026 PingCAP, Inc.
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

func WaitForRoutersHealthy(ctx context.Context, c client.Client, rg *v1alpha1.RouterGroup, timeout time.Duration) error {
	list := v1alpha1.RouterList{}
	return WaitForList(ctx, c, &list, func() error {
		errs := []error{}
		if len(list.Items) != int(*rg.Spec.Replicas) {
			errs = append(errs, fmt.Errorf("router %s/%s replicas %d not equal to %d", rg.Namespace, rg.Name, len(list.Items), *rg.Spec.Replicas))
		}
		for i := range list.Items {
			s := &list.Items[i]
			if err := checkInstanceStatus(v1alpha1.LabelValComponentRouter, s.Name, s.Namespace, s.Generation, s.Status.CommonStatus); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.NewAggregate(errs)
	}, timeout, client.InNamespace(rg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   rg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     rg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentRouter,
	})
}
