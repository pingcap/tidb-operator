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

	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

var (
	// node labels that can be used as tidb DC label Name
	topologyZoneLabels = []string{"zone", corev1.LabelTopologyZone}

	// tidb DC label Name
	tidbDCLabel = "zone"
)

func TaskServerLabels(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ServerLabels", func(ctx context.Context) task.Result {
		if !state.IsHealthy() || state.Pod() == nil || state.IsPodTerminating() {
			return task.Complete().With("skip sync server labels as the instance is not healthy")
		}
		if state.PDClient == nil {
			return task.Complete().With("skip sync server labels as the pd client is not ready")
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
		pdCfg, err := state.PDClient.GetConfig(ctx)
		if err != nil {
			return task.Fail().With("failed to get pd config: %s", err)
		}

		var zoneLabel string
	outer:
		for _, zl := range topologyZoneLabels {
			for _, ll := range pdCfg.Replication.LocationLabels {
				if ll == zl {
					zoneLabel = zl
					break outer
				}
			}
		}
		if zoneLabel == "" {
			return task.Complete().With("zone labels not found in pd location-label, skip sync server labels")
		}

		serverLabels := k8s.GetNodeLabelsForKeys(&node, pdCfg.Replication.LocationLabels)
		if len(serverLabels) == 0 {
			return task.Complete().With("no server labels from node %s to sync", nodeName)
		}
		serverLabels[tidbDCLabel] = serverLabels[zoneLabel]

		// TODO: is there any way to avoid unnecessary update?
		if err := state.TiDBClient.SetServerLabels(ctx, serverLabels); err != nil {
			return task.Fail().With("failed to set server labels: %s", err)
		}

		return task.Complete().With("server labels synced")
	})
}
