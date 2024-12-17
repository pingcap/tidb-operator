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

type State[PT runtime.Instance] interface {
	Add(obj PT)
	Del(name string) PT
	Get(name string) PT
	List() []PT
	Len() int
}

type state[PT runtime.Instance] struct {
	nameToObj map[string]PT
}

func NewState[PT runtime.Instance](instances []PT) State[PT] {
	nameToObj := make(map[string]PT)
	for _, instance := range instances {
		nameToObj[instance.GetName()] = instance
	}
	return &state[PT]{
		nameToObj: nameToObj,
	}
}

func (s *state[PT]) Add(obj PT) {
	s.nameToObj[obj.GetName()] = obj
}

func (s *state[PT]) Del(name string) PT {
	obj := s.nameToObj[name]
	delete(s.nameToObj, name)
	return obj
}

func (s *state[PT]) Get(name string) PT {
	return s.nameToObj[name]
}

func (s *state[PT]) List() []PT {
	l := make([]PT, 0, len(s.nameToObj))
	for _, obj := range s.nameToObj {
		l = append(l, obj)
	}
	return l
}

func (s *state[PT]) Len() int {
	return len(s.nameToObj)
}
