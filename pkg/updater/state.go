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

package updater

import (
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type State[R runtime.Instance] interface {
	Add(obj R)
	Del(name string) R
	Get(name string) R
	List() []R
	Len() int
}

type state[R runtime.Instance] struct {
	nameToObj map[string]R
}

func NewState[R runtime.Instance](instances []R) State[R] {
	nameToObj := make(map[string]R)
	for _, instance := range instances {
		nameToObj[instance.GetName()] = instance
	}
	return &state[R]{
		nameToObj: nameToObj,
	}
}

func (s *state[R]) Add(obj R) {
	s.nameToObj[obj.GetName()] = obj
}

func (s *state[R]) Del(name string) R {
	obj := s.nameToObj[name]
	delete(s.nameToObj, name)
	return obj
}

func (s *state[R]) Get(name string) R {
	return s.nameToObj[name]
}

func (s *state[R]) List() []R {
	l := make([]R, 0, len(s.nameToObj))
	for _, obj := range s.nameToObj {
		l = append(l, obj)
	}
	return l
}

func (s *state[R]) Len() int {
	return len(s.nameToObj)
}
