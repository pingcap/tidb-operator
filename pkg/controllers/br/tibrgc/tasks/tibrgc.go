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
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	brv1alpha1 "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	t "github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskEnsureSubResources(rtx *ReconcileContext) t.Task {
	return t.Block(
		TaskContextRefreshTiBRGC(rtx),
		t.IfNot(condTiBRGCOwnerReferencesExist(rtx), taskPatchTiBRGCWithOwnerReferences(rtx)),

		t.If(condT2StrategyDefined(rtx), taskApplyT2CronJob(rtx)),
		t.IfNot(condT2StrategyDefined(rtx), taskDeleteT2CronJob(rtx)),

		t.If(condT3StrategyDefined(rtx), taskApplyT3CronJob(rtx)),
		t.IfNot(condT3StrategyDefined(rtx), taskDeleteT3CronJob(rtx)),

		TaskContextRefreshTiBRGC(rtx),
		taskSetTiBRGCPhaseRunning(rtx),
	)
}

func TaskEnsureSubResourcesCleanup(rtx *ReconcileContext) t.Task {
	return t.Block(
		taskDeleteT2CronJob(rtx),
		taskDeleteT3CronJob(rtx),
		taskWaitTiBRGCSuspended(rtx),
		taskSetTiBRGCPhaseSuspended(rtx),
	)
}

func CondTiBRGCNotFound(rtx *ReconcileContext) t.Condition {
	return t.CondFunc(func() bool {
		return rtx.TiBRGC() == nil
	})
}

func CondTiBRGCIsDeleting(rtx *ReconcileContext) t.Condition {
	return t.CondFunc(func() bool {
		return !rtx.TiBRGC().GetDeletionTimestamp().IsZero()
	})
}

func CondMoreThanOneTiBRGCCreated(rtx *ReconcileContext) t.Condition {
	return t.CondFunc(func() bool {
		brgclist := rtx.TiBRGCList4SameCluster()
		return len(brgclist.Items) > 1
	})
}

func condTiBRGCOwnerReferencesExist(rtx *ReconcileContext) t.Condition {
	return t.CondFunc(func() bool {
		tibrgc := rtx.TiBRGC()
		return len(tibrgc.GetOwnerReferences()) != 0
	})
}

func condT2StrategyDefined(rtx *ReconcileContext) t.Condition {
	return t.CondFunc(func() bool {
		return rtx.TiBRGC() != nil && rtx.TiBRGC().Spec.GetT2Strategy() != nil
	})
}

func condT3StrategyDefined(rtx *ReconcileContext) t.Condition {
	return t.CondFunc(func() bool {
		return rtx.TiBRGC() != nil && rtx.TiBRGC().Spec.GetT3Strategy() != nil
	})
}

func taskSetTiBRGCPhaseRunning(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("SetTiBRGCPhaseRunning", func(ctx context.Context) t.Result {
		it := rtx.TiBRGC()
		if it.Status.Phase == brv1alpha1.TiBRGCPhaseRunning {
			return t.Complete().With("TiBRGC phase is already running")
		}
		it.Status.Phase = brv1alpha1.TiBRGCPhaseRunning
		err := rtx.Client().Status().Update(ctx, it)
		if err != nil {
			return t.Fail().With("failed to set TiBRGC phase to running: %s", err.Error())
		}
		return t.Complete().With("TiBRGC phase set to running")
	})
}

func taskSetTiBRGCPhaseSuspended(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("SetTiBRGCPhaseSuspended", func(ctx context.Context) t.Result {
		it := rtx.TiBRGC()
		if it.Status.Phase == brv1alpha1.TiBRGCPhaseSuspended {
			return t.Complete().With("TiBRGC phase is already suspended")
		}
		it.Status.Phase = brv1alpha1.TiBRGCPhaseSuspended
		err := rtx.Client().Status().Update(ctx, it)
		if err != nil {
			return t.Fail().With("failed to set TiBRGC phase to suspended: %s", err.Error())
		}
		return t.Complete().With("TiBRGC phase set to suspended")
	})
}

func taskPatchTiBRGCWithOwnerReferences(rtx *ReconcileContext) t.Task {
	return t.NameTaskFunc("PatchTiBRGCWithOwnerReferences", func(ctx context.Context) t.Result {
		it := rtx.TiBRGC()
		it.SetOwnerReferences([]v1.OwnerReference{
			*v1.NewControllerRef(rtx.Cluster(), v1alpha1.SchemeGroupVersion.WithKind("Cluster")),
		})
		err := rtx.Client().Update(ctx, it)
		if err != nil {
			return t.Fail().With("failed to patch TiBRGC %s with owner references: %s", it.Name, err.Error())
		}
		return t.Complete().With("owner references patched for TiBRGC")
	})
}

func taskWaitTiBRGCSuspended(rtx *ReconcileContext) t.Task {
	return t.Block(
		TaskContextRefreshT2CronJob(rtx),
		TaskContextRefreshT3CronJob(rtx),
		t.NameTaskFunc("WaitTiBRGCSuspended", func(ctx context.Context) t.Result {
			if rtx.T2Cronjob() != nil {
				return t.Retry(2 * time.Second).With("t2 cronjob is not cleanup yet")
			}
			if rtx.T3Cronjob() != nil {
				return t.Retry(2 * time.Second).With("t3 cronjob is not cleanup yet")
			}
			return t.Complete().With("all sub-resources cleanup")
		}))
}
