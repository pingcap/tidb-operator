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

	"k8s.io/client-go/tools/record"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskBackupCleaner(state *ReconcileContext, c client.Client, recorder record.EventRecorder) task.Task {
	return task.NameTaskFunc("BackupCleaner", func(ctx context.Context) task.Result {
		// because a finalizer is installed on the backup on creation, when backup is deleted,
		// backup.DeletionTimestamp will be set, controller will be informed with an onUpdate event,
		// this is the moment that we can do clean up work.
		if err := state.BackupCleaner.Clean(ctx, state.Backup()); err != nil {
			return task.Fail().With("clean backup failed: %s", err)
		}

		return task.Complete().With("backup cleaned")
	})
}
