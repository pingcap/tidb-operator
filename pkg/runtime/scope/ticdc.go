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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type TiCDC struct{}

func (TiCDC) From(f *v1alpha1.TiCDC) *runtime.TiCDC {
	return runtime.FromTiCDC(f)
}

func (TiCDC) To(t *runtime.TiCDC) *v1alpha1.TiCDC {
	return runtime.ToTiCDC(t)
}

func (TiCDC) Component() string {
	return v1alpha1.LabelValComponentTiCDC
}

func (TiCDC) NewList() client.ObjectList {
	return &v1alpha1.TiCDCList{}
}

type TiCDCGroup struct{}

func (TiCDCGroup) From(f *v1alpha1.TiCDCGroup) *runtime.TiCDCGroup {
	return runtime.FromTiCDCGroup(f)
}

func (TiCDCGroup) To(t *runtime.TiCDCGroup) *v1alpha1.TiCDCGroup {
	return runtime.ToTiCDCGroup(t)
}

func (TiCDCGroup) Component() string {
	return v1alpha1.LabelValComponentTiCDC
}

func (TiCDCGroup) NewList() client.ObjectList {
	return &v1alpha1.TiCDCGroupList{}
}

func (TiCDCGroup) NewInstanceList() client.ObjectList {
	return &v1alpha1.TiCDCList{}
}
