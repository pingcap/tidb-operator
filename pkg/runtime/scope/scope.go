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

// Package scope is defined to minimize specified generic types when call funcs
// Normally only one type(e.g. scope.TiKV/scope.TiKVGroup) need be specified when call any generic funcs
package scope

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/pkg/runtime"
)

// Object is a conversion between runtime object and api object
// runtime.Object <-> api.Object
type Object[F client.Object, T runtime.Object] interface {
	Scheme
	From(F) T
	To(T) F
}

// Instance is a conversion between runtime instance and api instance
// runtime.Instance <-> api.Instance
type Instance[F client.Object, T runtime.Instance] interface {
	Scheme
	From(F) T
	To(T) F
}

// Group is a conversion between runtime group and api group
// runtime.Group <-> api.Group
type Group[F client.Object, T runtime.Group] interface {
	Scheme
	From(F) T
	To(T) F
}

// List defines an interface to refer api.ObjectList and api.Object type
// See apicall.ListInstances for how to use it
type List[
	L client.ObjectList,
	I client.Object,
] interface {
	NewList() L
	GetItems(L) []I
}

// GroupInstance is defined to refer instance scope(IS) for
// - Instance: conversion from api.Instance to runtime.Instance
// - List: new api.InstanceList and get []api.Instance
// - InstanceList: both of the above
type GroupInstance[
	GF client.Object,
	GT runtime.Group,
	IS any,
] interface {
	Group[GF, GT]
	Instance() IS
}

type GroupList[
	F client.Object,
	T runtime.Group,
	L client.ObjectList,
] interface {
	Group[F, T]
	List[L, F]
}

type InstanceList[
	F client.Object,
	T runtime.Instance,
	L client.ObjectList,
] interface {
	Instance[F, T]
	List[L, F]
}

type ClientObject[T any] interface {
	client.Object
	*T
}

type Scheme interface {
	Component() string
	GVK() schema.GroupVersionKind
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

func GVK[S Scheme]() schema.GroupVersionKind {
	var s S
	return s.GVK()
}

func NewList[
	S List[OL, O],
	OL client.ObjectList,
	O client.Object,
]() OL {
	var s S
	return s.NewList()
}

func GetItems[
	S List[OL, O],
	OL client.ObjectList,
	O client.Object,
](l OL) []O {
	var s S
	return s.GetItems(l)
}
