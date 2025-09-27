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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
)

func WaitForClusterReady(ctx context.Context, c client.Client, ns, name string, timeout time.Duration) error {
	tc := v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}

	return WaitForObject(ctx, c, &tc, func() error {
		if tc.Status.PD == "" {
			return fmt.Errorf("pd is not available")
		}
		cond := meta.FindStatusCondition(tc.Status.Conditions, v1alpha1.ClusterCondAvailable)
		if cond == nil {
			return fmt.Errorf("available cond is unset")
		}
		if cond.Status != metav1.ConditionTrue {
			return fmt.Errorf("available cond is not true, status: %s, reason: %s, message: %s", cond.Status, cond.Reason, cond.Message)
		}

		return nil
	}, timeout)
}

func WaitForClusterPDRegistered(ctx context.Context, c client.Client, tc *v1alpha1.Cluster, timeout time.Duration) error {
	return WaitForObject(ctx, c, tc, func() error {
		if tc.Status.PD == "" {
			return fmt.Errorf("pd is not available")
		}

		return nil
	}, timeout)
}
