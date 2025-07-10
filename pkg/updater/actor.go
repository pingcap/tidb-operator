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
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type NewFactory[R runtime.Instance] interface {
	New() R
}

type NewFunc[R runtime.Instance] func() R

func (f NewFunc[R]) New() R {
	return f()
}

type actor[T runtime.Tuple[O, R], O client.Object, R runtime.Instance] struct {
	c client.Client

	noInPlaceUpdate bool

	f NewFactory[R]

	converter T

	update   State[R]
	outdated State[R]
	// deleted set records all instances that are marked by defer delete annotation
	deleted State[R]

	addHooks    []AddHook[R]
	updateHooks []UpdateHook[R]
	delHooks    []DelHook[R]

	scaleInSelector Selector[R]
	updateSelector  Selector[R]
}

func (act *actor[T, O, R]) chooseToUpdate(s []R) (string, error) {
	name := act.updateSelector.Choose(s)
	if name == "" {
		return "", fmt.Errorf("no instance can be updated")
	}

	return name, nil
}

func (act *actor[T, O, R]) chooseToScaleIn(s []R) (string, error) {
	name := act.scaleInSelector.Choose(s)
	if name == "" {
		return "", fmt.Errorf("no instance can be scale in")
	}

	return name, nil
}

func (act *actor[T, O, R]) ScaleOut(ctx context.Context) error {
	logger := logr.FromContextOrDiscard(ctx)
	obj := act.f.New()

	for _, hook := range act.addHooks {
		obj = hook.Add(obj)
	}

	logger.Info("act scale out", "namespace", obj.GetNamespace(), "name", obj.GetName())

	if err := act.c.Apply(ctx, act.converter.To(obj)); err != nil {
		return err
	}

	act.update.Add(obj)

	return nil
}

func (act *actor[T, O, R]) ScaleInUpdate(ctx context.Context) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)
	name, err := act.chooseToScaleIn(act.update.List())
	if err != nil {
		return false, err
	}

	obj := act.update.Del(name)

	isUnavailable := !obj.IsReady() || !obj.IsUpToDate()

	logger.Info("act scale in update", "choosed", name, "isUnavailable", isUnavailable, "remain", act.update.Len())

	if err := act.c.Delete(ctx, act.converter.To(obj)); err != nil {
		return false, err
	}

	for _, hook := range act.delHooks {
		hook.Delete(obj.GetName())
	}

	return isUnavailable, nil
}

func (act *actor[T, O, R]) ScaleInOutdated(ctx context.Context) (bool, error) {
	return act.scaleInOutdated(ctx, true)
}

func (act *actor[T, O, R]) scaleInOutdated(ctx context.Context, deferDel bool) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)
	name, err := act.chooseToScaleIn(act.outdated.List())
	if err != nil {
		return false, err
	}

	obj := act.outdated.Del(name)
	isUnavailable := !obj.IsReady() || !obj.IsUpToDate()

	logger.Info("act scale in outdated",
		"choosed", name, "defer", deferDel, "isUnavailable", isUnavailable, "remain", act.outdated.Len())

	if deferDel {
		if err := act.deferDelete(ctx, obj); err != nil {
			return false, err
		}
	} else {
		if err := act.c.Delete(ctx, act.converter.To(obj)); err != nil {
			return false, err
		}
	}

	for _, hook := range act.delHooks {
		hook.Delete(obj.GetName())
	}

	return isUnavailable, nil
}

type Patch struct {
	Metadata Metadata `json:"metadata"`
}

type Metadata struct {
	ResourceVersion string            `json:"resourceVersion"`
	Annotations     map[string]string `json:"annotations"`
}

func (act *actor[T, O, R]) deferDelete(ctx context.Context, obj R) error {
	o := act.converter.To(obj)
	p := Patch{
		Metadata: Metadata{
			ResourceVersion: o.GetResourceVersion(),
			Annotations: map[string]string{
				v1alpha1.AnnoKeyDeferDelete: v1alpha1.AnnoValTrue,
			},
		},
	}

	data, err := json.Marshal(&p)
	if err != nil {
		return fmt.Errorf("invalid patch: %w", err)
	}

	if err := act.c.Patch(ctx, o, client.RawPatch(types.MergePatchType, data)); err != nil {
		return fmt.Errorf("cannot mark obj %s/%s as defer delete: %w", obj.GetNamespace(), obj.GetName(), err)
	}

	act.deleted.Add(obj)

	return nil
}

func (act *actor[T, O, R]) Update(ctx context.Context) error {
	logger := logr.FromContextOrDiscard(ctx)
	if act.noInPlaceUpdate {
		if _, err := act.scaleInOutdated(ctx, false); err != nil {
			return err
		}
		if err := act.ScaleOut(ctx); err != nil {
			return err
		}

		return nil
	}

	name, err := act.chooseToUpdate(act.outdated.List())
	if err != nil {
		return err
	}

	outdated := act.outdated.Del(name)

	update := act.f.New()
	for _, hook := range act.updateHooks {
		update = hook.Update(update, outdated)
	}

	logger.Info("act update", "choosed", name, "remain", act.outdated.Len())

	if err := act.c.Apply(ctx, act.converter.To(update)); err != nil {
		return err
	}

	act.update.Add(update)

	return nil
}

func (act *actor[T, O, R]) Cleanup(ctx context.Context) error {
	for _, item := range act.deleted.List() {
		if err := act.c.Delete(ctx, act.converter.To(item)); err != nil {
			return err
		}
	}

	return nil
}
