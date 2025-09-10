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

package adoption

import (
	"k8s.io/apimachinery/pkg/util/sets"
)

type groups struct {
	m map[string]sets.Set[string]
}

func newGroups() *groups {
	return &groups{
		m: make(map[string]sets.Set[string]),
	}
}

func (g *groups) add(groupKey, name string) {
	if groupKey == "" {
		return
	}
	s, ok := g.m[groupKey]
	if !ok {
		s = sets.New[string]()
	}
	s.Insert(name)
	g.m[groupKey] = s
}

func (g *groups) del(groupKey, name string) {
	if groupKey == "" {
		return
	}
	s, ok := g.m[groupKey]
	if !ok {
		return
	}
	s.Delete(name)
}

func (g *groups) list(groupKey string) []string {
	if groupKey == "" {
		return nil
	}
	s, ok := g.m[groupKey]
	if !ok {
		return nil
	}
	return sets.List(s)
}
