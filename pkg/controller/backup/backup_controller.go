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

package backup

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/backup"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap.com/v1alpha1"
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
)

// controllerKind contains the schema.GroupVersionKind for this controller type.
var controllerKind = v1alpha1.SchemeGroupVersion.WithKind("Backup")

// Controller controls backup.
type Controller struct {
	// kubernetes client interface
	kubeClient kubernetes.Interface
	// operator client interface
	cli versioned.Interface
	// control returns an interface capable of syncing a backup.
	// Abstracted out for testing.
	control ControlInterface
	// backupLister is able to list/get backup from a shared informer's store
	backupLister listers.BackupLister
	// backupListerSynced returns true if the backup shared informer has synced at least once
	backupListerSynced cache.InformerSynced
	// backups that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a backup controller.
func NewController(
	kubeCli kubernetes.Interface,
	cli versioned.Interface,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
) *Controller {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "backup"})

	backupInformer := informerFactory.Pingcap().V1alpha1().Backups()
	jobInformer := kubeInformerFactory.Batch().V1().Jobs()
	statusUpdater := controller.NewRealBackupConditionUpdater(cli, backupInformer.Lister(), recorder)
	jobControl := controller.NewRealJobControl(kubeCli, jobInformer.Lister(), recorder)

	bkc := &Controller{
		kubeClient: kubeCli,
		cli:        cli,
		control: NewDefaultBackupControl(
			statusUpdater,
			backup.NewBackupManager(
				backupInformer.Lister(),
				jobControl,
			),
			recorder,
		),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"backup",
		),
	}

	backupInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: bkc.enqueueBackup,
		UpdateFunc: func(old, cur interface{}) {
			bkc.enqueueBackup(cur)
		},
		DeleteFunc: bkc.enqueueBackup,
	})
	bkc.backupLister = backupInformer.Lister()
	bkc.backupListerSynced = backupInformer.Informer().HasSynced

	return bkc
}

// Run runs the backup controller.
func (bkc *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer bkc.queue.ShutDown()

	glog.Info("Starting backup controller")
	defer glog.Info("Shutting down backup controller")

	if !cache.WaitForCacheSync(stopCh, bkc.backupListerSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(bkc.worker, time.Second, stopCh)
	}

	<-stopCh
}

// worker runs a worker goroutine that invokes processNextWorkItem until the the controller's queue is closed
func (bkc *Controller) worker() {
	for bkc.processNextWorkItem() {
		// revive:disable:empty-block
	}
}

// processNextWorkItem dequeues items, processes them, and marks them done. It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (bkc *Controller) processNextWorkItem() bool {
	key, quit := bkc.queue.Get()
	if quit {
		return false
	}
	defer bkc.queue.Done(key)
	if err := bkc.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			glog.Infof("backup: %v, still need sync: %v, requeuing", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("backup: %v, sync failed %v, requeuing", key.(string), err))
		}
		bkc.queue.AddRateLimited(key)
	} else {
		bkc.queue.Forget(key)
	}
	return true
}

// sync syncs the given backup.
func (bkc *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		glog.V(4).Infof("Finished syncing Backup %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	backup, err := bkc.backupLister.Backups(ns).Get(name)
	if errors.IsNotFound(err) {
		glog.Infof("Backup has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return bkc.syncBackup(backup.DeepCopy())
}

func (bkc *Controller) syncBackup(tc *v1alpha1.Backup) error {
	return bkc.control.UpdateBackup(tc)
}

// enqueueBackup enqueues the given backup in the work queue.
func (bkc *Controller) enqueueBackup(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("cound't get key for object %+v: %v", obj, err))
		return
	}
	bkc.queue.Add(key)
}
