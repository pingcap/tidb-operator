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
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type podInfo struct {
	name         string
	uid          string
	creationTime metav1.Time
	deletionTime metav1.Time
}

func WaitPodsRollingUpdateOnce[G runtime.Group](
	ctx context.Context,
	c client.Client,
	g G,
	// scale means scale out/in before/after rolling update
	// k means scale out k instances and -k means scale in k instances
	// 0 means only rolling update
	scale int,
	timeout time.Duration,
) error {
	ctx, cancel := watchtools.ContextWithOptionalTimeout(ctx, timeout)
	defer cancel()

	podMap, err := generatePodInfoMapByWatch(ctx, c, g)
	if err != nil {
		return err
	}

	infos := []podInfo{}
	for _, v := range podMap {
		infos = append(infos, v)
	}
	sortPodInfos(infos)
	detail := strings.Builder{}
	for _, info := range infos {
		if info.deletionTime.IsZero() {
			detail.WriteString(fmt.Sprintf("%v(%v) created at %s\n", info.name, info.uid, info.creationTime))
		} else {
			detail.WriteString(fmt.Sprintf("%v(%v) created at %s, deleted at %s\n", info.name, info.uid, info.creationTime, info.deletionTime))
		}
	}

	rollingUpdateTimes := int(g.Replicas())

	if scale > 0 {
		for i := range scale {
			if !infos[i].deletionTime.IsZero() {
				return fmt.Errorf("expect scale out %v pods before rolling update, detail:\n%v", scale, detail.String())
			}
		}

		infos = infos[scale:]
		rollingUpdateTimes -= scale
	}
	if scale < 0 {
		for i := range -scale {
			if infos[len(infos)-1-i].deletionTime.IsZero() {
				return fmt.Errorf("expect scale in %v pods after rolling update, detail:\n%v", scale, detail.String())
			}
		}
		infos = infos[:len(infos)+scale]
	}

	if len(infos) != 2*rollingUpdateTimes {
		return fmt.Errorf("expect %v pods info, now only %v, detail:\n%v", 2*rollingUpdateTimes, len(infos), detail.String())
	}
	for i := range rollingUpdateTimes {
		if infos[2*i].name != infos[2*i+1].name {
			return fmt.Errorf("pod may be restarted at same time, detail:\n%v", detail.String())
		}
	}

	return nil
}

func generatePodInfoMapByWatch[G runtime.Group](ctx context.Context, c client.Client, g G) (map[string]podInfo, error) {
	podMap := map[string]podInfo{}
	lw := newListWatch(ctx, c, g)
	_, err := watchtools.UntilWithSync(ctx, lw, &corev1.Pod{}, nil, func(event watch.Event) (bool, error) {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			// ignore events without pod
			return false, nil
		}

		info, ok := podMap[string(pod.UID)]
		if !ok {
			info = podInfo{
				name:         pod.Name,
				uid:          string(pod.UID),
				creationTime: pod.CreationTimestamp,
			}
		}

		if !pod.DeletionTimestamp.IsZero() && pod.DeletionGracePeriodSeconds != nil && *pod.DeletionGracePeriodSeconds == 0 {
			info.deletionTime = *pod.DeletionTimestamp
		}
		podMap[string(pod.UID)] = info

		return false, nil
	})

	if !wait.Interrupted(err) {
		return nil, fmt.Errorf("watch stopped unexpected: %w", err)
	}

	return podMap, nil
}

func newListWatch[G runtime.Group](ctx context.Context, c client.Client, g G) cache.ListerWatcher {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (kuberuntime.Object, error) {
			list := &corev1.PodList{}
			if err := c.List(ctx, list, &client.ListOptions{
				Namespace: g.GetNamespace(),
				LabelSelector: labels.SelectorFromSet(labels.Set{
					v1alpha1.LabelKeyCluster:   g.Cluster(),
					v1alpha1.LabelKeyGroup:     g.GetName(),
					v1alpha1.LabelKeyComponent: g.Component(),
				}),
				Raw: &options,
			}); err != nil {
				return nil, err
			}
			return list, nil
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			list := &corev1.PodList{}
			return c.Watch(ctx, list, &client.ListOptions{
				Namespace: g.GetNamespace(),
				LabelSelector: labels.SelectorFromSet(labels.Set{
					v1alpha1.LabelKeyCluster:   g.Cluster(),
					v1alpha1.LabelKeyGroup:     g.GetName(),
					v1alpha1.LabelKeyComponent: g.Component(),
				}),
				Raw: &options,
			})
		},
	}

	return lw
}

func sortPodInfos(infos []podInfo) {
	slices.SortFunc(infos, func(a podInfo, b podInfo) int {
		if a.deletionTime.IsZero() && b.deletionTime.IsZero() {
			return a.creationTime.Compare(b.creationTime.Time)
		}
		if a.deletionTime.IsZero() {
			return a.creationTime.Compare(b.deletionTime.Time)
		}
		if b.deletionTime.IsZero() {
			return a.deletionTime.Compare(b.creationTime.Time)
		}
		return a.deletionTime.Compare(b.deletionTime.Time)
	})
}

func WaitForPodsReady[G runtime.Group](ctx context.Context, c client.Client, g G, timeout time.Duration) error {
	list := corev1.PodList{}
	return WaitForList(ctx, c, &list, func() error {
		if len(list.Items) != int(g.Replicas()) {
			return fmt.Errorf("%s/%s pod replicas %d not equal to %d", g.GetNamespace(), g.GetName(), len(list.Items), g.Replicas())
		}
		for i := range list.Items {
			pod := &list.Items[i]
			if pod.Status.Phase != corev1.PodRunning {
				return fmt.Errorf("%s/%s pod %s is not running, current phase: %s", g.GetNamespace(), g.GetName(), pod.Name, pod.Status.Phase)
			}
			for i := range pod.Status.Conditions {
				cond := &pod.Status.Conditions[i]
				if cond.Type != corev1.PodReady {
					continue
				}
				if cond.Status != corev1.ConditionTrue {
					return fmt.Errorf("%s/%s pod %s is not ready, current status: %s, reason: %v, message: %v",
						g.GetNamespace(),
						g.GetName(),
						pod.Name,
						cond.Status,
						cond.Reason,
						cond.Message,
					)
				}
			}
		}

		return nil
	}, timeout, client.InNamespace(g.GetNamespace()), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   g.Cluster(),
		v1alpha1.LabelKeyGroup:     g.GetName(),
		v1alpha1.LabelKeyComponent: g.Component(),
	})
}

func WaitForPodsRecreated[G runtime.Group](
	ctx context.Context,
	c client.Client,
	g G,
	changeTime time.Time,
	timeout time.Duration,
) error {
	list := corev1.PodList{}
	return WaitForList(ctx, c, &list, func() error {
		if len(list.Items) != int(g.Replicas()) {
			return fmt.Errorf("%s/%s replicas %d not equal to %d", g.GetNamespace(), g.GetName(), len(list.Items), g.Replicas())
		}
		for i := range list.Items {
			pod := &list.Items[i]
			if pod.CreationTimestamp.Time.Before(changeTime) {
				return fmt.Errorf("pod %s/%s is created at %v before change time %v", pod.Namespace, pod.Name, pod.CreationTimestamp, changeTime)
			}
		}

		return nil
	}, timeout, client.InNamespace(g.GetNamespace()), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   g.Cluster(),
		v1alpha1.LabelKeyGroup:     g.GetName(),
		v1alpha1.LabelKeyComponent: g.Component(),
	})
}

func MaxPodsCreateTimestamp[G runtime.Group](
	ctx context.Context,
	c client.Client,
	g G,
) (*time.Time, error) {
	list := corev1.PodList{}
	if err := c.List(ctx, &list, client.InNamespace(g.GetNamespace()), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   g.Cluster(),
		v1alpha1.LabelKeyGroup:     g.GetName(),
		v1alpha1.LabelKeyComponent: g.Component(),
	}); err != nil {
		return nil, err
	}
	maxTime := &time.Time{}
	for i := range list.Items {
		pod := &list.Items[i]
		if pod.CreationTimestamp.After(*maxTime) {
			maxTime = &pod.CreationTimestamp.Time
		}
	}

	return maxTime, nil
}
