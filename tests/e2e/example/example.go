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

package example

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/yaml"
)

var _ = ginkgo.Describe("Example", label.KindExample, label.P0, func() {
	f := framework.New()
	f.Setup(framework.WithSkipClusterCreation())

	ginkgo.DescribeTable("Example",
		func(ctx context.Context, dir string, cn string) {
			// NOTE(liubo02): ignore resources config to avoid resource limit in local
			f.Must(yaml.ApplyDir(ctx, f.Client, dir, yaml.Namespace(f.Namespace.Name), yaml.IgnoreResources{}))
			f.WaitForAllReady(ctx, cn)

			// now delete ns directly may be blocked by tikv prestop hook
			// TODO(liubo02): remove it after prestop hook is fixed
			c := v1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: f.Namespace.Name,
					Name:      cn,
				},
			}
			f.Must(f.Client.Delete(ctx, &c))
			f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, &c, waiter.LongTaskTimeout))
		},
		ginkgo.Entry("basic", "example/data/basic", "basic"),
	)
})
