package coreutil

import (
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
		Reason:  "Ready",
		Message: "all subreources are ready",
	}
}

func Unready() *metav1.Condition {
	return &metav1.Condition{
		Type:    v1alpha1.CondReady,
		Status:  metav1.ConditionFalse,
		Reason:  "Unready",
		Message: "not all instances are ready",
	}
}

func Synced() *metav1.Condition {
	return &metav1.Condition{
		Type:    v1alpha1.CondSynced,
		Status:  metav1.ConditionTrue,
		Reason:  "Synced",
		Message: "all subreources are synced",
	}
}

func Unsynced() *metav1.Condition {
	return &metav1.Condition{
		Type:    v1alpha1.CondSynced,
		Status:  metav1.ConditionFalse,
		Reason:  "Unsynced",
		Message: "not all subreources are synced",
	}
}
