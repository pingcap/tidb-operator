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
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	t "github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskUpdateStatusIfNeeded(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("UpdateStatusIfNeeded", func(ctx context.Context) t.Result {
		var conditions = &rtx.TiBR().Status.Conditions
		notReadyReasons := evaluateSyncedStatus(rtx)
		updated := doUpdateConditions(conditions, notReadyReasons, rtx.unSyncedReasons)
		if updated {
			if err := rtx.Client().Status().Update(ctx, rtx.TiBR()); err != nil {
				return t.Fail().With("failed to update status: %s", err.Error())
			} else {
				return t.Complete().With("status updated successfully")
			}
		}

		return t.Complete().With("no status change needed")
	})
}

func doUpdateConditions(conditions *[]metav1.Condition, notReadyReasons, unSyncedReasons []string) bool {
	readyStatusNeedUpdate := false
	syncedStatusNeedUpdate := false

	if len(notReadyReasons) > 0 {
		readyStatusNeedUpdate = meta.SetStatusCondition(conditions, metav1.Condition{
			Type:    v1alpha1br.TiBRCondReady,
			Status:  metav1.ConditionFalse,
			Reason:  v1alpha1br.TiBRReasonNotReady,
			Message: strings.Join(notReadyReasons, "; "),
		})
	} else {
		readyStatusNeedUpdate = meta.SetStatusCondition(conditions, metav1.Condition{
			Type:   v1alpha1br.TiBRCondReady,
			Status: metav1.ConditionTrue,
			Reason: v1alpha1br.TiBRReasonReady,
		})
	}

	if len(unSyncedReasons) > 0 {
		syncedStatusNeedUpdate = meta.SetStatusCondition(conditions, metav1.Condition{
			Type:    v1alpha1br.TiBRCondSynced,
			Status:  metav1.ConditionFalse,
			Reason:  v1alpha1br.TiBRReasonUnSynced,
			Message: strings.Join(unSyncedReasons, "; "),
		})
	} else {
		syncedStatusNeedUpdate = meta.SetStatusCondition(conditions, metav1.Condition{
			Type:   v1alpha1br.TiBRCondSynced,
			Status: metav1.ConditionTrue,
			Reason: v1alpha1br.TiBRReasonSynced,
		})
	}
	return readyStatusNeedUpdate || syncedStatusNeedUpdate
}

// TODO: currently, we haven't support update TiBR yet, so just take the ready status as synced status
func evaluateSyncedStatus(rtx *ReconcileContext) (unSyncedReasons []string) {
	return evaluateReadyStatus(rtx)
}

func evaluateReadyStatus(rtx *ReconcileContext) (notReadyReasons []string) {
	if rtx.ConfigMap() == nil {
		notReadyReasons = append(notReadyReasons, "configmap is not existed")
	}

	if rtx.StatefulSet() == nil {
		notReadyReasons = append(notReadyReasons, "statefulset is not existed")
	} else if rtx.StatefulSet().Status.ReadyReplicas != StatefulSetReplica {
		notReadyReasons = append(notReadyReasons, fmt.Sprintf("ready replica of statefulset is not %d", StatefulSetReplica))
	}

	if rtx.HeadlessSvc() == nil {
		notReadyReasons = append(notReadyReasons, "headless service is not existed")
	}
	return notReadyReasons
}
