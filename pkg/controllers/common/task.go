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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/v2/pkg/utils/map"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

func TaskSuspendPod(state PodState, c client.Client) task.Task {
	return task.NameTaskFunc("SuspendPod", func(ctx context.Context) task.Result {
		pod := state.Pod()
		if pod == nil {
			return task.Complete().With("pod has been deleted")
		}
		if !pod.GetDeletionTimestamp().IsZero() {
			return task.Complete().With("pod has been terminating")
		}
		if err := c.Delete(ctx, pod); err != nil {
			if errors.IsNotFound(err) {
				return task.Complete().With("pod is deleted")
			}
			return task.Fail().With("can't delete pod %s/%s: %v", pod.Namespace, pod.Name, err)
		}

		return task.Retry(task.DefaultRequeueAfter).With("pod is deleting")
	})
}

type ContextObjectNewer[
	F client.Object,
] interface {
	Key() types.NamespacedName
	SetObject(f F)
}

func TaskContextObject[
	S scope.Object[F, T],
	F Object[O],
	T runtime.Object,
	O any,
](state ContextObjectNewer[F], c client.Client) task.Task {
	return task.NameTaskFunc("ContextObject", func(ctx context.Context) task.Result {
		key := state.Key()
		var obj F = new(O)
		if err := c.Get(ctx, key, obj); err != nil {
			if !errors.IsNotFound(err) {
				return task.Fail().With("can't get %s: %v", key, err)
			}

			return task.Complete().With("obj %s does not exist", key)
		}
		state.SetObject(obj)
		return task.Complete().With("object %s/%s(rv: %s) is set",
			obj.GetNamespace(),
			obj.GetName(),
			obj.GetResourceVersion(),
		)
	})
}

type ContextSliceNewer[
	GF client.Object,
	IF client.Object,
] interface {
	ObjectState[GF]
	SetInstanceSlice(f []IF)
}

// TaskContextSlice init instance slice in state by the group
func TaskContextSlice[
	S scope.GroupInstance[GF, GT, IS],
	IS scope.List[IL, I],
	GF client.Object,
	GT runtime.Group,
	IL client.ObjectList,
	I client.Object,
](state ContextSliceNewer[GF, I], c client.Client) task.Task {
	return task.NameTaskFunc("ContextSlice", func(ctx context.Context) task.Result {
		g := state.Object()
		objs, err := apicall.ListInstances[S](ctx, c, g)
		if err != nil {
			return task.Fail().With("cannot get instance slice: %v", err)
		}

		state.SetInstanceSlice(objs)
		return task.Complete().With("instance slice is set")
	})
}

// TaskContextPeerSlice init peer instance slice in state by the instance
func TaskContextPeerSlice[
	S scope.InstanceList[F, T, L],
	F client.Object,
	T runtime.Instance,
	L client.ObjectList,
](state ContextSliceNewer[F, F], c client.Client) task.Task {
	return task.NameTaskFunc("ContextPeerSlice", func(ctx context.Context) task.Result {
		in := state.Object()
		objs, err := apicall.ListPeerInstances[S](ctx, c, in)
		if err != nil {
			return task.Fail().With("cannot get peer instance slice: %v", err)
		}

		state.SetInstanceSlice(objs)
		return task.Complete().With("peer instance slice is set")
	})
}

type ContextClusterNewer[
	F client.Object,
] interface {
	ObjectState[F]
	SetCluster(c *v1alpha1.Cluster)
}

func TaskContextCluster[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](state ContextClusterNewer[F], c client.Client) task.Task {
	return task.NameTaskFunc("ContextCluster", func(ctx context.Context) task.Result {
		cluster, err := apicall.GetCluster[S](ctx, c, state.Object())
		if err != nil {
			return task.Fail().With("cannot get cluster: %v", err)
		}
		state.SetCluster(cluster)
		return task.Complete().With("cluster is set")
	})
}

type ContextPodNewer[
	F client.Object,
] interface {
	ObjectState[F]
	SetPod(pod *corev1.Pod)
}

func TaskContextPod[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](state ContextPodNewer[F], c client.Client) task.Task {
	return task.NameTaskFunc("ContextPod", func(ctx context.Context) task.Result {
		pod, err := apicall.GetPod[S](ctx, c, state.Object())
		if err != nil {
			if errors.IsNotFound(err) {
				return task.Complete().With("pod doesn't exist")
			}
			return task.Fail().With("cannot get pod: %v", err)
		}

		state.SetPod(pod)
		return task.Complete().With("pod is set")
	})
}

type ServerLabelsUpdater[T client.Object] interface {
	PodState
	ObjectState[T]
	HealthyState

	stateutil.IPDClient
}

func TaskServerLabels[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](
	state ServerLabelsUpdater[F],
	c client.Client,
	m pdm.PDClientManager,
	setLabelsFunc func(context.Context, map[string]string) error,
) task.Task {
	return task.NameTaskFunc("ServerLabels", func(ctx context.Context) task.Result {
		pdc, ok := state.GetPDClient(m)
		if !ok {
			return task.Wait().With("wait if pd client is not registered")
		}

		pod := state.Pod()
		if pod == nil || state.IsPodTerminating() {
			return task.Complete().With("skip sync server labels as the instance is not healthy")
		}

		// TODO: too many API calls to PD?
		pdCfg, err := getPDConfig(ctx, pdc.Underlay())
		if err != nil {
			return task.Fail().With(err.Error())
		}

		nodeLabels, err := getNodeLabels(ctx, c, pod, pdCfg.Replication.LocationLabels)
		if err != nil {
			return task.Fail().With(err.Error())
		}

		obj := state.Object()
		serverLabels := maputil.Merge(coreutil.ServerLabels[S](obj), nodeLabels)
		// TODO: is there any way to avoid unnecessary update?
		if err := setLabelsFunc(ctx, serverLabels); err != nil {
			if task.IsWaitError(err) {
				return task.Wait().With(err.Error())
			}
			return task.Fail().With("failed to set server labels %v: %s", serverLabels, err)
		}

		return task.Complete().With("server labels synced")
	})
}

// getNodeLabels retrieves node labels for the given pod
func getNodeLabels(ctx context.Context, c client.Client, pod *corev1.Pod, locationLabels []string) (map[string]string, error) {
	nodeName := pod.Spec.NodeName
	if nodeName == "" {
		return nil, fmt.Errorf("pod %s/%s has not been scheduled", pod.Namespace, pod.Name)
	}

	var node corev1.Node
	if err := c.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
		return nil, fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	return k8s.GetNodeLabelsForKeys(&node, locationLabels), nil
}

// getPDConfig retrieves PD configuration
func getPDConfig(ctx context.Context, pdClient pdapi.PDClient) (*pdapi.PDConfigFromAPI, error) {
	pdCfg, err := pdClient.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pd config: %w", err)
	}

	if pdCfg == nil || pdCfg.Replication == nil {
		return nil, fmt.Errorf("invalid pd config: replication config is nil")
	}

	return pdCfg, nil
}

type TrackState[F client.Object] interface {
	Key() types.NamespacedName
	ObjectState[F]
}

func TaskTrack[
	S scope.Instance[F, T],
	F Object[P],
	T runtime.Instance,
	P any,
](state TrackState[F], t tracker.Tracker) task.Task {
	return task.NameTaskFunc("Track", func(context.Context) task.Result {
		obj := state.Object()
		if obj == nil {
			key := state.Key()
			t.Untrack(key.Namespace, key.Name)
			return task.Complete().With("untrack deleted instance")
		}
		var group string
		owner := coreutil.OwnerGroup[S](obj)
		if owner != nil {
			group = owner.Name
		}
		t.Track(obj.GetNamespace(), obj.GetName(), group)
		return task.Complete().With("track instance %s with group %v", state.Key(), group)
	})
}

func TaskDeleteOfflinedStore[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](state ObjectState[F], c client.Client) task.Task {
	return task.NameTaskFunc("DeleteOfflinedStore", func(ctx context.Context) task.Result {
		obj := state.Object()
		if !obj.GetDeletionTimestamp().IsZero() {
			return task.Complete().With("store is deleting")
		}
		conds := coreutil.StatusConditions[S](state.Object())
		if !meta.IsStatusConditionTrue(conds, v1alpha1.StoreOfflinedConditionType) {
			return task.Complete().With("store is not offlined")
		}
		if err := c.Delete(ctx, obj); err != nil {
			return task.Fail().With("cannot delete store: %v", err)
		}

		return task.Wait().With("wait until store deletion is watched")
	})
}
