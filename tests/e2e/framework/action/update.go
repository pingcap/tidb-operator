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
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
)

func MustUpdateCluster(
	ctx context.Context,
	f *framework.Framework,
	patches ...data.ClusterPatch,
) {
	ginkgo.By(fmt.Sprintf("update cluster %v", client.ObjectKeyFromObject(f.Cluster)))

	mp := client.MergeFrom(f.Cluster.DeepCopy())
	for _, p := range patches {
		p(f.Cluster)
	}
	err := f.Client.Patch(ctx, f.Cluster, mp)
	gomega.ExpectWithOffset(1, err).To(gomega.Succeed())
}
