package waiter

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

func WaitForInstance[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](
	ctx context.Context,
	c client.Client,
	instance F,
	cond func(instance F) error,
	timeout time.Duration,
) error {
	var lastErr error
	if err := wait.PollUntilContextTimeout(ctx, Poll, timeout, true, func(ctx context.Context) (bool, error) {
		key := client.ObjectKeyFromObject(instance)
		if err := c.Get(ctx, key, instance); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}

			return false, fmt.Errorf("can't get obj %s: %w", key, err)
		}

		if err := cond(instance); err != nil {
			lastErr = err
			return false, nil
		}

		return true, nil
	}); err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("wait for instance %T(%v) condition timeout: %w", instance, client.ObjectKeyFromObject(instance), lastErr)
		}

		return fmt.Errorf("can't wait for instance %T(%v) condition, error : %w", instance, client.ObjectKeyFromObject(instance), err)
	}

	return nil
}
