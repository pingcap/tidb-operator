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
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	tsom "github.com/pingcap/tidb-operator/v2/pkg/timanager/tso"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type ModeSwitchState interface {
	State
	SetModeSwitchBlocked(bool)
	ModeSwitchBlocked() bool
}

type tsoClientGetter interface {
	Get(key string) (tsom.TSOClient, bool)
}

func TaskModeSwitch(state *ReconcileContext, tsocm tsoClientGetter) task.Task {
	return task.NameTaskFunc("ModeSwitch", func(ctx context.Context) task.Result {
		state.SetModeSwitchBlocked(false)

		pdg := state.PDGroup()
		target := pdg.Spec.Template.Spec.Mode
		updateRevision, _, _ := state.Revision()

		if pdInstancesAtTarget(state.PDSlice(), target, updateRevision, state.Group().Replicas()) {
			changed := setModeSwitchComplete(pdg, target)
			if changed {
				state.SetStatusChanged()
				return task.Complete().With("PD mode switch is complete")
			}
			return task.Complete().With("PD mode switch is not needed")
		}

		if !modeSwitchActive(pdg, state.PDSlice(), target) {
			changed := coreutil.SetStatusCondition[scope.PDGroup](pdg, *coreutil.ModeSwitchNotNeeded())
			if changed {
				state.SetStatusChanged()
			}
			return task.Complete().With("PD mode switch is not needed")
		}

		if blocked, reason, msg := unsupportedTopology(state.PDGroups()); blocked {
			blockModeSwitch(state, reason, msg)
			return task.Retry(defaultUpdateWaitTime).With(msg)
		}

		if target == v1alpha1.PDModeMS {
			if blocked, reason, msg := checkTSODependency(ctx, state.TSOGroups(), tsocm); blocked {
				blockModeSwitch(state, reason, msg)
				return task.Retry(defaultUpdateWaitTime).With(msg)
			}
		}

		msg := fmt.Sprintf("switching PD instances to mode %q", target)
		changed := coreutil.SetStatusCondition[scope.PDGroup](pdg, *coreutil.ModeSwitching(v1alpha1.ReasonSwitchingPDInstances, msg))
		if changed {
			state.SetStatusChanged()
		}
		return task.Complete().With(msg)
	})
}

func CondModeSwitchBlocked(state ModeSwitchState) task.Condition {
	return task.CondFunc(func() bool { return state.ModeSwitchBlocked() })
}

func modeSwitchActive(pdg *v1alpha1.PDGroup, pds []*v1alpha1.PD, target v1alpha1.PDMode) bool {
	if coreutil.PDGroupModeSwitching(pdg) {
		return true
	}
	if len(pds) == 0 && pdg.Status.Mode == v1alpha1.PDModeNormal {
		return false
	}
	if pdg.Status.Mode != target {
		return true
	}
	for _, pd := range pds {
		if pd.Spec.Mode != target {
			return true
		}
	}
	return false
}

func pdInstancesAtTarget(pds []*v1alpha1.PD, target v1alpha1.PDMode, updateRevision string, replicas int32) bool {
	if replicas < 0 || len(pds) != int(replicas) {
		return false
	}
	for _, pd := range pds {
		if pd.Spec.Mode != target {
			return false
		}
		if updateRevision != "" && pd.Status.CurrentRevision != updateRevision {
			return false
		}
		if !meta.IsStatusConditionTrue(pd.Status.Conditions, v1alpha1.CondReady) {
			return false
		}
		if !meta.IsStatusConditionTrue(pd.Status.Conditions, v1alpha1.CondSynced) {
			return false
		}
	}
	return true
}

func setModeSwitchComplete(pdg *v1alpha1.PDGroup, mode v1alpha1.PDMode) bool {
	changed := false
	if pdg.Status.Mode != mode {
		pdg.Status.Mode = mode
		changed = true
	}
	return coreutil.SetStatusCondition[scope.PDGroup](pdg, *coreutil.ModeSwitchComplete()) || changed
}

func blockModeSwitch(state *ReconcileContext, reason, msg string) {
	pdg := state.PDGroup()
	state.SetModeSwitchBlocked(true)
	changed := coreutil.SetStatusCondition[scope.PDGroup](pdg, *coreutil.ModeSwitching(reason, msg))
	if changed {
		state.SetStatusChanged()
	}
}

func unsupportedTopology(pdgs []*v1alpha1.PDGroup) (blocked bool, reason, message string) {
	if len(pdgs) != 1 {
		return true, v1alpha1.ReasonWaitingForSinglePDGroup, fmt.Sprintf("waiting for exactly one PDGroup, got %d", len(pdgs))
	}
	return false, "", ""
}

func checkTSODependency(ctx context.Context, tgs []*v1alpha1.TSOGroup, tsocm tsoClientGetter) (blocked bool, reason, message string) {
	if len(tgs) == 0 {
		return true, v1alpha1.ReasonWaitingForTSOGroup, "waiting for at least one TSOGroup, got 0"
	}

	for _, tg := range tgs {
		if tg.Spec.Replicas == nil || *tg.Spec.Replicas == 0 || !coreutil.IsGroupHealthyAndUpToDate[scope.TSOGroup](tg) {
			return true, v1alpha1.ReasonWaitingForTSOGroupReady, fmt.Sprintf("waiting for TSOGroup %s to be ready and up-to-date", tg.Name)
		}
	}

	tg := tgs[0]
	tsoc, ok := tsocm.Get(timanager.PrimaryKey(tg.Namespace, tg.Spec.Cluster.Name))
	if !ok {
		return true, v1alpha1.ReasonWaitingForTSOGroupReady, fmt.Sprintf("waiting for TSOGroup %s health check client", tg.Name)
	}
	healthy, err := tsoc.Underlay().IsHealthy(ctx)
	if err != nil {
		return true, v1alpha1.ReasonWaitingForTSOGroupReady,
			fmt.Sprintf("waiting for TSOGroup %s health API to be ready: %v", tg.Name, err)
	}
	if !healthy {
		return true, v1alpha1.ReasonWaitingForTSOGroupReady, fmt.Sprintf("waiting for TSOGroup %s health API to be ready", tg.Name)
	}

	// The TSO health API is served by TSO itself and is safe to use before PD
	// rolls to MS. It only proves that the TSO service is accepting requests.

	// Do not wait for TSO members or the TSO primary here. They are exposed by
	// PD's MS APIs, which may be unavailable while PD is still running in normal
	// mode. Waiting for them before rolling PD to MS would deadlock the mode
	// switch: PD must enter MS mode before the MS member registry is reliable.
	return false, "", ""
}
