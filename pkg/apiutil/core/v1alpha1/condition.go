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

package coreutil

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func Suspended() *metav1.Condition {
	return &metav1.Condition{
		Type:    v1alpha1.CondSuspended,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha1.ReasonSuspended,
		Message: "group/instance is suspended",
	}
}

func Suspending() *metav1.Condition {
	return &metav1.Condition{
		Type:    v1alpha1.CondSuspended,
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.ReasonSuspending,
		Message: "group/instance is suspending",
	}
}

func Unsuspended() *metav1.Condition {
	return &metav1.Condition{
		Type:    v1alpha1.CondSuspended,
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.ReasonUnsuspended,
		Message: "group/instance is unsuspended",
	}
}

func Ready() *metav1.Condition {
	return &metav1.Condition{
		Type:    v1alpha1.CondReady,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha1.CondReady,
		Message: "all subreources are ready",
	}
}

func Unready(reason string) *metav1.Condition {
	var msg string
	switch reason {
	case v1alpha1.ReasonNotAllInstancesReady:
		msg = "not all instances are ready"
	case v1alpha1.ReasonPodNotCreated:
		msg = "pod of the instance does not exist"
	case v1alpha1.ReasonPodNotReady:
		msg = "pod of the instance is not ready"
	case v1alpha1.ReasonPodTerminating:
		msg = "pod of the instance is terminating"
	case v1alpha1.ReasonInstanceNotHealthy:
		msg = "instance is not probed as healthy"
	default:
		msg = fmt.Sprintf("unready because of unknown reason: %s", reason)
		reason = v1alpha1.ReasonUnready
	}

	return &metav1.Condition{
		Type:    v1alpha1.CondReady,
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: msg,
	}
}

func Synced() *metav1.Condition {
	return &metav1.Condition{
		Type:    v1alpha1.CondSynced,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha1.ReasonSynced,
		Message: "all subreources are synced",
	}
}

func Unsynced(reason string) *metav1.Condition {
	var msg string
	switch reason {
	case v1alpha1.ReasonNotAllInstancesUpToDate:
		msg = "not all instances are up to date"
	case v1alpha1.ReasonPodNotUpToDate:
		msg = "pod is not up to date"
	case v1alpha1.ReasonPodNotDeleted:
		msg = "cluster is suspending or instance is deleting, pod still exists"
	default:
		msg = fmt.Sprintf("unsynced because of unknown reason: %s", reason)
		reason = v1alpha1.ReasonUnsynced
	}
	return &metav1.Condition{
		Type:    v1alpha1.CondSynced,
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: msg,
	}
}
