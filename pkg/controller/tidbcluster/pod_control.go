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

package tidbcluster

import (
	"context"
	"fmt"
	"sync"
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// PodController control pods of tidb cluster.
// see docs/design-proposals/2021-11-24-graceful-reschedule-tikv-pod.md
type PodController struct {
	deps  *controller.Dependencies
	queue workqueue.RateLimitingInterface

	podStatsMu sync.Mutex
	podStats   map[string]stat

	// only set in test
	testPDClient               pdapi.PDClient
	recheckLeaderCountDuration time.Duration
}

// NewPodController create a PodController.
func NewPodController(deps *controller.Dependencies) *PodController {
	c := &PodController{
		deps: deps,
		queue: workqueue.NewNamedRateLimitingQueue(
			controller.NewControllerRateLimiter(1*time.Second, 100*time.Second),
			"tidbcluster pods",
		),
		podStats:                   make(map[string]stat),
		recheckLeaderCountDuration: time.Second * 15,
	}

	podsInformer := deps.KubeInformerFactory.Core().V1().Pods()
	podsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueuePod,
		UpdateFunc: func(old, cur interface{}) {
			c.enqueuePod(cur)
		},
	})

	return c
}

type stat struct {
	observeAnnotationCounts int
	finishAnnotationCounts  int
}

func (c *PodController) getPodStat(pod *corev1.Pod) stat {
	c.podStatsMu.Lock()
	defer c.podStatsMu.Unlock()

	stat := c.podStats[pod.Namespace+pod.Name]
	return stat
}

func (c *PodController) setPodStat(pod *corev1.Pod, stat stat) {
	c.podStatsMu.Lock()
	defer c.podStatsMu.Unlock()

	c.podStats[pod.Namespace+pod.Name] = stat
}

// enqueueTidbCluster enqueues the given pod in the work queue.
func (c *PodController) enqueuePod(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Cound't get key for object %+v: %v", obj, err))
		return
	}
	c.queue.Add(key)
}

// Run the controller.
func (c *PodController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting tidbcluster pod controller")
	defer klog.Info("Shutting down tidbcluster pod controller")

	for i := 0; i < workers; i++ {
		go wait.Until(c.worker, time.Second, stopCh)
	}

	<-stopCh
}

// worker runs a worker goroutine that invokes processNextWorkItem until the the controller's queue is closed
func (c *PodController) worker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem dequeues items, processes them, and marks them done. It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (c *PodController) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	result, err := c.sync(key.(string))
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("TidbCluster pod: %v, sync failed %v, requeuing", key.(string), err))
		c.queue.AddRateLimited(key)
	} else {
		if result.RequeueAfter > 0 {
			c.queue.AddAfter(key, result.RequeueAfter)
		} else if result.Requeue {
			c.queue.AddRateLimited(key)
		} else {
			c.queue.Forget(key)
		}
	}
	return true
}

func (c *PodController) sync(key string) (reconcile.Result, error) {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return reconcile.Result{}, err
	}

	pod, err := c.deps.PodLister.Pods(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("Pod %v has been deleted", key)
		return reconcile.Result{}, nil
	}
	if err != nil {
		return reconcile.Result{}, err
	}
	pod = pod.DeepCopy()

	// labels of TiKV:
	// app.kubernetes.io/component=tikv,app.kubernetes.io/instance=db1369682775200135879,
	// app.kubernetes.io/managed-by=tidb-operator ...
	managedBy := pod.Labels[label.ManagedByLabelKey]
	if managedBy != label.TiDBOperator {
		return reconcile.Result{}, nil
	}

	tcName := pod.Labels[label.InstanceLabelKey]
	if tcName == "" {
		return reconcile.Result{}, nil
	}

	tc, err := c.deps.TiDBClusterLister.TidbClusters(ns).Get(tcName)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.V(4).Infof("TidbCluster %q is not found, skip sync the Pod %s", ns+"/"+tcName, name)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, perrors.Annotatef(err, "failed to get TidbCluster %q", ns+"/"+tcName)
	}
	tc = tc.DeepCopy()

	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing TidbCluster pod %q (%v)", key, time.Since(startTime))
	}()

	component := pod.Labels[label.ComponentLabelKey]
	ctx := context.Background()
	switch component {
	case label.TiKVLabelVal:
		return c.syncTiKVPod(ctx, pod, tc)
	default:
		return reconcile.Result{}, nil
	}
}

func (c *PodController) getPDClient(tc *v1alpha1.TidbCluster) pdapi.PDClient {
	if c.testPDClient != nil {
		return c.testPDClient
	}

	pdClient := controller.GetPDClient(c.deps.PDControl, tc)
	return pdClient
}

func (c *PodController) syncTiKVPod(ctx context.Context, pod *corev1.Pod, tc *v1alpha1.TidbCluster) (reconcile.Result, error) {
	key, value, ok := needEvictLeader(pod)

	if ok {
		switch value {
		case v1alpha1.EvictLeaderValueNone:
		case v1alpha1.EvictLeaderValueDeletePod:
		default:
			klog.Warningf("Ignore unknown value %q of annotation %q for Pod %s/%s", value, key, pod.Namespace, pod.Name)
			return reconcile.Result{}, nil
		}
	}

	if ok {
		evictStatus := &v1alpha1.EvictLeaderStatus{
			PodCreateTime: pod.CreationTimestamp,
			Value:         value,
		}
		nowStatus := tc.Status.TiKV.EvictLeader[pod.Name]
		if nowStatus == nil || *nowStatus != *evictStatus {
			if tc.Status.TiKV.EvictLeader == nil {
				tc.Status.TiKV.EvictLeader = make(map[string]*v1alpha1.EvictLeaderStatus)
			}
			tc.Status.TiKV.EvictLeader[pod.Name] = evictStatus
			var err error
			key := fmt.Sprintf("%s/%s", tc.Namespace, tc.Name)
			tc, err = c.deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Update(ctx, tc, metav1.UpdateOptions{})
			if err != nil {
				return reconcile.Result{}, perrors.Annotatef(err, "failed to update tc %q status", key)
			}

			stat := c.getPodStat(pod)
			stat.observeAnnotationCounts++
			c.setPodStat(pod, stat)
		}

		pdClient := c.getPDClient(tc)
		storeID, err := member.TiKVStoreIDFromStatus(tc, pod.Name)
		if err != nil {
			return reconcile.Result{}, perrors.Annotatef(err, "failed to get tikv store id from status for pod %s/%s", pod.Namespace, pod.Name)
		}
		err = pdClient.BeginEvictLeader(storeID)
		if err != nil {
			return reconcile.Result{}, perrors.Annotatef(err, "failed to evict leader for store %d (Pod %s/%s)", storeID, pod.Namespace, pod.Name)
		}

		if value == v1alpha1.EvictLeaderValueDeletePod {
			tlsEnabled := tc.IsTLSClusterEnabled()
			kvClient := c.deps.TiKVControl.GetTiKVPodClient(tc.Namespace, tc.Name, pod.Name, tlsEnabled)
			leaderCount, err := kvClient.GetLeaderCount()
			if err != nil {
				return reconcile.Result{}, perrors.Annotatef(err, "failed to get leader count for pod %s/%s", pod.Namespace, pod.Name)
			}

			klog.Infof("Region leader count is %d for Pod %s/%s", leaderCount, pod.Namespace, pod.Name)

			if leaderCount == 0 {
				err = c.deps.KubeClientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
				if err != nil && !errors.IsNotFound(err) {
					return reconcile.Result{}, perrors.Annotatef(err, "failed to delete pod %q", pod.Name)
				}
			} else {
				// re-check leader count next time
				return reconcile.Result{RequeueAfter: c.recheckLeaderCountDuration}, nil
			}
		}
	} else {
		// 1. delete evict-leader scheduler
		// 2. delete pod from tc.Status.TiKV.EvictLeader and update it to api-server
		endEvict := func() error {
			pdClient := c.getPDClient(tc)
			storeID, err := member.TiKVStoreIDFromStatus(tc, pod.Name)
			if err != nil {
				return perrors.Annotatef(err, "failed to get tikv store id from status for pod %s/%s", pod.Namespace, pod.Name)
			}

			err = pdClient.EndEvictLeader(storeID)
			if err != nil {
				return perrors.Annotatef(err, "failed to remove evict leader scheduler for store %d, pod %s/%s", storeID, pod.Namespace, pod.Name)
			}

			err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				delete(tc.Status.TiKV.EvictLeader, pod.Name)
				_, updateErr := c.deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Update(ctx, tc, metav1.UpdateOptions{})
				if updateErr == nil {
					return nil
				}

				if updated, err := c.deps.TiDBClusterLister.TidbClusters(tc.Namespace).Get(tc.Name); err == nil {
					// // make a copy so we don't mutate the shared cache
					tc = updated.DeepCopy()
				} else {
					utilruntime.HandleError(fmt.Errorf("error getting updated tc %s/%s from lister: %v", tc.Namespace, tc.Name, err))
				}

				return updateErr
			})
			if err != nil {
				return perrors.Annotatef(err, "failed to update status for tc %s/%s", tc.Namespace, tc.Name)
			}

			return nil
		}

		evictStatus := tc.Status.TiKV.EvictLeader[pod.Name]
		if evictStatus != nil {
			if evictStatus.Value == v1alpha1.EvictLeaderValueDeletePod {
				if podutil.IsPodReady(pod) {
					err := endEvict()
					if err != nil {
						return reconcile.Result{}, err
					}
				}
			} else if evictStatus.Value == v1alpha1.EvictLeaderValueNone {
				err := endEvict()
				if err != nil {
					return reconcile.Result{}, err
				}
			}
			stat := c.getPodStat(pod)
			stat.finishAnnotationCounts++
			c.setPodStat(pod, stat)
		}
	}

	return reconcile.Result{}, nil
}

func needEvictLeader(pod *corev1.Pod) (string, string, bool) {
	for _, key := range v1alpha1.EvictLeaderAnnKeys {
		value, exist := pod.Annotations[key]
		if exist {
			return key, value, true
		}
	}

	return "", "", false
}
