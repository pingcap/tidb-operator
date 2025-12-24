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
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/cert"
)

type Options struct {
	Features []metav1alpha1.Feature

	TLS bool

	NextGen           bool
	EnableTiKVWorkers bool

	Namespace string
	// all CA suffix will be added after the ns
	ClusterCASuffix          string
	ClusterCertKeyPairSuffix string
	MySQLClientCASuffix      string
	MySQLCertKeyPairSuffix   string
}

type Option interface {
	With(opts *Options)
}

func DefaultOptions() *Options {
	return &Options{
		ClusterCASuffix:          "cluster-ca",
		ClusterCertKeyPairSuffix: "internal",
		MySQLClientCASuffix:      "mysql-ca",
		MySQLCertKeyPairSuffix:   "mysql-tls",
	}
}

func (o *Options) Labels() ginkgo.Labels {
	var ls []string
	for _, f := range o.Features {
		ls = append(ls, "f:"+string(f))
	}

	if o.TLS {
		ls = append(ls, "f:TLS")
	}

	return ginkgo.Label(ls...)
}

func (o *Options) String() string {
	return strings.Join(o.Labels(), ",")
}

func (o *Options) TiDBMySQLClientTLS() (string, string) {
	return cert.MySQLClient(o.TiDBMySQLTLS())
}

func (o *Options) TiDBMySQLTLS() (string, string) {
	return o.Namespace + "-tidb-" + o.MySQLClientCASuffix, "tidb-" + o.MySQLCertKeyPairSuffix
}

func (o *Options) ClusterCA() string {
	return o.Namespace + "-" + o.ClusterCASuffix
}

type WithOption func(opts *Options)

func (opt WithOption) With(opts *Options) {
	opt(opts)
}

func TLS() Option {
	return WithOption(func(opts *Options) {
		opts.TLS = true
	})
}

func Features(fs ...metav1alpha1.Feature) Option {
	return WithOption(func(opts *Options) {
		opts.Features = append(opts.Features, fs...)
	})
}

func NextGen() Option {
	return WithOption(func(opts *Options) {
		opts.NextGen = true
	})
}

func EnableTiKVWorkers() Option {
	return WithOption(func(opts *Options) {
		opts.EnableTiKVWorkers = true
	})
}
