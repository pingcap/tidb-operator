// Copyright 2019. PingCAP, Inc.
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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Tikv v1.Pod

type TikvList v1.PodList

// implement type Object interface
func (t Tikv) GetObjectKind() schema.ObjectKind {
	return t.GetObjectKind()
}

func (t Tikv) DeepCopyObject() runtime.Object {
	return t.DeepCopyObject()
}

// implement type Object interface
func (t TikvList) GetObjectKind() schema.ObjectKind {
	return t.GetObjectKind()
}

func (t TikvList) DeepCopyObject() runtime.Object {
	return t.DeepCopyObject()
}
