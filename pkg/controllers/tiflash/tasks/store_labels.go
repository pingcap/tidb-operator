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

package tasks

import (
	"context"
	"reflect"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/pingcap/kvproto/pkg/metapb"
	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskStoreLabels(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("StoreLabels", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		if state.StoreState != v1alpha1.StoreStateServing || state.PodIsTerminating || state.Pod() == nil {
			return task.Complete().With("skip sync store labels as the store is not serving")
		}

		nodeName := state.Pod().Spec.NodeName
		if nodeName == "" {
			return task.Fail().With("pod %s/%s has not been scheduled", state.Pod().Namespace, state.Pod().Name)
		}

		var node corev1.Node
		if err := c.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
			return task.Fail().With("failed to get node %s: %s", nodeName, err)
		}

		// TODO: too many API calls to PD?
		pdCfg, err := state.PDClient.Underlay().GetConfig(ctx)
		if err != nil {
			return task.Fail().With("failed to get pd config: %s", err)
		}
		keys := pdCfg.Replication.LocationLabels
		if len(keys) == 0 {
			return task.Complete().With("no store labels need to sync")
		}

		storeLabels := k8s.GetNodeLabelsForKeys(&node, keys)
		if len(storeLabels) == 0 {
			return task.Complete().With("no store labels from node %s to sync", nodeName)
		}

		if !storeLabelsEqualNodeLabels(state.StoreLabels, storeLabels) {
			storeID, err := strconv.ParseUint(state.StoreID, 10, 64)
			if err != nil {
				return task.Fail().With("failed to parse store id %s: %s", state.StoreID, err)
			}
			set, err := state.PDClient.Underlay().SetStoreLabels(ctx, storeID, storeLabels)
			if err != nil {
				return task.Fail().With("failed to set store labels: %s", err)
			} else if set {
				logger.Info("store labels synced", "storeID", state.StoreID, "storeLabels", storeLabels)
			}
		}

		return task.Complete().With("store labels synced")
	})
}

func storeLabelsEqualNodeLabels(storeLabels []*metapb.StoreLabel, nodeLabels map[string]string) bool {
	ls := map[string]string{}
	for _, label := range storeLabels {
		key := label.GetKey()
		if _, ok := nodeLabels[key]; ok {
			val := label.GetValue()
			ls[key] = val
		}
	}
	return reflect.DeepEqual(ls, nodeLabels)
}
