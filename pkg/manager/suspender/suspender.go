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
	suspendOrderForDM = []v1alpha1.MemberType{}
)

type Suspender interface {
	SuspendComponent(*v1alpha1.TidbCluster, v1alpha1.MemberType) (bool, error)
	SuspendDMComponent(*v1alpha1.DMCluster, v1alpha1.MemberType) (bool, error)
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

func buildContext(cluster v1alpha1.Cluster, comp v1alpha1.MemberType) *suspendComponentCtx {
	ctx := &suspendComponentCtx{
		cluster:   cluster,
		component: comp,
		spec:      cluster.ComponentSpec(comp),
		status:    cluster.ComponentStatus(comp),
	}
	return ctx
}

func (c *suspendComponentCtx) ComponentID() string {
	return fmt.Sprintf("%s/%s:%s", c.cluster.GetNamespace(), c.cluster.GetName(), c.component)
}

type suspender struct {
	deps *controller.Dependencies
}

func (s *suspender) SuspendComponent(tc *v1alpha1.TidbCluster, comp v1alpha1.MemberType) (bool, error) {
	ctx := buildContext(tc, comp)
	return s.suspendComponent(ctx)
}

func (s *suspender) SuspendDMComponent(dc *v1alpha1.DMCluster, comp v1alpha1.MemberType) (bool, error) {
	ctx := buildContext(dc, comp)
	return s.suspendComponent(ctx)
}

func (s *suspender) suspendComponent(ctx *suspendComponentCtx) (bool, error) {
	spec := ctx.spec
	status := ctx.status
	if spec == nil || status == nil {
		return false, fmt.Errorf("spec or status for component %s is not found", ctx.ComponentID())
	}

	suspended := isComponentSuspended(ctx)
	needed := needsSuspendComponent(ctx) && canSuspendComponent(ctx)

	if !needed {
		if suspended {
			err := s.end(ctx)
			return true, err
		}

		klog.V(4).Infof("component %s is not needed to be suspended", ctx.ComponentID())
		return false, nil
	}

	if !suspended {
		err := s.begion(ctx)
		return true, err
	}

	err := s.suspendResources(ctx, *spec.SuspendAction())
	return true, err
}

func (s *suspender) suspendResources(ctx *suspendComponentCtx, action v1alpha1.SuspendAction) error {
	errs := []error{}

	if action.SuspendStatefuleSet {
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
	stsName := fmt.Sprintf("%s-%s", name, ctx.component)

	_, err := s.deps.StatefulSetLister.StatefulSets(ns).Get(stsName)
	stsNotExist := errors.IsNotFound(err)
	if err != nil && !stsNotExist {
		return fmt.Errorf("failed to get sts %s/%s: %s", ns, stsName, err)
	}

	if !stsNotExist {
		klog.Infof("suspend statefulset %s/%s for component %s", ns, stsName, ctx.ComponentID())
		err = s.deps.KubeClientset.AppsV1().StatefulSets(ns).Delete(context.TODO(), stsName, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete sts %s/%s: %s", ns, stsName, err)
		}
	}

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

	return nil
}

func (s *suspender) begion(ctx *suspendComponentCtx) error {
	status := ctx.status

	ctx.status.SetPhase(v1alpha1.SuspendedPhase)

	klog.Infof("begin to suspend component %s and transfer phase from %s to %s", ctx.ComponentID(), status.GetPhase(), v1alpha1.SuspendedPhase)
	return nil
}

func (s *suspender) end(ctx *suspendComponentCtx) error {
	status := ctx.status

	ctx.status.SetPhase(v1alpha1.SuspendedPhase)
	klog.Infof("end to suspend component %s and transfer phase from %s to %s", ctx.ComponentID(), status.GetPhase(), v1alpha1.NormalPhase)
	return nil
}

func isComponentSuspended(ctx *suspendComponentCtx) bool {
	status := ctx.status
	if status == nil {
		return false
	}
	return status.GetPhase() == v1alpha1.SuspendedPhase
}

func needsSuspendComponent(ctx *suspendComponentCtx) bool {
	spec := ctx.spec

	if spec == nil {
		return false
	}
	action := spec.SuspendAction()
	if action == nil {
		return false
	}
	if !action.SuspendStatefuleSet {
		return false
	}

	return true
}

func canSuspendComponent(ctx *suspendComponentCtx) bool {
	status := ctx.status

	phase := status.GetPhase()
	if phase != v1alpha1.NormalPhase && phase != v1alpha1.SuspendedPhase {
		return false
	}

	// wait for other components to be suspended
	suspendOrder := []v1alpha1.MemberType{}
	switch ctx.cluster.(type) {
	case *v1alpha1.TidbCluster:
		suspendOrder = suspendOrderForTC
	case *v1alpha1.DMCluster:
		suspendOrder = suspendOrderForDM
	}
	for _, typ := range suspendOrder {
		if typ == ctx.component {
			break
		}

		// FIXME: maybe move to api package
		cctx := buildContext(ctx.cluster, typ)
		if needsSuspendComponent(cctx) && !isComponentSuspended(cctx) {
			return false
		}

	}

	return true
}
