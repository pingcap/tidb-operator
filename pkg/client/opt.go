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

package client

type ApplyOptions struct {
	// Immutable defines fields which is immutable
	// It's only for some fields which cannot be changed but actually maybe changed.
	// For example,
	// Storage class ref can be changed in instance CR to modify volumes
	// if feature VolumeAttributesClass is not enabled(just like v1).
	// However, it's immutable in PVC. When VolumeAttributesClass is enabled,
	// storage class should not be applied to PVC.
	// NOTE; now slice/array is not supported
	Immutable [][]string
}

type ApplyOption interface {
	With(opts *ApplyOptions)
}

type ApplyOptionFunc func(opts *ApplyOptions)

func (f ApplyOptionFunc) With(opts *ApplyOptions) {
	f(opts)
}

func Immutable(fields ...string) ApplyOption {
	return ApplyOptionFunc(func(opts *ApplyOptions) {
		if len(fields) == 0 {
			return
		}
		opts.Immutable = append(opts.Immutable, fields)
	})
}
