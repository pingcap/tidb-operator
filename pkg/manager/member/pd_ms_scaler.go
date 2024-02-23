// Copyright 2023 PingCAP, Inc.
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

package member

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"

	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type pdMSScaler struct {
	generalScaler
}

// NewPDMSScaler returns a PD Micro Service Scaler.
func NewPDMSScaler(deps *controller.Dependencies) *pdMSScaler {
	return &pdMSScaler{generalScaler: generalScaler{deps: deps}}
}

// Scale scales in or out of the statefulset.
func (s *pdMSScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return s.ScaleOut(meta, oldSet, newSet)
	} else if scaling < 0 {
		return s.ScaleIn(meta, oldSet, newSet)
	}
	return nil
}

// ScaleOut scales out of the statefulset.
func (s *pdMSScaler) ScaleOut(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc, ok := meta.(*v1alpha1.TidbCluster)
	if !ok {
		return nil
	}

	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	serviceName := controller.PDMSTrimName(oldSet.Name)

	klog.Infof("scaling out PDMS component %s for cluster [%s/%s] statefulset, ordinal: %d (replicas: %d, delete slots: %v)", serviceName, oldSet.Namespace, tcName, ordinal, replicas, deleteSlots.List())
	if !tc.Status.PDMS[serviceName].Synced {
		return fmt.Errorf("PDMS component %s for cluster [%s/%s] status sync failed, can't scale out now", serviceName, ns, tcName)
	}

	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

// ScaleIn scales in of the statefulset.
func (s *pdMSScaler) ScaleIn(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc, ok := meta.(*v1alpha1.TidbCluster)
	if !ok {
		return nil
	}

	// NOW, we can only remove one member at a time when scaling in
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	serviceName := controller.PDMSTrimName(oldSet.Name)

	klog.Infof("scaling in PDMS component %s for cluster [%s/%s] statefulset, ordinal: %d (replicas: %d, delete slots: %v)", serviceName, oldSet.Namespace, tcName, ordinal, replicas, deleteSlots.List())
	if !tc.Status.PDMS[serviceName].Synced {
		return fmt.Errorf("PDMS component %s for cluster [%s/%s] status sync failed, can't scale in now", serviceName, ns, tcName)
	}

	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

type fakePDMSScaler struct{}

// NewFakePDMSScaler returns a fake Scaler
func NewFakePDMSScaler() Scaler {
	return &fakePDMSScaler{}
}

func (s *fakePDMSScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if *newSet.Spec.Replicas > *oldSet.Spec.Replicas {
		return s.ScaleOut(meta, oldSet, newSet)
	} else if *newSet.Spec.Replicas < *oldSet.Spec.Replicas {
		return s.ScaleIn(meta, oldSet, newSet)
	}
	return nil
}

func (s *fakePDMSScaler) ScaleOut(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas+1, nil)
	return nil
}

func (s *fakePDMSScaler) ScaleIn(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas-1, nil)
	return nil
}
