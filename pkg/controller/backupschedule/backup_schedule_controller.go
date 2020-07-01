// Copyright 2019 PingCAP, Inc.
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

package backupschedule

import (
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/backupschedule"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

// Controller controls restore.
type Controller struct {
	// kubernetes client interface
	kubeClient kubernetes.Interface
	// operator client interface
	cli versioned.Interface
	// control returns an interface capable of syncing a restore.
	// Abstracted out for testing.
	control ControlInterface
	// bsLister is able to list/get restore from a shared informer's store
	bsLister listers.BackupScheduleLister
	// bsListerSynced returns true if the restore shared informer has synced at least once
	bsListerSynced cache.InformerSynced
	// backupSchedules that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a backup schedule controller.
func NewController(
	kubeCli kubernetes.Interface,
	cli versioned.Interface,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
) *Controller {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "backupSchedule"})

	bsInformer := informerFactory.Pingcap().V1alpha1().BackupSchedules()
	backupInformer := informerFactory.Pingcap().V1alpha1().Backups()
	jobInformer := kubeInformerFactory.Batch().V1().Jobs()
	backupControl := controller.NewRealBackupControl(cli, recorder)
	statusUpdater := controller.NewRealBackupScheduleStatusUpdater(cli, bsInformer.Lister(), recorder)
	jobControl := controller.NewRealJobControl(kubeCli, recorder)

	bsc := &Controller{
		kubeClient: kubeCli,
		cli:        cli,
		control: NewDefaultBackupScheduleControl(
			statusUpdater,
			backupschedule.NewBackupScheduleManager(
				backupInformer.Lister(),
				backupControl,
				jobInformer.Lister(),
				jobControl,
			),
			recorder,
		),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"backupSchedule",
		),
	}

	bsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: bsc.enqueueBackupSchedule,
		UpdateFunc: func(old, cur interface{}) {
			bsc.enqueueBackupSchedule(cur)
		},
		DeleteFunc: bsc.enqueueBackupSchedule,
	})
	bsc.bsLister = bsInformer.Lister()
	bsc.bsListerSynced = bsInformer.Informer().HasSynced

	return bsc
}

// Run runs the backup schedule controller.
func (bsc *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer bsc.queue.ShutDown()

	klog.Info("Starting backup schedule controller")
	defer klog.Info("Shutting down backup schedule controller")

	for i := 0; i < workers; i++ {
		go wait.Until(bsc.worker, time.Second, stopCh)
	}

	<-stopCh
}

// worker runs a worker goroutine that invokes processNextWorkItem until the the controller's queue is closed
func (bsc *Controller) worker() {
	for bsc.processNextWorkItem() {
		// revive:disable:empty-block
	}
}

// processNextWorkItem dequeues items, processes them, and marks them done. It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (bsc *Controller) processNextWorkItem() bool {
	key, quit := bsc.queue.Get()
	if quit {
		return false
	}
	defer bsc.queue.Done(key)
	if err := bsc.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("BackupSchedule: %v, still need sync: %v, requeuing", key.(string), err)
			bsc.queue.AddRateLimited(key)
		} else if perrors.Find(err, controller.IsIgnoreError) != nil {
			klog.V(4).Infof("BackupSchedule: %v, ignore err: %v, waiting for the next sync", key.(string), err)
		} else {
			utilruntime.HandleError(perrors.Errorf("BackupSchedule: %v, sync failed, err: %v, requeuing", key.(string), err))
			bsc.queue.AddRateLimited(key)
		}
	} else {
		bsc.queue.Forget(key)
	}
	return true
}

// sync syncs the given backupSchedule.
func (bsc *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing BackupSchedule %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	bs, err := bsc.bsLister.BackupSchedules(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("BackupSchedule has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return bsc.syncBackupSchedule(bs.DeepCopy())
}

func (bsc *Controller) syncBackupSchedule(bs *v1alpha1.BackupSchedule) error {
	return bsc.control.UpdateBackupSchedule(bs)
}

// enqueueBackupSchedule enqueues the given restore in the work queue.
func (bsc *Controller) enqueueBackupSchedule(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(perrors.Errorf("cound't get key for object %+v: %v", obj, err))
		return
	}
	bsc.queue.Add(key)
}
