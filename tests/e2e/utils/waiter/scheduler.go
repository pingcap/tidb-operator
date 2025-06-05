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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
)

func WaitForSchedulersHealthy(ctx context.Context, c client.Client, sg *v1alpha1.SchedulerGroup, timeout time.Duration) error {
	list := v1alpha1.SchedulerList{}
	return WaitForList(ctx, c, &list, func() error {
		if len(list.Items) != int(*sg.Spec.Replicas) {
			return fmt.Errorf("scheduler %s/%s replicas %d not equal to %d", sg.Namespace, sg.Name, len(list.Items), *sg.Spec.Replicas)
		}
		for i := range list.Items {
			s := &list.Items[i]
			if err := checkInstanceStatus(v1alpha1.LabelValComponentScheduler, s.Name, s.Namespace, s.Generation, s.Status.CommonStatus); err != nil {
				return err
			}
		}

		return nil
	}, timeout, client.InNamespace(sg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   sg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     sg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentScheduler,
	})
}
