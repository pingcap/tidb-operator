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

package runtime

import (
	"unsafe"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

type (
	PD      v1alpha1.PD
	PDGroup v1alpha1.PDGroup
)

type PDTuple struct{}

var _ InstanceTuple[*v1alpha1.PD, *PD] = PDTuple{}

func (PDTuple) From(t *v1alpha1.PD) *PD {
	return FromPD(t)
}

func (PDTuple) FromSlice(t []*v1alpha1.PD) []*PD {
	return FromPDSlice(t)
}

func (PDTuple) To(t *PD) *v1alpha1.PD {
	return ToPD(t)
}

func (PDTuple) ToSlice(t []*PD) []*v1alpha1.PD {
	return ToPDSlice(t)
}

type PDGroupTuple struct{}

var _ GroupTuple[*v1alpha1.PDGroup, *PDGroup] = PDGroupTuple{}

func (PDGroupTuple) From(t *v1alpha1.PDGroup) *PDGroup {
	return FromPDGroup(t)
}

func (PDGroupTuple) FromSlice(t []*v1alpha1.PDGroup) []*PDGroup {
	return FromPDGroupSlice(t)
}

func (PDGroupTuple) To(t *PDGroup) *v1alpha1.PDGroup {
	return ToPDGroup(t)
}

func (PDGroupTuple) ToSlice(t []*PDGroup) []*v1alpha1.PDGroup {
	return ToPDGroupSlice(t)
}

func FromPD(pd *v1alpha1.PD) *PD {
	return (*PD)(pd)
}

func ToPD(pd *PD) *v1alpha1.PD {
	return (*v1alpha1.PD)(pd)
}

func FromPDSlice(pds []*v1alpha1.PD) []*PD {
	return *(*[]*PD)(unsafe.Pointer(&pds))
}

func ToPDSlice(pds []*PD) []*v1alpha1.PD {
	return *(*[]*v1alpha1.PD)(unsafe.Pointer(&pds))
}

func FromPDGroup(pdg *v1alpha1.PDGroup) *PDGroup {
	return (*PDGroup)(pdg)
}

func ToPDGroup(pdg *PDGroup) *v1alpha1.PDGroup {
	return (*v1alpha1.PDGroup)(pdg)
}

func FromPDGroupSlice(pdgs []*v1alpha1.PDGroup) []*PDGroup {
	return *(*[]*PDGroup)(unsafe.Pointer(&pdgs))
}

func ToPDGroupSlice(pdgs []*PDGroup) []*v1alpha1.PDGroup {
	return *(*[]*v1alpha1.PDGroup)(unsafe.Pointer(&pdgs))
}

var _ Instance = &PD{}

func (pd *PD) DeepCopyObject() runtime.Object {
	return (*v1alpha1.PD)(pd).DeepCopyObject()
}

func (pd *PD) To() *v1alpha1.PD {
	return ToPD(pd)
}

func (pd *PD) GetTopology() v1alpha1.Topology {
	return pd.Spec.Topology
}

func (pd *PD) SetTopology(t v1alpha1.Topology) {
	pd.Spec.Topology = t
}

func (pd *PD) GetUpdateRevision() string {
	if pd.Labels == nil {
		return ""
	}
	return pd.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
}

func (pd *PD) IsHealthy() bool {
	return meta.IsStatusConditionTrue(pd.Status.Conditions, v1alpha1.PDCondHealth)
}

func (pd *PD) IsUpToDate() bool {
	return pd.Status.ObservedGeneration == pd.GetGeneration() && pd.GetUpdateRevision() == pd.Status.CurrentRevision
}

func (pd *PD) Conditions() []metav1.Condition {
	return pd.Status.Conditions
}

func (pd *PD) SetConditions(conds []metav1.Condition) {
	pd.Status.Conditions = conds
}

func (pd *PD) ObservedGeneration() int64 {
	return pd.Status.ObservedGeneration
}

func (pd *PD) SetObservedGeneration(g int64) {
	pd.Status.ObservedGeneration = g
}

func (pd *PD) Cluster() string {
	return pd.Spec.Cluster.Name
}

func (*PD) Component() string {
	return v1alpha1.LabelValComponentPD
}

var _ Group = &PDGroup{}

func (pdg *PDGroup) DeepCopyObject() runtime.Object {
	return (*v1alpha1.PDGroup)(pdg)
}

func (pdg *PDGroup) To() *v1alpha1.PDGroup {
	return ToPDGroup(pdg)
}

func (pdg *PDGroup) SetReplicas(replicas *int32) {
	pdg.Spec.Replicas = replicas
}

func (pdg *PDGroup) Replicas() *int32 {
	return pdg.Spec.Replicas
}

func (pdg *PDGroup) Cluster() string {
	return pdg.Spec.Cluster.Name
}

func (*PDGroup) Component() string {
	return v1alpha1.LabelValComponentPD
}

func (pdg *PDGroup) Conditions() []metav1.Condition {
	return pdg.Status.Conditions
}

func (pdg *PDGroup) SetConditions(conds []metav1.Condition) {
	pdg.Status.Conditions = conds
}

func (pdg *PDGroup) ObservedGeneration() int64 {
	return pdg.Status.ObservedGeneration
}

func (pdg *PDGroup) SetObservedGeneration(g int64) {
	pdg.Status.ObservedGeneration = g
}
