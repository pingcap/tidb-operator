// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package alias

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type TikvList struct {
	PodList    *v1.PodList
	TikvStatus *v1alpha1.TiKVStatus
}

// implement type Object interface
func (t TikvList) GetObjectKind() schema.ObjectKind {
	return t.PodList.GetObjectKind()
}

func (t TikvList) DeepCopyObject() runtime.Object {
	out := TikvList{
		PodList:    nil,
		TikvStatus: nil,
	}
	if t.PodList != nil {
		out.PodList = t.PodList.DeepCopy()
	}
	if t.TikvStatus != nil {
		out.TikvStatus = t.TikvStatus.DeepCopy()
	}
	return out
}
