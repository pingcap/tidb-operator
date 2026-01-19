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

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/v2/pkg/scheme"
)

type Options struct {
	client.Options

	GVs []schema.GroupVersion
}

type Option interface {
	With(opts *Options)
}

type OptionFunc func(opts *Options)

func (f OptionFunc) With(opts *Options) {
	f(opts)
}

func DefaultOptions() *Options {
	o := Options{}
	o.Scheme = scheme.Scheme
	o.GVs = scheme.GroupVersions()

	return &o
}

func GroupVersions(gvs ...schema.GroupVersion) Option {
	return OptionFunc(func(opts *Options) {
		opts.GVs = gvs
	})
}

func Scheme(s *runtime.Scheme) Option {
	return OptionFunc(func(opts *Options) {
		opts.Scheme = s
	})
}

func FromRawOptions(opt *client.Options) Option {
	return OptionFunc(func(opts *Options) {
		opts.Options = *opt
		if opts.Scheme == nil {
			opts.Scheme = scheme.Scheme
		}
	})
}

// AnnoKeyIgnoreDiff defines the fields list to ignore difference which may be produced by mutating webhook.
// Fields are separated by comma. Example: "spec.xxx,spec.yyy"
// It may be deprecated after mutating admission policy is enabled by default. We can easily create a mutating admission policy to
// fix mutating webhooks which don't handle the "server-side apply" properly.
// See https://kubernetes.io/docs/reference/access-authn-authz/mutating-admission-policy/
const AnnoKeyIgnoreDiff = "pingcap.com/ignore-diff"

type ApplyOptions struct {
	// Immutable defines fields which is immutable
	// It's only for some fields which cannot be changed but actually maybe changed.
	// For example,
	// - Storage class ref can be changed in instance CR to modify volumes
	//   if feature VolumeAttributesClass is not enabled(just like v1).
	//   However, it's immutable in PVC. When VolumeAttributesClass is enabled,
	//   storage class should not be applied to PVC.
	// - The immutable field is changed when creation by the webhook.
	// NOTE: now slice/array is not supported
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

func NewApplyOptions(obj client.Object) *ApplyOptions {
	o := &ApplyOptions{}
	anno := obj.GetAnnotations()
	val, ok := anno[AnnoKeyIgnoreDiff]
	if !ok {
		return o
	}

	for f := range strings.SplitSeq(val, ",") {
		fields := strings.Split(f, ".")
		if len(fields) == 0 {
			continue
		}
		o.Immutable = append(o.Immutable, fields)
	}

	return o
}
