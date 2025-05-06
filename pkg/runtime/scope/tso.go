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

type TSO struct{}

func (TSO) From(f *v1alpha1.TSO) *runtime.TSO {
	return runtime.FromTSO(f)
}

func (TSO) To(t *runtime.TSO) *v1alpha1.TSO {
	return runtime.ToTSO(t)
}

func (TSO) Component() string {
	return v1alpha1.LabelValComponentTSO
}

func (TSO) NewList() client.ObjectList {
	return &v1alpha1.TSOList{}
}

type TSOGroup struct{}

func (TSOGroup) From(f *v1alpha1.TSOGroup) *runtime.TSOGroup {
	return runtime.FromTSOGroup(f)
}

func (TSOGroup) To(t *runtime.TSOGroup) *v1alpha1.TSOGroup {
	return runtime.ToTSOGroup(t)
}

func (TSOGroup) NewInstanceList() *v1alpha1.TSOList {
	return &v1alpha1.TSOList{}
}

func (TSOGroup) GetInstanceItems(l *v1alpha1.TSOList) []*v1alpha1.TSO {
	items := make([]*v1alpha1.TSO, 0, len(l.Items))
	for i := range l.Items {
		items = append(items, &l.Items[i])
	}
	return items
}

func (TSOGroup) Component() string {
	return v1alpha1.LabelValComponentTSO
}

func (TSOGroup) NewList() client.ObjectList {
	return &v1alpha1.TSOGroupList{}
}
