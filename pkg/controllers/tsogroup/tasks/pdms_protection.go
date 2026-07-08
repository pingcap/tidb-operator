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
	"context"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type PDMSProtectionState interface {
	State
	SetPDMSProtected(bool)
	PDMSProtected() bool
}

func TaskPDMSProtection(state *ReconcileContext) task.Task {
	return task.NameTaskFunc("PDMSProtection", func(ctx context.Context) task.Result {
		tg := state.Object()
		replicas := coreutil.Replicas[scope.TSOGroup](tg)
		involvesMS := coreutil.PDTopologyInvolvesMS(state.PDMSProtectionPDGroups(), state.PDMSProtectionPDs())
		deleting := !tg.DeletionTimestamp.IsZero()
		protected := involvesMS && ((!deleting && replicas == 0) ||
			(deleting && !coreutil.HasAvailableOtherTSOGroup(state.PDMSProtectionTSOGroups(), tg)))
		state.SetPDMSProtected(protected)

		var condChanged bool
		if protected {
			condChanged = coreutil.SetStatusCondition[scope.TSOGroup](
				tg,
				*coreutil.PDMSProtected("TSOGroup is protected because PD desired, actual, transition, or instances involve microservice mode"),
			)
		} else {
			condChanged = coreutil.SetStatusCondition[scope.TSOGroup](
				tg,
				*coreutil.PDMSProtectionNotNeeded(),
			)
		}
		if condChanged {
			state.SetStatusChanged()
		}

		return task.Complete().With("PDMS protection is synced")
	})
}

func CondPDMSProtected(state PDMSProtectionState) task.Condition {
	return task.CondFunc(func() bool { return state.PDMSProtected() })
}

func TaskPDMSProtectedRetry() task.Task {
	return task.NameTaskFunc("PDMSProtectedRetry", func(ctx context.Context) task.Result {
		return task.Retry(defaultUpdateWaitTime).With("TSOGroup deletion is protected by PDMS mode")
	})
}

func effectiveReplicas(tg *v1alpha1.TSOGroup, protected bool) int32 {
	replicas := coreutil.Replicas[scope.TSOGroup](tg)
	if protected && replicas == 0 {
		return 1
	}
	return replicas
}
