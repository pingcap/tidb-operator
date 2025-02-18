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

package common

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilerr "k8s.io/apimachinery/pkg/util/errors"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TaskGroupFinalizerAdd[
	GT runtime.GroupTuple[OG, RG],
	OG client.Object,
	RG runtime.Group,
](state GroupState[RG], c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerAdd", func(ctx context.Context) task.Result {
		var t GT
		if err := k8s.EnsureFinalizer(ctx, c, t.To(state.Group())); err != nil {
			return task.Fail().With("failed to ensure finalizer has been added: %v", err)
		}
		return task.Complete().With("finalizer is added")
	})
}

func TaskInstanceFinalizerAdd[
	IT runtime.InstanceTuple[OI, RI],
	OI client.Object,
	RI runtime.Instance,
](state InstanceState[RI], c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerAdd", func(ctx context.Context) task.Result {
		var t IT
		if err := k8s.EnsureFinalizer(ctx, c, t.To(state.Instance())); err != nil {
			return task.Fail().With("failed to ensure finalizer has been added: %v", err)
		}
		return task.Complete().With("finalizer is added")
	})
}

const defaultDelWaitTime = 10 * time.Second

func TaskGroupFinalizerDel[
	GT runtime.GroupTuple[OG, RG],
	IT runtime.InstanceTuple[OI, RI],
	OG client.Object,
	RG runtime.Group,
	OI client.Object,
	RI runtime.Instance,
](state GroupAndInstanceSliceState[RG, RI], c client.Client) task.Task {
	var it IT
	var gt GT
	return task.NameTaskFunc("FinalizerDel", func(ctx context.Context) task.Result {
		var errList []error
		var names []string
		for _, peer := range state.Slice() {
			names = append(names, peer.GetName())
			if peer.GetDeletionTimestamp().IsZero() {
				if err := c.Delete(ctx, it.To(peer)); err != nil {
					if errors.IsNotFound(err) {
						continue
					}
					errList = append(errList, fmt.Errorf("try to delete the instance %v failed: %w", peer.GetName(), err))
					continue
				}
			}
		}

		if len(errList) != 0 {
			return task.Fail().With("failed to delete all instances: %v", utilerr.NewAggregate(errList))
		}

		if len(names) != 0 {
			return task.Retry(defaultDelWaitTime).With("wait for all instances being removed, %v still exists", names)
		}

		wait, err := k8s.DeleteGroupSubresource(ctx, c, state.Group(), &corev1.ServiceList{})
		if err != nil {
			return task.Fail().With("cannot delete subresources: %w", err)
		}
		if wait {
			return task.Retry(defaultDelWaitTime).With("wait all subresources deleted")
		}

		if err := k8s.RemoveFinalizer(ctx, c, gt.To(state.Group())); err != nil {
			return task.Fail().With("failed to ensure finalizer has been removed: %w", err)
		}

		return task.Complete().With("finalizer has been removed")
	})
}

func TaskJobFinalizerAdd[
	J runtime.Job,
](state JobState[J], c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerAdd", func(ctx context.Context) task.Result {
		if err := k8s.EnsureFinalizer(ctx, c, state.Job().Object()); err != nil {
			return task.Fail().With("failed to ensure finalizer has been added: %v", err)
		}
		return task.Complete().With("finalizer is added")
	})
}

func TaskJobFinalizerDel[
	J runtime.Job,
](state JobState[J], c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerDel", func(ctx context.Context) task.Result {
		if err := k8s.RemoveFinalizer(ctx, c, state.Job().Object()); err != nil {
			return task.Fail().With("failed to ensure finalizer has been removed: %v", err)
		}
		return task.Complete().With("finalizer is removed")
	})
}
