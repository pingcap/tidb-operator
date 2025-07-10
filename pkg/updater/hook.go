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
	"github.com/pingcap/tidb-operator/pkg/utils/tracker"
)

// UpdateHook a hook executes when update an instance.
// e.g. for some write once fields(name, topology, etc.)
type UpdateHook[R runtime.Instance] interface {
	Update(update, outdated R) R
}

// AddHook a hook executes when add an instance.
// e.g. for topology scheduling
type AddHook[R runtime.Instance] interface {
	Add(update R) R
}

type DelHook[R runtime.Instance] interface {
	Delete(name string)
}

type UpdateHookFunc[R runtime.Instance] func(update, outdated R) R

func (f UpdateHookFunc[R]) Update(update, outdated R) R {
	return f(update, outdated)
}

type AddHookFunc[PT runtime.Instance] func(update PT) PT

func (f AddHookFunc[PT]) Add(update PT) PT {
	return f(update)
}

type DelHookFunc[PT runtime.Instance] func(name string)

func (f DelHookFunc[PT]) Delete(name string) {
	f(name)
}

// KeepName does not change name when update obj
func KeepName[R runtime.Instance]() UpdateHook[R] {
	return UpdateHookFunc[R](func(update, outdated R) R {
		update.SetName(outdated.GetName())
		return update
	})
}

// KeepTopology does not change topology when update obj
func KeepTopology[R runtime.Instance]() UpdateHook[R] {
	return UpdateHookFunc[R](func(update, outdated R) R {
		update.SetTopology(outdated.GetTopology())
		return update
	})
}

// AllocateName will set name for new instance
// If a new name is allocated, it will be recorded until it is observed.
// This hook is defined to avoid dirty read(no read-after-write consistency) when scale out.
// For example, when a new item is created but not observed in next reconciliation,
// updater will create a new one again with a new name.
// It means an unexpected item may be created.
// This hook can ensure that updater will try to create the item again if it is not observed
func AllocateName[R runtime.Instance](allocator tracker.Allocator) AddHook[R] {
	count := 0
	return AddHookFunc[R](func(update R) R {
		name := allocator.Allocate(count)
		count++
		update.SetName(name)

		return update
	})
}
