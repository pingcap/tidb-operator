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

// api.group -> []api.Instance
type InstanceSlice[
	GF client.Object,
	GT runtime.Group,
	IL client.ObjectList,
	I client.Object,
] interface {
	Group[GF, GT]
	InstanceListCreator[IL]
	InstanceItemsGetter[IL, I]
}

type InstanceListCreator[
	L client.ObjectList,
] interface {
	NewInstanceList() L
}

type InstanceItemsGetter[
	IL client.ObjectList,
	I client.Object,
] interface {
	GetInstanceItems(IL) []I
}

type Scheme interface {
	Component() string
	NewList() client.ObjectList
}

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

func NewInstanceList[
	S InstanceListCreator[L],
	L client.ObjectList,
]() L {
	var s S
	return s.NewInstanceList()
}

func GetInstanceItems[
	S InstanceItemsGetter[IL, I],
	IL client.ObjectList,
	I client.Object,
](l IL) []I {
	var s S
	return s.GetInstanceItems(l)
}
