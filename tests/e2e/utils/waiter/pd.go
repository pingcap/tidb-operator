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

func WaitForPDsHealthy(ctx context.Context, c client.Client, pdg *v1alpha1.PDGroup, timeout time.Duration) error {
	list := v1alpha1.PDList{}
	return WaitForList(ctx, c, &list, func() error {
		if len(list.Items) != int(*pdg.Spec.Replicas) {
			return fmt.Errorf("pd %s/%s replicas %d not equal to %d", pdg.Namespace, pdg.Name, len(list.Items), *pdg.Spec.Replicas)
		}
		for i := range list.Items {
			pd := &list.Items[i]
			if pd.Generation != pd.Status.ObservedGeneration {
				return fmt.Errorf("pd %s/%s is not synced", pd.Namespace, pd.Name)
			}
			if !meta.IsStatusConditionPresentAndEqual(pd.Status.Conditions, v1alpha1.PDCondInitialized, metav1.ConditionTrue) {
				return fmt.Errorf("pd %s/%s is not initialized", pd.Namespace, pd.Name)
			}
			if !meta.IsStatusConditionPresentAndEqual(pd.Status.Conditions, v1alpha1.CondReady, metav1.ConditionTrue) {
				return fmt.Errorf("pd %s/%s is not healthy", pd.Namespace, pd.Name)
			}
		}

		return nil
	}, timeout, client.InNamespace(pdg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   pdg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     pdg.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
	})
}
