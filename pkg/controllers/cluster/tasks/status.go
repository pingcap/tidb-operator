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
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
)

type TaskStatus struct {
	Logger          logr.Logger
	Client          client.Client
	PDClientManager pdm.PDClientManager
}

func NewTaskStatus(logger logr.Logger, c client.Client, pdcm pdm.PDClientManager) task.Task[ReconcileContext] {
	return &TaskStatus{
		Logger:          logger,
		Client:          c,
		PDClientManager: pdcm,
	}
}

func (*TaskStatus) Name() string {
	return "Status"
}

func (t *TaskStatus) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	var needUpdate bool
	if rtx.Cluster.Status.ObservedGeneration != rtx.Cluster.Generation {
		rtx.Cluster.Status.ObservedGeneration = rtx.Cluster.Generation
		needUpdate = true
	}
	if rtx.PDGroup != nil {
		// TODO: extract into a common util
		scheme := "http"
		if rtx.Cluster.IsTLSClusterEnabled() {
			scheme = "https"
		}
		// TODO(liubo02): extract a common util to get pd addr
		pdAddr := fmt.Sprintf("%s://%s-pd.%s:%d", scheme, rtx.PDGroup.Name, rtx.PDGroup.Namespace, rtx.PDGroup.GetClientPort())
		if rtx.Cluster.Status.PD != pdAddr { // TODO(csuzhangxc): verify switch between TLS and non-TLS
			rtx.Cluster.Status.PD = pdAddr
			needUpdate = true
		}
	}
	needUpdate = t.syncComponentStatus(rtx) || needUpdate
	needUpdate = t.syncConditions(rtx) || needUpdate
	needUpdate = t.syncClusterID(ctx, rtx) || needUpdate

	if needUpdate {
		if err := t.Client.Status().Update(ctx, rtx.Cluster); err != nil {
			return task.Fail().With(fmt.Sprintf("can't update cluster status: %v", err))
		}
	}

	if rtx.Cluster.Status.ID == "" {
		// no watch for this, so we need to retry
		//nolint:mnd // only one usage
		return task.Retry(5 * time.Second).With("cluster id is not set")
	}
	return task.Complete().With("updated status")
}

func (*TaskStatus) syncComponentStatus(rtx *ReconcileContext) bool {
	components := make([]v1alpha1.ComponentStatus, 0)
	if rtx.PDGroup != nil {
		pd := v1alpha1.ComponentStatus{Kind: v1alpha1.ComponentKindPD}
		if rtx.PDGroup.Spec.Replicas != nil {
			// TODO: use real replicas
			pd.Replicas += *rtx.PDGroup.Spec.Replicas
		}
		components = append(components, pd)
	}

	if len(rtx.TiKVGroups) > 0 {
		tikv := v1alpha1.ComponentStatus{Kind: v1alpha1.ComponentKindTiKV}
		for _, tikvGroup := range rtx.TiKVGroups {
			if tikvGroup.Spec.Replicas != nil {
				// TODO: use real replicas
				tikv.Replicas += *tikvGroup.Spec.Replicas
			}
		}
		components = append(components, tikv)
	}

	if len(rtx.TiFlashGroups) > 0 {
		tiflash := v1alpha1.ComponentStatus{Kind: v1alpha1.ComponentKindTiFlash}
		for _, tiflashGroup := range rtx.TiFlashGroups {
			if tiflashGroup.Spec.Replicas != nil {
				tiflash.Replicas += *tiflashGroup.Spec.Replicas
			}
		}
		components = append(components, tiflash)
	}

	if len(rtx.TiDBGroups) > 0 {
		tidb := v1alpha1.ComponentStatus{Kind: v1alpha1.ComponentKindTiDB}
		for _, tidbGroup := range rtx.TiDBGroups {
			if tidbGroup.Spec.Replicas != nil {
				tidb.Replicas += *tidbGroup.Spec.Replicas
			}
		}
		components = append(components, tidb)
	}

	sort.Slice(components, func(i, j int) bool {
		return components[i].Kind < components[j].Kind
	})

	if reflect.DeepEqual(rtx.Cluster.Status.Components, components) {
		return false
	}
	rtx.Cluster.Status.Components = components
	return true
}

func (*TaskStatus) syncConditions(rtx *ReconcileContext) bool {
	// TODO(csuzhangxc): calculate progressing condition based on components' observed generation?
	prgCond := metav1.Condition{
		Type:               v1alpha1.ClusterCondProgressing,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: rtx.Cluster.Generation,
		Reason:             v1alpha1.ClusterCreationReason,
		Message:            "Cluster is being created",
	}
	if rtx.Cluster.DeletionTimestamp != nil {
		prgCond.Reason = v1alpha1.ClusterDeletionReason
		prgCond.Message = "Cluster is being deleted"
	}
	changed := meta.SetStatusCondition(&rtx.Cluster.Status.Conditions, prgCond)

	availCond := metav1.Condition{
		Type:               v1alpha1.ClusterCondAvailable,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: rtx.Cluster.Generation,
		Reason:             v1alpha1.ClusterAvailableReason,
		Message:            "Cluster is not available",
	}
	suspended := rtx.PDGroup != nil && meta.IsStatusConditionTrue(rtx.PDGroup.Status.Conditions, v1alpha1.PDGroupCondSuspended)
	for _, tidbg := range rtx.TiDBGroups {
		if meta.IsStatusConditionTrue(tidbg.Status.Conditions, v1alpha1.TiDBGroupCondAvailable) {
			// if any tidb group is available, the cluster is available
			availCond.Status = metav1.ConditionTrue
			availCond.Message = "Cluster is available"
			break
		}
		if !meta.IsStatusConditionTrue(tidbg.Status.Conditions, v1alpha1.TiDBGroupCondSuspended) {
			// if any group is not suspended, the cluster is not suspended
			suspended = false
		}
	}
	changed = meta.SetStatusCondition(&rtx.Cluster.Status.Conditions, availCond) || changed

	if suspended {
		for _, tikvGroup := range rtx.TiKVGroups {
			if !meta.IsStatusConditionTrue(tikvGroup.Status.Conditions, v1alpha1.TiKVGroupCondSuspended) {
				suspended = false
				break
			}
		}
	}
	var (
		suspendStatus  = metav1.ConditionFalse
		suspendMessage = "Cluster is not suspended"
	)
	if suspended {
		suspendStatus = metav1.ConditionTrue
		suspendMessage = "Cluster is suspended"
	} else if rtx.Cluster.ShouldSuspendCompute() {
		suspendMessage = "Cluster is suspending"
	}
	return meta.SetStatusCondition(&rtx.Cluster.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.ClusterCondSuspended,
		Status:             suspendStatus,
		ObservedGeneration: rtx.Cluster.Generation,
		Reason:             v1alpha1.ClusterSuspendReason,
		Message:            suspendMessage,
	}) || changed
}

func (t *TaskStatus) syncClusterID(ctx context.Context, rtx *ReconcileContext) bool {
	if rtx.Cluster.Status.ID != "" {
		// already synced, this will nerver change
		return false
	}

	pdClient, ok := t.PDClientManager.Get(pdm.PrimaryKey(rtx.Cluster.Namespace, rtx.Cluster.Name))
	if !ok {
		t.Logger.Info("pd client is not registered")
		return false // wait for next sync
	}

	cluster, err := pdClient.Underlay().GetCluster(ctx)
	if err != nil {
		t.Logger.Error(err, "failed to get cluster info")
		return false
	}

	if cluster.Id == 0 {
		return false
	}

	rtx.Cluster.Status.ID = strconv.FormatUint(cluster.Id, 10)
	return true
}
