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
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type Object[F client.Object, T runtime.Object] interface {
	Scheme
	From(F) T
	To(T) F
}

// runtime.Instance <-> api.Instance
type Instance[F client.Object, T runtime.Instance] interface {
	Scheme
	From(F) T
	To(T) F
}

// runtime.Group <-> api.Group
type Group[F client.Object, T runtime.Group] interface {
	Scheme
	From(F) T
	To(T) F
}

// []runtime.Instance <-> []api.Instance
type InstanceSlice[F client.Object, T runtime.Instance] interface {
	From([]F) []T
	To([]T) []F
}

type Scheme interface {
	Component() string
	NewList() client.ObjectList
}

// runtime.Group --> api.InstanceList
// type GroupList[F any, T any] interface{}

func From[
	S Object[F, T],
	F client.Object,
	T runtime.Object,
](f F) T {
	var s S
	return s.From(f)
}

func Component[S Scheme]() string {
	var s S
	return s.Component()
}

func NewList[S Scheme]() client.ObjectList {
	var s S
	return s.NewList()
}
