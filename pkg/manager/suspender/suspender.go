// Copyright 2021 PingCAP, Inc.
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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errutil "k8s.io/apimachinery/pkg/util/errors"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

type Suspender interface {
	Suspend(v1alpha1.MemberType, *v1alpha1.TidbCluster) error
}

type suspender struct {
	deps *controller.Dependencies
}

func (s *suspender) Suspend(typ v1alpha1.MemberType, tc *v1alpha1.TidbCluster) error {
	spec := tc.ComponentSpec(typ)
	status := tc.ComponentStatus(typ)

	suspended := tc.ComponentIsSuspended(typ)
	needed := s.needsSuspend(spec, status)

	if !needed {
		if suspended {
			err := s.resume()
			if err != nil {
				return err
			}

			status.SetPhase(v1alpha1.NormalPhase)
		}
		return nil
	}

	if !suspended {
		status.SetPhase(v1alpha1.SuspendedPhase)
		return nil
	}

	return s.suspend(tc, status, *spec.SuspendAction())
}

func (s *suspender) suspend(cluster metav1.Object, status v1alpha1.ComponentStatus, action v1alpha1.SuspendAction) error {
	ns := cluster.GetNamespace()
	name := cluster.GetName()
	component := status.MemberType()

	errs := []error{}
	if action.SuspendStatefuleSet {
		// FIXME: use a common function
		stsName := fmt.Sprintf("%s-%s", name, component)
		err := s.deps.KubeClientset.AppsV1().StatefulSets(ns).Delete(context.TODO(), stsName, metav1.DeleteOptions{})
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errutil.NewAggregate(errs)
}

func (s *suspender) resume() error {
	return nil
}

func (s *suspender) suspendSts(cluster metav1.Object, status v1alpha1.ComponentStatus) error {
	ns := cluster.GetNamespace()
	name := cluster.GetName()

	stsName := fmt.Sprintf("%s-%s", name, status.MemberType())
	return s.deps.KubeClientset.AppsV1().StatefulSets(ns).Delete(stsName, &metav1.DeleteOptions{})
}

func (s *suspender) needsSuspend(spec v1alpha1.ComponentAccessor, status v1alpha1.ComponentStatus) bool {
	// check if need to suspend
	action := spec.SuspendAction()
	if action == nil {
		return false
	}

	if !action.SuspendStatefuleSet {
		return false
	}

	// check if can suspend
	phase := status.GetPhase()
	if phase != v1alpha1.NormalPhase && phase != v1alpha1.SuspendedPhase {
		return false
	}

	return true
}
