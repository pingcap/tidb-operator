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

package tasks

import (
	"cmp"
	"context"
	"slices"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type ReconcileContext struct {
	State

	PDClient pdapi.PDClient
	Members  []Member

	// mark pdgroup is bootstrapped if cache of pd is synced
	IsBootstrapped bool

	// Status fields
	v1alpha1.CommonStatus
}

// TODO: move to pdapi
type Member struct {
	ID   string
	Name string
}

func TaskContextPDClient(state *ReconcileContext, m pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("ContextPDClient", func(_ context.Context) task.Result {
		if len(state.PDSlice()) > 0 {
			// TODO: register pd client after it is ready
			if err := m.Register(state.PDGroup()); err != nil {
				return task.Fail().With("cannot register pd client: %v", err)
			}
		}
		ck := state.Cluster()
		pc, ok := m.Get(pdm.PrimaryKey(ck.Namespace, ck.Name))
		if !ok {
			return task.Complete().With("context without pd client is completed, pd cannot be visited")
		}
		state.PDClient = pc.Underlay()

		if !pc.HasSynced() {
			return task.Complete().With("context without pd client is completed, cache of pd info is not synced")
		}

		state.IsBootstrapped = true

		ms, err := pc.Members().List(labels.Everything())
		if err != nil {
			return task.Fail().With("cannot list members: %w", err)
		}

		for _, m := range ms {
			state.Members = append(state.Members, Member{
				Name: m.Name,
				ID:   m.ID,
			})
		}
		slices.SortFunc(state.Members, func(a, b Member) int {
			return cmp.Compare(a.Name, b.Name)
		})

		return task.Complete().With("context is fully completed")
	})
}
