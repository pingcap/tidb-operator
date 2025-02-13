package tasks

import (
	"context"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/manager/backup"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"k8s.io/client-go/tools/record"
)

func TaskBackupManager(state *ReconcileContext, c client.Client, recorder record.EventRecorder) task.Task {
	return task.NameTaskFunc("BackupManager", func(ctx context.Context) task.Result {
		mgr := backup.NewBackupManager(c, state.PDClientManager, recorder)
		err := mgr.Sync(state.Backup())
		if err != nil {
			return task.Fail().With("sync backup manager failed: %s", err)
		}

		return task.Complete().With("backup manager synced")
	})
}
