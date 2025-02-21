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

type TiKV struct{}

func (TiKV) From(f *v1alpha1.TiKV) *runtime.TiKV {
	return runtime.FromTiKV(f)
}

func (TiKV) To(t *runtime.TiKV) *v1alpha1.TiKV {
	return runtime.ToTiKV(t)
}

func (TiKV) Component() string {
	return v1alpha1.LabelValComponentTiKV
}

func (TiKV) NewList() client.ObjectList {
	return &v1alpha1.TiKVList{}
}

type TiKVGroup struct{}

func (TiKVGroup) From(f *v1alpha1.TiKVGroup) *runtime.TiKVGroup {
	return runtime.FromTiKVGroup(f)
}

func (TiKVGroup) To(t *runtime.TiKVGroup) *v1alpha1.TiKVGroup {
	return runtime.ToTiKVGroup(t)
}

func (TiKVGroup) Component() string {
	return v1alpha1.LabelValComponentTiKV
}

func (TiKVGroup) NewList() client.ObjectList {
	return &v1alpha1.TiKVGroupList{}
}

func (TiKVGroup) NewInstanceList() client.ObjectList {
	return &v1alpha1.TiKVList{}
}
