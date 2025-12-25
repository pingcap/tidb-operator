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
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
)

func ClusterPatches(o *Options, ps ...data.ClusterPatch) []data.ClusterPatch {
	if o.TLS {
		ps = append(ps, data.WithClusterTLSEnabled())
	}
	if o.FQDN {
		ps = append(ps, data.WithFQDN())
	}
	ps = append(ps, data.WithFeatureGates(o.Features...))
	return ps
}

func GroupPatches[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](o *Options, ps ...data.GroupPatch[F]) []data.GroupPatch[F] {
	if o.TLS {
		ps = append(ps, data.WithClusterTLSSuffix[S](
			o.ClusterCASuffix,
			o.ClusterCertKeyPairSuffix,
		))
	}
	return ps
}

func TiDBPatches(o *Options, ps ...data.GroupPatch[*v1alpha1.TiDBGroup]) []data.GroupPatch[*v1alpha1.TiDBGroup] {
	if o.TLS {
		ps = append(ps,
			data.WithTiDBMySQLTLS(o.TiDBMySQLTLS()),
		)
	}
	if o.NextGen {
		ps = append(ps,
			data.WithTiDBNextGen(),
			data.WithKeyspace("SYSTEM"),
		)
	}
	return GroupPatches[scope.TiDBGroup](o, ps...)
}

func PDPatches(o *Options, ps ...data.GroupPatch[*v1alpha1.PDGroup]) []data.GroupPatch[*v1alpha1.PDGroup] {
	if o.NextGen {
		ps = append(ps,
			data.WithMSMode(),
			data.WithPDNextGen(),
		)
	}
	return GroupPatches[scope.PDGroup](o, ps...)
}

func TiKVPatches(o *Options, ps ...data.GroupPatch[*v1alpha1.TiKVGroup]) []data.GroupPatch[*v1alpha1.TiKVGroup] {
	if o.NextGen {
		ps = append(ps,
			data.WithTiKVNextGen(),
		)
	}
	if o.EnableTiKVWorkers {
		ps = append(ps,
			data.WithTiKVWorkers(),
		)
	}
	return GroupPatches[scope.TiKVGroup](o, ps...)
}

func TiProxyPatches(o *Options, ps ...data.GroupPatch[*v1alpha1.TiProxyGroup]) []data.GroupPatch[*v1alpha1.TiProxyGroup] {
	if o.NextGen {
		ps = append(ps,
			data.WithTiProxyNextGen(),
		)
	}
	return GroupPatches[scope.TiProxyGroup](o, ps...)
}

func TiCDCPatches(o *Options, ps ...data.GroupPatch[*v1alpha1.TiCDCGroup]) []data.GroupPatch[*v1alpha1.TiCDCGroup] {
	return GroupPatches[scope.TiCDCGroup](o, ps...)
}

func TiKVWorkerPatches(o *Options, ps ...data.GroupPatch[*v1alpha1.TiKVWorkerGroup]) []data.GroupPatch[*v1alpha1.TiKVWorkerGroup] {
	if o.NextGen {
		ps = append(ps,
			data.WithTiKVWorkerNextGen(),
		)
	}
	return GroupPatches[scope.TiKVWorkerGroup](o, ps...)
}
