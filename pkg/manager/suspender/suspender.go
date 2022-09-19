// Copyright 2022 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package suspender

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

var (
	suspendOrderForTC = []v1alpha1.MemberType{
		v1alpha1.TiDBMemberType,
		v1alpha1.TiFlashMemberType,
		v1alpha1.TiCDCMemberType,
		v1alpha1.TiKVMemberType,
		v1alpha1.PumpMemberType,
		v1alpha1.PDMemberType,
	}
	suspendOrderForDM = []v1alpha1.MemberType{
		v1alpha1.DMWorkerMemberType,
		v1alpha1.DMMasterMemberType,
	}

	_ Suspender = &suspender{}
	_ Suspender = &FakeSuspender{}
)

type Suspender interface {
	// SuspendComponent suspends the component if needed.
	//
	// Returns true if the component is needed to be suspended, and the reconciliation should be skipped.
	SuspendComponent(v1alpha1.Cluster, v1alpha1.MemberType) (bool, error)
}

func NewSuspender(deps *controller.Dependencies) Suspender {
	return &suspender{
		deps: deps,
	}
}

type suspendComponentCtx struct {
	cluster   v1alpha1.Cluster
	component v1alpha1.MemberType

	spec   v1alpha1.ComponentAccessor
	status v1alpha1.ComponentStatus
}

func (c *suspendComponentCtx) ComponentID() string {
	return fmt.Sprintf("%s/%s:%s", c.cluster.GetNamespace(), c.cluster.GetName(), c.component)
}

type suspender struct {
	deps *controller.Dependencies
}

// SuspendComponent suspends the component if needed.
//
// Returns true if the component is needed to be suspended, and the reconciliation should be skipped.
func (s *suspender) SuspendComponent(cluster v1alpha1.Cluster, comp v1alpha1.MemberType) (bool, error) {
	ctx := &suspendComponentCtx{
		cluster:   cluster,
		component: comp,
		spec:      cluster.ComponentSpec(comp),
		status:    cluster.ComponentStatus(comp),
	}
	if ctx.spec == nil || ctx.status == nil {
		return false, fmt.Errorf("spec or status for component %s is not found", ctx.ComponentID())
	}

	suspending := cluster.ComponentIsSuspending(comp)

	if !needsSuspendComponent(ctx.cluster, ctx.component) {
		if suspending {
			err := s.end(ctx)
			return true, err
		}

		klog.V(4).Infof("component %s is not needed to be suspended", ctx.ComponentID())
		return false, nil
	}

	if !suspending {
		if can, reason := canSuspendComponent(ctx.cluster, ctx.component); !can {
			klog.Warningf("component %s can not be suspended now because: %s", ctx.ComponentID(), reason)
			return false, nil
		}

		err := s.begin(ctx)
		return true, err
	}

	err := s.suspendResources(ctx, ctx.spec.SuspendAction())
	return true, err
}

func (s *suspender) suspendResources(ctx *suspendComponentCtx, action *v1alpha1.SuspendAction) error {
	if action == nil {
		return nil
	}

	errs := []error{}

	if action.SuspendStatefulSet {
		err := s.suspendSts(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errutil.NewAggregate(errs)
}

// suspendSts delete the statefulset and clear the status of the component.
func (s *suspender) suspendSts(ctx *suspendComponentCtx) error {
	ns := ctx.cluster.GetNamespace()
	name := ctx.cluster.GetName()
	stsName := controller.MemberName(name, ctx.component)

	_, err := s.deps.StatefulSetLister.StatefulSets(ns).Get(stsName)
	stsNotExist := errors.IsNotFound(err)
	if err != nil && !stsNotExist {
		return fmt.Errorf("failed to get sts %s/%s: %s", ns, stsName, err)
	}

	if !stsNotExist {
		// delete sts with foreground option.
		//
		// NOTE: For tidb cluster and dm cluster, operator delete sts for all components one by one,
		// so if any pod or sts deletion fails, the process will be stuck.
		klog.Infof("suspend statefulset %s/%s for component %s", ns, stsName, ctx.ComponentID())

		foreground := metav1.DeletePropagationForeground
		err = s.deps.KubeClientset.AppsV1().StatefulSets(ns).Delete(context.TODO(), stsName,
			metav1.DeleteOptions{PropagationPolicy: &foreground})
		if err != nil {
			return fmt.Errorf("failed to delete sts %s/%s: %s", ns, stsName, err)
		}
	} else {
		// clear the status after the sts is deleted actually, so that we can
		// use the `status.statefulset` is nil to determine whether sts suspension is done.
		ctx.status.SetSynced(false)
		ctx.status.SetStatefulSet(nil)
		switch ctx.component {
		case v1alpha1.PDMemberType:
			ctx.status.(*v1alpha1.PDStatus).Members = nil
			ctx.status.(*v1alpha1.PDStatus).Leader = v1alpha1.PDMember{}
		case v1alpha1.TiDBMemberType:
			ctx.status.(*v1alpha1.TiDBStatus).Members = nil
		case v1alpha1.TiKVMemberType:
			ctx.status.(*v1alpha1.TiKVStatus).Stores = nil
		case v1alpha1.TiFlashMemberType:
			ctx.status.(*v1alpha1.TiFlashStatus).Stores = nil
		case v1alpha1.PumpMemberType:
			ctx.status.(*v1alpha1.PumpStatus).Members = nil
		case v1alpha1.TiCDCMemberType:
			ctx.status.(*v1alpha1.TiCDCStatus).Captures = nil
		case v1alpha1.DMMasterMemberType:
			ctx.status.(*v1alpha1.MasterStatus).Members = nil
			ctx.status.(*v1alpha1.MasterStatus).Leader = v1alpha1.MasterMember{}
		case v1alpha1.DMWorkerMemberType:
			ctx.status.(*v1alpha1.WorkerStatus).Members = nil
		}
	}

	return nil
}

func (s *suspender) begin(ctx *suspendComponentCtx) error {
	status := ctx.status
	phase := v1alpha1.SuspendPhase
	klog.Infof("begin to suspend component %s and transfer phase from %s to %s",
		ctx.ComponentID(), status.GetPhase(), phase)
	ctx.status.SetPhase(phase)
	return nil
}

func (s *suspender) end(ctx *suspendComponentCtx) error {
	status := ctx.status
	phase := v1alpha1.NormalPhase
	klog.Infof("end to suspend component %s and transfer phase from %s to %s",
		ctx.ComponentID(), status.GetPhase(), phase)
	ctx.status.SetPhase(phase)
	return nil
}

// needsSuspendComponent returns whether suspender needs to to suspend the component
func needsSuspendComponent(cluster v1alpha1.Cluster, comp v1alpha1.MemberType) bool {
	spec := cluster.ComponentSpec(comp)
	if spec == nil {
		return false
	}
	action := spec.SuspendAction()
	if action == nil {
		return false
	}

	if action.SuspendStatefulSet {
		return true
	}

	return false
}

// canSuspendComponent checks whether suspender can start to suspend the component
func canSuspendComponent(cluster v1alpha1.Cluster, comp v1alpha1.MemberType) (bool, string) {
	// only support to suspend Normal or Suspend cluster
	// If the cluster is Upgrading or Scaling, the sts can not be deleted.
	if !cluster.ComponentIsNormal(comp) && !cluster.ComponentIsSuspending(comp) {
		return false, "component phase is not Normal or Suspend"
	}

	// wait for other components to be suspended
	var suspendOrder []v1alpha1.MemberType
	switch cluster.(type) {
	case *v1alpha1.TidbCluster:
		suspendOrder = suspendOrderForTC
	case *v1alpha1.DMCluster:
		suspendOrder = suspendOrderForDM
	}
	for _, typ := range suspendOrder {
		if typ == comp {
			break
		}

		if needsSuspendComponent(cluster, typ) && !cluster.ComponentIsSuspended(typ) {
			return false, fmt.Sprintf("wait another component %s to be suspended", typ)
		}

	}

	return true, ""
}
