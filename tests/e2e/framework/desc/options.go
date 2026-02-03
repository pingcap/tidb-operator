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

package desc

import (
	"strings"

	"github.com/onsi/ginkgo/v2"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

type Options struct {
	Features []metav1alpha1.Feature

	Namespace string

	Focus bool
}

type Option interface {
	With(opts *Options)
}

func DefaultOptions() *Options {
	return &Options{}
}

func (o *Options) Labels() ginkgo.Labels {
	var ls []string
	for _, f := range o.Features {
		ls = append(ls, "f:"+string(f))
	}

	return ginkgo.Label(ls...)
}

func (o *Options) String() string {
	return strings.Join(o.Labels(), ",")
}

type WithOption func(opts *Options)

func (opt WithOption) With(opts *Options) {
	opt(opts)
}

// Focus on this case, like ginkgo.Focus
func Focus() Option {
	return WithOption(func(opts *Options) {
		opts.Focus = true
	})
}

func Features(fs ...metav1alpha1.Feature) Option {
	return WithOption(func(opts *Options) {
		opts.Features = append(opts.Features, fs...)
	})
}
