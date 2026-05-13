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

package framework

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

func (f *Framework) MustCreateDM(ctx context.Context, ps ...data.GroupPatch[*v1alpha1.DMGroup]) *v1alpha1.DMGroup {
	dmg := data.NewDMGroup(f.Namespace.Name, ps...)
	ginkgo.By("Creating a dm group")
	f.Must(f.Client.Create(ctx, dmg))

	return dmg
}

func (f *Framework) WaitForDMGroupReady(ctx context.Context, dmg *v1alpha1.DMGroup) {
	ginkgo.By("wait for dm group ready")
	f.Must(waiter.WaitForObjectCondition[scope.DMGroup](
		ctx,
		f.Client,
		dmg,
		v1alpha1.CondReady,
		metav1.ConditionTrue,
		waiter.LongTaskTimeout,
	))
	f.Must(waiter.WaitForDMGroupsHealthy(ctx, f.Client, dmg, waiter.LongTaskTimeout))
	f.Must(waiter.WaitForPodsReady(ctx, f.Client, runtime.FromDMGroup(dmg), waiter.LongTaskTimeout))
}
