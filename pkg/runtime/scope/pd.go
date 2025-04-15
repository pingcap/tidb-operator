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

type PD struct{}

func (PD) From(f *v1alpha1.PD) *runtime.PD {
	return runtime.FromPD(f)
}

func (PD) To(t *runtime.PD) *v1alpha1.PD {
	return runtime.ToPD(t)
}

func (PD) Component() string {
	return v1alpha1.LabelValComponentPD
}

func (PD) NewList() client.ObjectList {
	return &v1alpha1.PDList{}
}

type PDGroup struct{}

func (PDGroup) From(f *v1alpha1.PDGroup) *runtime.PDGroup {
	return runtime.FromPDGroup(f)
}

func (PDGroup) To(t *runtime.PDGroup) *v1alpha1.PDGroup {
	return runtime.ToPDGroup(t)
}

func (PDGroup) Component() string {
	return v1alpha1.LabelValComponentPD
}

func (PDGroup) NewList() client.ObjectList {
	return &v1alpha1.PDGroupList{}
}

func (PDGroup) NewInstanceList() client.ObjectList {
	return &v1alpha1.PDList{}
}
