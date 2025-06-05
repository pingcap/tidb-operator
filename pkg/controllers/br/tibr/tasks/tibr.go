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
	"strings"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	t "github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskEnsureSubResources(rtx *ReconcileContext) t.Task {
	return t.Block(
		t.IfNot(condTiBROwnerReferencesExist(rtx), taskPatchTiBRWithOwnerReferences(rtx), TaskContextRefreshTiBR(rtx)),
		t.IfNot(condConfigMapExist(rtx), taskCreateConfigMap(rtx), TaskContextRefreshConfigMap(rtx)),
		t.IfNot(condStatefulSetExist(rtx), taskCreateStatefulSet(rtx), TaskContextRefreshStatefulSet(rtx)),
		t.IfNot(condHeadlessSvcExist(rtx), taskCreateHeadlessSvc(rtx), TaskContextRefreshHeadlessSvc(rtx)),
		t.If(condPVCDeclared(rtx), taskPatchPVCWithOwnerReferences(rtx)),
		// TODO: not support update at first version, just recreate it as workaround
		// taskUpdateSubResourcesIfNeeded(rtx),
		taskWaitAllSubResourcesSynced(rtx))
}

func TaskEnsureSubResourcesCleanup(rtx *ReconcileContext) t.Task {
	return t.Block(
		t.If(condHeadlessSvcExist(rtx), taskDeleteHeadlessSvc(rtx)),
		t.If(condStatefulSetExist(rtx), taskDeleteStatefulSet(rtx)),
		t.If(condConfigMapExist(rtx), taskDeleteConfigMap(rtx)),
		taskWaitTiBRSuspended(rtx))
}

func CondTiBRNotFound(rtx *ReconcileContext) t.Condition {
	return t.CondFunc(func() bool {
		return rtx.TiBR() == nil
	})
}

func CondTiBRIsDeleting(rtx *ReconcileContext) t.Condition {
	return t.CondFunc(func() bool {
		return !rtx.TiBR().GetDeletionTimestamp().IsZero()
	})
}

func condTiBROwnerReferencesExist(rtx *ReconcileContext) t.Condition {
	return t.CondFunc(func() bool {
		tibr := rtx.TiBR()
		return len(tibr.GetOwnerReferences()) != 0
	})
}

func taskPatchTiBRWithOwnerReferences(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("PatchTiBRWithOwnerReferences", func(ctx context.Context) t.Result {
		it := rtx.TiBR()
		it.SetOwnerReferences([]v1.OwnerReference{
			*v1.NewControllerRef(rtx.Cluster(), v1alpha1.SchemeGroupVersion.WithKind("Cluster")),
		})
		err := rtx.Client().Update(ctx, it)
		if err != nil {
			return t.Fail().With("failed to patch TiBR %s with owner references: %s", it.Name, err.Error())
		}
		return t.Complete().With("owner references patched for TiBR")
	})
}

func taskWaitAllSubResourcesSynced(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("WaitAllSubResourcesSynced", func(ctx context.Context) t.Result {
		// set status for later status update task
		rtx.unSyncedReasons = evaluateSyncedStatus(rtx)
		if len(rtx.unSyncedReasons) != 0 {
			return t.Retry(2 * time.Second).With(strings.Join(rtx.unSyncedReasons, "; "))
		}
		return t.Complete().With("all sub-resources synced")
	})
}

func taskWaitTiBRSuspended(rtx *ReconcileContext) t.Task {
	return t.Block(
		TaskContextRefreshConfigMap(rtx),
		TaskContextRefreshStatefulSet(rtx),
		TaskContextRefreshHeadlessSvc(rtx),
		t.NameTaskFunc("WaitTiBRSuspended", func(ctx context.Context) t.Result {
			if rtx.ConfigMap() != nil {
				return t.Retry(2 * time.Second).With("configmap is not cleanup yet")
			}
			if rtx.StatefulSet() != nil {
				return t.Retry(2 * time.Second).With("statefulset is not cleanup yet")
			}
			if rtx.HeadlessSvc() != nil {
				return t.Retry(2 * time.Second).With("headless service is not cleanup yet")
			}
			return t.Complete().With("all sub-resources cleanup")
		}))
}
