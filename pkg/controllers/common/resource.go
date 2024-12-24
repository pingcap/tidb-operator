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

package common

type Setter[T any] interface {
	Set(T)
}

type NamespaceOption interface {
	Namespace() string
}

type NameOption interface {
	Name() string
}

type LabelsOption interface {
	Labels() map[string]string
}

type GetOptions interface {
	NamespaceOption
	NameOption
}

type ListOptions interface {
	NamespaceOption
	LabelsOption
}

type NameFunc func() string

func (f NameFunc) Namespace() string {
	return f()
}

func (f NameFunc) Name() string {
	return f()
}

type Namespace string

func (n Namespace) Namespace() string {
	return string(n)
}

type Name string

func (n Name) Name() string {
	return string(n)
}

type Labels map[string]string

func (l Labels) Labels() map[string]string {
	return l
}

type LabelsFunc func() map[string]string

func (f LabelsFunc) Labels() map[string]string {
	return f()
}

type SetFunc[T any] func(T)

func (f SetFunc[T]) Set(obj T) {
	f(obj)
}

type ResourceInitializer[T any] interface {
	GetOptions
	Setter[T]
}

type Resource[T any] interface {
	WithNamespace(NamespaceOption) Resource[T]
	WithName(NameOption) Resource[T]
	Initializer() ResourceInitializer[T]
}

func NewResource[T any](setter SetFunc[T]) Resource[T] {
	return &resource[T]{
		setter: setter,
	}
}

type resource[T any] struct {
	setter Setter[T]
	ns     NamespaceOption
	name   NameOption
}

func (r *resource[T]) Set(obj T) {
	r.setter.Set(obj)
}

func (r *resource[T]) WithNamespace(ns NamespaceOption) Resource[T] {
	r.ns = ns
	return r
}

func (r *resource[T]) WithName(name NameOption) Resource[T] {
	r.name = name
	return r
}

func (r *resource[T]) Namespace() string {
	return r.ns.Namespace()
}

func (r *resource[T]) Name() string {
	return r.name.Name()
}

func (r *resource[T]) Initializer() ResourceInitializer[T] {
	return r
}

type ResourceSliceInitializer[T any] interface {
	ListOptions
	Setter[[]T]
}

type ResourceSlice[T any] interface {
	WithNamespace(NamespaceOption) ResourceSlice[T]
	WithLabels(LabelsOption) ResourceSlice[T]
	Initializer() ResourceSliceInitializer[T]
}

func NewResourceSlice[T any](setter SetFunc[[]T]) ResourceSlice[T] {
	return &resourceSlice[T]{
		setter: setter,
	}
}

type resourceSlice[T any] struct {
	ns     NamespaceOption
	labels LabelsOption
	setter Setter[[]T]
}

func (r *resourceSlice[T]) Namespace() string {
	return r.ns.Namespace()
}

func (r *resourceSlice[T]) Labels() map[string]string {
	return r.labels.Labels()
}

func (r *resourceSlice[T]) Set(objs []T) {
	r.setter.Set(objs)
}

func (r *resourceSlice[T]) WithNamespace(ns NamespaceOption) ResourceSlice[T] {
	r.ns = ns
	return r
}

func (r *resourceSlice[T]) WithLabels(labels LabelsOption) ResourceSlice[T] {
	r.labels = labels
	return r
}

func (r *resourceSlice[T]) Initializer() ResourceSliceInitializer[T] {
	return r
}
