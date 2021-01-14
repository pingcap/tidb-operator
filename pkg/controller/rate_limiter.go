// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

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
