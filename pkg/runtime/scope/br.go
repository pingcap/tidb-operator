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

package scope

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
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
