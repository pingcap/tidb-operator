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

package waiter

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"

	"github.com/pingcap/tidb-operator/pkg/apicall"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

func WaitForInstanceList[
	GS scope.GroupInstance[GF, GT, IS],
	IS scope.List[IL, I],
	GF client.Object,
	GT runtime.Group,
	IL client.ObjectList,
	I client.Object,
](
	ctx context.Context,
	c client.Client,
	g GF,
	cond func(items []I) error,
	timeout time.Duration,
) error {
	var lastErr error
	if err := wait.PollUntilContextTimeout(ctx, Poll, timeout, true, func(ctx context.Context) (bool, error) {
		items, err := apicall.ListInstances[GS](ctx, c, g)
		if err != nil {
			return false, fmt.Errorf("can't list instances: %w", err)
		}

		if err := cond(items); err != nil {
			lastErr = err
			return false, nil
		}

		return true, nil
	}); err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("wait for instance list %T(%v) condition timeout: %w", g, client.ObjectKeyFromObject(g), lastErr)
		}

		return fmt.Errorf("can't wait for instance list %T(%v) condition, error : %w", g, client.ObjectKeyFromObject(g), err)
	}

	return nil
}

func WaitForInstanceListDeleted[
	GS scope.GroupInstance[GF, GT, IS],
	IS scope.List[IL, I],
	GF client.Object,
	GT runtime.Group,
	IL client.ObjectList,
	I client.Object,
](
	ctx context.Context,
	c client.Client,
	g GF,
) error {
	return WaitForInstanceList[GS](ctx, c, g, ListIsEmpty, LongTaskTimeout)
}

func WaitForOneInstanceDeleting[
	GS scope.GroupInstance[GF, GT, IS],
	IS scope.List[IL, I],
	GF client.Object,
	GT runtime.Group,
	IL client.ObjectList,
	I client.Object,
](
	ctx context.Context,
	c client.Client,
	g GF,
	target *I,
) error {
	return WaitForInstanceList[GS](ctx, c, g, OneDeleting(target), ShortTaskTimeout)
}

func ListIsEmpty[I client.Object](items []I) error {
	if len(items) == 0 {
		return nil
	}

	return fmt.Errorf("there are still %v items", len(items))
}

func OneDeleting[I client.Object](target *I) func(items []I) error {
	return func(items []I) error {
		var deleting []I
		for _, item := range items {
			if !item.GetDeletionTimestamp().IsZero() {
				deleting = append(deleting, item)
			}
		}
		if len(deleting) != 1 {
			return fmt.Errorf("expected only one instance is deleting, actual: %v", len(deleting))
		}
		*target = deleting[0]
		return nil
	}
}

// WatchUntilInstanceList use watch to ensure something is not happened
func WatchUntilInstanceList[
	GS scope.GroupInstance[GF, GT, IS],
	IS scope.List[IL, PI],
	GF client.Object,
	GT runtime.Group,
	IL client.ObjectList,
	PI scope.ClientObject[I],
	I any,
](
	ctx context.Context,
	c client.Client,
	g GF,
	cond func(item PI) (bool, error),
	timeout time.Duration,
	synced chan struct{},
) error {
	lw := apicall.NewInstanceListerWatcher[GS](ctx, c, g)
	var obj PI = new(I)
	if _, err := watchtools.UntilWithSync(ctx, lw, obj, func(store cache.Store) (bool, error) {
		if synced != nil {
			close(synced)
		}
		return false, nil
	}, func(event watch.Event) (bool, error) {
		instance, ok := event.Object.(PI)
		if !ok {
			// ignore events without instance
			return false, nil
		}

		done, err := cond(instance)
		if err != nil {
			return false, err
		}

		return done, nil
	}); err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("wait for instance list %T(%v) condition timeout", g, client.ObjectKeyFromObject(g))
		}

		return fmt.Errorf("can't wait for instance list %T(%v) condition, error : %w", g, client.ObjectKeyFromObject(g), err)
	}

	return nil
}
