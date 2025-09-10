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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

var (
	Poll = time.Second * 2

	ShortTaskTimeout = time.Minute * 3
	LongTaskTimeout  = time.Minute * 10
)

func WaitForObject(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	cond func() error,
	timeout time.Duration,
) error {
	var lastErr error
	if err := wait.PollUntilContextTimeout(ctx, Poll, timeout, true, func(ctx context.Context) (bool, error) {
		key := client.ObjectKeyFromObject(obj)
		if err := c.Get(ctx, key, obj); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}

			return false, fmt.Errorf("can't get obj %s: %w", key, err)
		}

		if err := cond(); err != nil {
			lastErr = err
			return false, nil
		}

		return true, nil
	}); err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("wait for object %T(%v) condition timeout: %w", obj, client.ObjectKeyFromObject(obj), lastErr)
		}

		return fmt.Errorf("can't wait for object %T(%v) condition, error : %w", obj, client.ObjectKeyFromObject(obj), err)
	}

	return nil
}

func WaitForObjectV2(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	cond func() (stop bool, _ error),
	timeout time.Duration,
) error {
	var lastErr error
	if err := wait.PollUntilContextTimeout(ctx, Poll, timeout, true, func(ctx context.Context) (bool, error) {
		key := client.ObjectKeyFromObject(obj)
		if err := c.Get(ctx, key, obj); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}

			return false, fmt.Errorf("can't get obj %s: %w", key, err)
		}

		stop, err := cond()
		if err != nil {
			if stop {
				return false, err
			}
			lastErr = err
		}

		return stop, nil
	}); err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("wait for object %T(%v) condition timeout: %w", obj, client.ObjectKeyFromObject(obj), lastErr)
		}

		return fmt.Errorf("can't wait for object %T(%v) condition, error : %w", obj, client.ObjectKeyFromObject(obj), err)
	}

	return nil
}

func WaitForObjectDeleted(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	timeout time.Duration,
) error {
	if err := wait.PollUntilContextTimeout(ctx, Poll, timeout, true, func(ctx context.Context) (bool, error) {
		key := client.ObjectKeyFromObject(obj)
		if err := c.Get(ctx, key, obj); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}

			return false, fmt.Errorf("can't get object %s: %w", key, err)
		}

		return false, nil
	}); err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("wait for object %v deleted timeout: %w", obj, err)
		}

		return fmt.Errorf("can't wait for object %v deleted, error : %w", obj, err)
	}

	return nil
}

func WaitForList(
	ctx context.Context,
	c client.Client,
	list client.ObjectList,
	cond func() error,
	timeout time.Duration,
	opts ...client.ListOption,
) error {
	var lastErr error
	if err := wait.PollUntilContextTimeout(ctx, Poll, timeout, true, func(ctx context.Context) (bool, error) {
		if err := c.List(ctx, list, opts...); err != nil {
			return false, fmt.Errorf("can't list object %T: %w", list, err)
		}

		if err := cond(); err != nil {
			lastErr = err
			return false, nil
		}

		return true, nil
	}); err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("wait for list %T condition timeout: %w", list, lastErr)
		}

		return fmt.Errorf("can't wait for list %T condition, error : %w", list, err)
	}

	return nil
}

func WaitForListDeleted(
	ctx context.Context,
	c client.Client,
	list client.ObjectList,
	timeout time.Duration,
	opts ...client.ListOption,
) error {
	return WaitForList(ctx, c, list, func() error {
		l := meta.LenList(list)
		if l == 0 {
			return nil
		}

		return fmt.Errorf("there are still %v items", l)
	}, timeout, opts...)
}

func WaitForObjectCondition[T runtime.ObjectTuple[O, U], O client.Object, U runtime.Object](
	ctx context.Context,
	c client.Client,
	obj O,
	condType string,
	status metav1.ConditionStatus,
	timeout time.Duration,
) error {
	var t T
	return WaitForObject(ctx, c, obj, func() error {
		ro := t.From(obj)
		cond := meta.FindStatusCondition(ro.Conditions(), condType)
		if cond == nil {
			return fmt.Errorf("obj %s/%s's condition %s is not set", obj.GetNamespace(), obj.GetName(), condType)
		}
		if cond.Status == status {
			return nil
		}
		return fmt.Errorf("obj %s/%s's condition %s has unexpected status, expected is %v, current is %v, reason: %v, message: %v",
			obj.GetNamespace(),
			obj.GetName(),
			cond.Type,
			status,
			cond.Status,
			cond.Reason,
			cond.Message,
		)
	}, timeout)
}

func checkInstanceStatus(kind, name, ns string, generation int64, status v1alpha1.CommonStatus) error {
	objID := fmt.Sprintf("%s %s/%s", kind, ns, name)
	if generation != status.ObservedGeneration || !meta.IsStatusConditionPresentAndEqual(status.Conditions, v1alpha1.CondSynced, metav1.ConditionTrue) {
		return fmt.Errorf("%s is not synced", objID)
	}
	if !meta.IsStatusConditionPresentAndEqual(status.Conditions, v1alpha1.CondReady, metav1.ConditionTrue) {
		return fmt.Errorf("%s is not ready", objID)
	}
	return nil
}
