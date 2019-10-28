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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/backup"
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
)

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
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	secretInformer := kubeInformerFactory.Core().V1().Secrets()
	statusUpdater := controller.NewRealBackupConditionUpdater(cli, backupInformer.Lister(), recorder)
	jobControl := controller.NewRealJobControl(kubeCli, recorder)
	pvcControl := controller.NewRealGeneralPVCControl(kubeCli, recorder)
	backupCleaner := backup.NewBackupCleaner(statusUpdater, secretInformer.Lister(), jobInformer.Lister(), jobControl)

	bkc := &Controller{
		kubeClient: kubeCli,
		cli:        cli,
		control: NewDefaultBackupControl(
			cli,
			backup.NewBackupManager(
				backupCleaner,
				statusUpdater,
				secretInformer.Lister(),
				jobInformer.Lister(),
				jobControl,
				pvcInformer.Lister(),
				pvcControl,
			),
		),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"backup",
		),
	}

	backupInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: bkc.updateBackup,
		UpdateFunc: func(old, cur interface{}) {
			bkc.updateBackup(cur)
		},
		DeleteFunc: bkc.updateBackup,
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
			glog.Infof("Backup: %v, still need sync: %v, requeuing", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("Backup: %v, sync failed, err: %v, requeuing", key.(string), err))
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

func (bkc *Controller) syncBackup(backup *v1alpha1.Backup) error {
	return bkc.control.UpdateBackup(backup)
}

func (bkc *Controller) updateBackup(cur interface{}) {
	newBackup := cur.(*v1alpha1.Backup)
	ns := newBackup.GetNamespace()
	name := newBackup.GetName()

	if newBackup.DeletionTimestamp != nil {
		// the backup is being deleted, we need to do some cleanup work, enqueue backup.
		glog.Infof("backup %s/%s is being deleted", ns, name)
		bkc.enqueueBackup(newBackup)
		return
	}

	if v1alpha1.IsBackupComplete(newBackup) {
		glog.V(4).Infof("backup %s/%s is Complete, skipping.", ns, name)
		return
	}

	if v1alpha1.IsBackupScheduled(newBackup) {
		glog.V(4).Infof("backup %s/%s is already scheduled, skipping", ns, name)
		return
	}

	glog.V(4).Infof("backup object %s/%s enqueue", ns, name)
	bkc.enqueueBackup(newBackup)
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
