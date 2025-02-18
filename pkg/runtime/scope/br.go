package scope

import (
	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type Backup struct{}

func (Backup) From(f *v1alpha1.Backup) *runtime.Backup {
	return runtime.FromBackup(f)
}

func (Backup) To(t *runtime.Backup) *v1alpha1.Backup {
	return runtime.ToBackup(t)
}

func (Backup) Component() string {
	return v1alpha1.LabelValComponentBackup
}

func (Backup) NewList() client.ObjectList {
	return &v1alpha1.BackupList{}
}

type Restore struct{}

func (Restore) From(f *v1alpha1.Restore) *runtime.Restore {
	return runtime.FromRestore(f)
}

func (Restore) To(t *runtime.Restore) *v1alpha1.Restore {
	return runtime.ToRestore(t)
}

func (Restore) Component() string {
	return v1alpha1.LabelValComponentRestore
}

func (Restore) NewList() client.ObjectList {
	return &v1alpha1.RestoreList{}
}
