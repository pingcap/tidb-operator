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

package action

import (
	"context"

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/desc"
)

func MustCreatePD(
	ctx context.Context,
	f *framework.Framework,
	o *desc.Options,
	ps ...data.GroupPatch[*v1alpha1.PDGroup],
) *v1alpha1.PDGroup {
	pdg := data.NewPDGroup(
		f.Namespace.Name,
		desc.PDPatches(o, ps...)...,
	)
	ginkgo.By("Creating a pd group")
	f.Must(f.Client.Create(ctx, pdg))

	return pdg
}

func MustCreateTiKV(
	ctx context.Context,
	f *framework.Framework,
	o *desc.Options,
	ps ...data.GroupPatch[*v1alpha1.TiKVGroup],
) *v1alpha1.TiKVGroup {
	kvg := data.NewTiKVGroup(
		f.Namespace.Name,
		desc.TiKVPatches(o, ps...)...,
	)
	ginkgo.By("Creating a tikv group")
	f.Must(f.Client.Create(ctx, kvg))

	return kvg
}

func MustCreateTiDB(
	ctx context.Context,
	f *framework.Framework,
	o *desc.Options,
	ps ...data.GroupPatch[*v1alpha1.TiDBGroup],
) *v1alpha1.TiDBGroup {
	dbg := data.NewTiDBGroup(
		f.Namespace.Name,
		desc.TiDBPatches(o, ps...)...,
	)
	ginkgo.By("Creating a tidb group")
	f.Must(f.Client.Create(ctx, dbg))

	return dbg
}

func MustCreateTiCDC(
	ctx context.Context,
	f *framework.Framework,
	o *desc.Options,
	ps ...data.GroupPatch[*v1alpha1.TiCDCGroup],
) *v1alpha1.TiCDCGroup {
	cg := data.NewTiCDCGroup(
		f.Namespace.Name,
		desc.TiCDCPatches(o, ps...)...,
	)
	ginkgo.By("Creating a ticdc group")
	f.Must(f.Client.Create(ctx, cg))

	return cg
}
