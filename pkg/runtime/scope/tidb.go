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
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type TiDB struct{}

func (TiDB) From(f *v1alpha1.TiDB) *runtime.TiDB {
	return runtime.FromTiDB(f)
}

func (TiDB) To(t *runtime.TiDB) *v1alpha1.TiDB {
	return runtime.ToTiDB(t)
}

func (TiDB) Component() string {
	return v1alpha1.LabelValComponentTiDB
}

func (TiDB) NewList() client.ObjectList {
	return &v1alpha1.TiDBList{}
}

type TiDBGroup struct{}

func (TiDBGroup) From(f *v1alpha1.TiDBGroup) *runtime.TiDBGroup {
	return runtime.FromTiDBGroup(f)
}

func (TiDBGroup) To(t *runtime.TiDBGroup) *v1alpha1.TiDBGroup {
	return runtime.ToTiDBGroup(t)
}

func (TiDBGroup) Component() string {
	return v1alpha1.LabelValComponentTiDB
}

func (TiDBGroup) NewList() client.ObjectList {
	return &v1alpha1.TiDBGroupList{}
}

func (TiDBGroup) NewInstanceList() client.ObjectList {
	return &v1alpha1.TiDBList{}
}
