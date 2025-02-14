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

type TiFlash struct{}

func (TiFlash) From(f *v1alpha1.TiFlash) *runtime.TiFlash {
	return runtime.FromTiFlash(f)
}

func (TiFlash) To(t *runtime.TiFlash) *v1alpha1.TiFlash {
	return runtime.ToTiFlash(t)
}

func (TiFlash) Component() string {
	return v1alpha1.LabelValComponentTiFlash
}

func (TiFlash) NewList() client.ObjectList {
	return &v1alpha1.TiFlashList{}
}

type TiFlashGroup struct{}

func (TiFlashGroup) From(f *v1alpha1.TiFlashGroup) *runtime.TiFlashGroup {
	return runtime.FromTiFlashGroup(f)
}

func (TiFlashGroup) To(t *runtime.TiFlashGroup) *v1alpha1.TiFlashGroup {
	return runtime.ToTiFlashGroup(t)
}

func (TiFlashGroup) Component() string {
	return v1alpha1.LabelValComponentTiFlash
}

func (TiFlashGroup) NewList() client.ObjectList {
	return &v1alpha1.TiFlashGroupList{}
}
