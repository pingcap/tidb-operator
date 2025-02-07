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

package v1alpha1

import "k8s.io/apimachinery/pkg/runtime/schema"

var _ Group = &FakeGroup{}

// FakeGroup is a mock implementation of the Group interface for testing purposes.
type FakeGroup struct {
	Name              string
	Namespace         string
	ClusterName       string
	ComponentKindVal  ComponentKind
	Generation        int64
	ObservedGen       int64
	CurrentRev        string
	UpdateRev         string
	CollisionCountVal *int32
	Healthy           bool
	DesiredReplicas   int32
	DesiredVersion    string
	ActualVersion     string
	Status            GroupStatus
}

func (f *FakeGroup) GetName() string {
	return f.Name
}

func (f *FakeGroup) GetNamespace() string {
	return f.Namespace
}

func (f *FakeGroup) GetClusterName() string {
	return f.ClusterName
}

func (f *FakeGroup) ComponentKind() ComponentKind {
	return f.ComponentKindVal
}

func (f *FakeGroup) GVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{}
}

func (f *FakeGroup) GetGeneration() int64 {
	return f.Generation
}

func (f *FakeGroup) ObservedGeneration() int64 {
	return f.ObservedGen
}

func (f *FakeGroup) CurrentRevision() string {
	return f.CurrentRev
}

func (f *FakeGroup) UpdateRevision() string {
	return f.UpdateRev
}

func (f *FakeGroup) CollisionCount() *int32 {
	return f.CollisionCountVal
}

func (f *FakeGroup) IsHealthy() bool {
	return f.Healthy
}

func (f *FakeGroup) GetDesiredReplicas() int32 {
	return f.DesiredReplicas
}

func (f *FakeGroup) GetDesiredVersion() string {
	return f.DesiredVersion
}

func (f *FakeGroup) GetActualVersion() string {
	return f.ActualVersion
}

func (f *FakeGroup) GetStatus() GroupStatus {
	return f.Status
}
