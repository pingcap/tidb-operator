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
	"context"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type NewFactory[PT runtime.Instance] interface {
	New() PT
}

type NewFunc[PT runtime.Instance] func() PT

func (f NewFunc[PT]) New() PT {
	return f()
}

// e.g. for some write once fields(name, topology, etc.)
type UpdateHook[PT runtime.Instance] interface {
	Update(update, outdated PT) PT
}

// e.g. for topology scheduling
type AddHook[PT runtime.Instance] interface {
	Add(update PT) PT
}

type DelHook[PT runtime.Instance] interface {
	Delete(name string)
}

type UpdateHookFunc[PT runtime.Instance] func(update, outdated PT) PT

func (f UpdateHookFunc[PT]) Update(update, outdated PT) PT {
	return f(update, outdated)
}

type actor[PT runtime.Instance] struct {
	c client.Client

	f NewFactory[PT]

	update   State[PT]
	outdated State[PT]

	addHooks    []AddHook[PT]
	updateHooks []UpdateHook[PT]
	delHooks    []DelHook[PT]

	scaleInSelector Selector[PT]
	updateSelector  Selector[PT]
}

func (act *actor[PT]) chooseToUpdate(s []PT) (string, error) {
	name := act.updateSelector.Choose(s)
	if name == "" {
		return "", fmt.Errorf("no instance can be updated")
	}

	return name, nil
}

func (act *actor[PT]) chooseToScaleIn(s []PT) (string, error) {
	name := act.scaleInSelector.Choose(s)
	if name == "" {
		return "", fmt.Errorf("no instance can be scale in")
	}

	return name, nil
}

func (act *actor[PT]) ScaleOut(ctx context.Context) error {
	obj := act.f.New()

	for _, hook := range act.addHooks {
		obj = hook.Add(obj)
	}

	if err := act.c.Apply(ctx, obj.To()); err != nil {
		return err
	}

	act.update.Add(obj)

	return nil
}

func (act *actor[PT]) ScaleInUpdate(ctx context.Context) (bool, error) {
	name, err := act.chooseToScaleIn(act.update.List())
	if err != nil {
		return false, err
	}
	obj := act.update.Del(name)

	isUnavailable := !obj.IsHealthy() || !obj.IsUpToDate()

	if err := act.c.Delete(ctx, obj.To()); err != nil {
		return false, err
	}

	for _, hook := range act.delHooks {
		hook.Delete(obj.GetName())
	}

	return isUnavailable, nil
}

func (act *actor[PT]) ScaleInOutdated(ctx context.Context) (bool, error) {
	name, err := act.chooseToScaleIn(act.outdated.List())
	if err != nil {
		return false, err
	}
	obj := act.outdated.Del(name)
	isUnavailable := !obj.IsHealthy() || !obj.IsUpToDate()

	if err := act.c.Delete(ctx, obj.To()); err != nil {
		return false, err
	}

	for _, hook := range act.delHooks {
		hook.Delete(obj.GetName())
	}

	return isUnavailable, nil
}

func (act *actor[PT]) Update(ctx context.Context) error {
	name, err := act.chooseToUpdate(act.outdated.List())
	if err != nil {
		return err
	}
	outdated := act.outdated.Del(name)

	update := act.f.New()
	for _, hook := range act.updateHooks {
		update = hook.Update(update, outdated)
	}

	if err := act.c.Apply(ctx, update.To()); err != nil {
		return err
	}

	act.update.Add(update)

	return nil
}
