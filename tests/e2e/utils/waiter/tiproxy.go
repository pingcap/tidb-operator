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

func WaitForTiProxysHealthy(ctx context.Context, c client.Client, proxyg *v1alpha1.TiProxyGroup, timeout time.Duration) error {
	list := v1alpha1.TiProxyList{}
	return WaitForList(ctx, c, &list, func() error {
		errs := []error{}
		if len(list.Items) != int(*proxyg.Spec.Replicas) {
			errs = append(errs, fmt.Errorf("proxy %s/%s replicas %d not equal to %d", proxyg.Namespace, proxyg.Name, len(list.Items), *proxyg.Spec.Replicas))
		}
		for i := range list.Items {
			tiproxy := &list.Items[i]
			if err := checkInstanceStatus(v1alpha1.LabelValComponentTiProxy, tiproxy.Name, tiproxy.Namespace, tiproxy.Generation, tiproxy.Status.CommonStatus); err != nil {
				errs = append(errs, err)
			}
		}

		return errors.NewAggregate(errs)
	}, timeout, client.InNamespace(proxyg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   proxyg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     proxyg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiProxy,
	})
}
