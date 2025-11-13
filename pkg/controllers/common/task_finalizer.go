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

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TaskFinalizerAdd[
	S scope.Object[F, T],
	F Object[O],
	T runtime.Object,
	O any,
](state ObjectState[F], c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerAdd", func(ctx context.Context) task.Result {
		obj := state.Object()
		if err := k8s.EnsureFinalizer(ctx, c, obj); err != nil {
			return task.Fail().With("failed to ensure finalizer has been added: %v", err)
		}
		return task.Complete().With("finalizer is added")
	})
}

// Deprecated: prefer TaskFinalizerAdd, remove it
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

// Deprecated: prefer TaskFinalizerAdd, remove it
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

type GroupFinalizerDelState[
	GF client.Object,
	I client.Object,
] interface {
	ObjectState[GF]
	SliceState[I]
}

func TaskGroupFinalizerDel[
	S scope.Group[GF, GT],
	GF client.Object,
	GT runtime.Group,
	I client.Object,
](state GroupFinalizerDelState[GF, I], c client.Client) task.Task {
	return task.NameTaskFunc("FinalizerDel", func(ctx context.Context) task.Result {
		var errList []error
		var names []string
		for _, peer := range state.InstanceSlice() {
			names = append(names, peer.GetName())
			if peer.GetDeletionTimestamp().IsZero() {
				if err := c.Delete(ctx, peer); err != nil {
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

		obj := state.Object()
		wait, err := k8s.DeleteGroupSubresource(ctx, c, scope.From[S](obj), &corev1.ServiceList{})
		if err != nil {
			return task.Fail().With("cannot delete subresources: %w", err)
		}
		if wait {
			return task.Retry(defaultDelWaitTime).With("wait all subresources deleted")
		}

		if err := k8s.RemoveFinalizer(ctx, c, obj); err != nil {
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
