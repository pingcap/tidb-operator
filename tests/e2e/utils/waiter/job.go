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

	batchv1 "k8s.io/api/batch/v1"

	"github.com/pingcap/tidb-operator/pkg/client"
)

func WaitForJobComplete(ctx context.Context, c client.Client, job *batchv1.Job, timeout time.Duration) error {
	return WaitForObject(ctx, c, job, func() error {
		for _, cond := range job.Status.Conditions {
			switch cond.Type {
			case batchv1.JobComplete:
				return nil
			case batchv1.JobFailed:
				return fmt.Errorf("job is failed: %v", cond.Message)
			}
		}

		return fmt.Errorf("job status is unknown")
	}, timeout)
}
