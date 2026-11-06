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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/util"
	task "github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"

	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/backup"
)

type Config struct {
	BackupManagerImage string
}
type ReconcileContext struct {
	State

	Config        Config
	BackupManager backup.BackupManager
	BackupCleaner backup.BackupCleaner
}

// TaskContextBackupCluster resolves the authorized target before pause/suspend checks.
func TaskContextBackupCluster(state *ReconcileContext, cli client.Client) task.Task {
	return task.NameTaskFunc("ContextBackupCluster", func(ctx context.Context) task.Result {
		cluster, err := util.GetBackupCluster(ctx, cli, state.Backup().Namespace, state.Backup().Spec.BR)
		if err != nil {
			updateErr := state.BackupManager.UpdateStatus(ctx, state.Backup(), &br.BackupCondition{
				Command: br.ParseLogBackupSubcommand(state.Backup()),
				Condition: metav1.Condition{
					Type: string(br.BackupRetryTheFailed), Status: metav1.ConditionTrue,
					Reason: "ClusterAccessFailed", Message: err.Error(),
				},
			}, nil)
			return task.Fail().With("cannot access backup cluster: %v (status update: %v)", err, updateErr)
		}
		state.SetCluster(cluster)
		return task.Complete().With("authorized backup cluster is set")
	})
}
