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
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
)

var (
	// node labels that can be used as tidb DC label Name
	topologyZoneLabels = []string{"zone", corev1.LabelTopologyZone}

	// tidb DC label Name
	tidbDCLabel = "zone"
)

type TaskServerLabels struct {
	Client client.Client
	Logger logr.Logger
}

func NewTaskServerLabels(logger logr.Logger, c client.Client) task.Task[ReconcileContext] {
	return &TaskServerLabels{
		Client: c,
		Logger: logger,
	}
}

func (*TaskServerLabels) Name() string {
	return "ServerLabels"
}

func (t *TaskServerLabels) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	if !rtx.Healthy || rtx.Pod == nil || rtx.PodIsTerminating {
		return task.Complete().With("skip sync server labels as the instance is not healthy")
	}

	nodeName := rtx.Pod.Spec.NodeName
	if nodeName == "" {
		return task.Fail().With("pod %s/%s has not been scheduled", rtx.TiDB.Namespace, rtx.TiDB.Name)
	}
	var node corev1.Node
	if err := t.Client.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
		return task.Fail().With("failed to get node %s: %s", nodeName, err)
	}

	// TODO: too many API calls to PD?
	pdCfg, err := rtx.PDClient.GetConfig(ctx)
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
	if err := rtx.TiDBClient.SetServerLabels(ctx, serverLabels); err != nil {
		return task.Fail().With("failed to set server labels: %s", err)
	}

	return task.Complete().With("server labels synced")
}
