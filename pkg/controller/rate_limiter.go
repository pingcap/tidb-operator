package controller

import (
	"time"

	"golang.org/x/time/rate"
	wq "k8s.io/client-go/util/workqueue"
)

// NewControllerRateLimiter returns a RateLimiter, which limit the request rate by the stricter one's desicion of two
// RateLimiters: ItemExponentialFailureRateLimiter and BucketRateLimiter
func NewControllerRateLimiter(baseDelay, maxDelay time.Duration) wq.RateLimiter {
	return wq.NewMaxOfRateLimiter(
		wq.NewItemExponentialFailureRateLimiter(baseDelay, maxDelay),
		// 10 qps, 100 bucket size.  This is only for retry speed and its only the overall factor (not per item)
		&wq.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
	)
}
