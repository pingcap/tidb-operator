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

import "github.com/pingcap/tidb-operator/pkg/runtime"

// e.g. for some write once fields(name, topology, etc.)
type UpdateHook[R runtime.Instance] interface {
	Update(update, outdated R) R
}

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

// Do not change name when update obj
func KeepName[R runtime.Instance]() UpdateHook[R] {
	return UpdateHookFunc[R](func(update, outdated R) R {
		update.SetName(outdated.GetName())
		return update
	})
}

// Do not change topology when update obj
func KeepTopology[R runtime.Instance]() UpdateHook[R] {
	return UpdateHookFunc[R](func(update, outdated R) R {
		update.SetTopology(outdated.GetTopology())
		return update
	})
}
