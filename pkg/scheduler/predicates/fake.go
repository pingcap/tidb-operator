// Copyright 2020 PingCAP, Inc.
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

package predicates

import (
	v1 "k8s.io/api/core/v1"
)

type FakePredicate struct {
	FakeName string
	Nodes    []v1.Node
	Err      error
}

var _ Predicate = &FakePredicate{}

func NewFakePredicate(name string, nodes []v1.Node, err error) *FakePredicate {
	return &FakePredicate{}
}

func (f *FakePredicate) Name() string {
	return f.FakeName
}

func (f *FakePredicate) Filter(_ string, _ *v1.Pod, nodes []v1.Node) ([]v1.Node, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Nodes != nil {
		return f.Nodes, nil
	}
	return nodes, nil
}
