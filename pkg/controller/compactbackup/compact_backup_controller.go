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
// See the License for the specific language governing permissions and
// limitations under the License.

package compact

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pingcap/errors"
	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	backuputil "github.com/pingcap/tidb-operator/pkg/backup/util"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/metrics"
	"github.com/pingcap/tidb-operator/pkg/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"k8s.io/utils/ptr"
)

// Controller controls backup.
type Controller struct {
	deps *controller.Dependencies
	// backups that need to be synced.
	queue workqueue.RateLimitingInterface
	cli   versioned.Interface
	statusUpdater controller.CompactStatusUpdaterInterface
}

// NewController creates a backup controller.
func NewController(deps *controller.Dependencies) *Controller {
	c := &Controller{
		deps: deps,
		queue: workqueue.NewNamedRateLimitingQueue(
			controller.NewControllerRateLimiter(1*time.Second, 100*time.Second),
			"compactBackup",
		),
		cli: deps.Clientset,
	}

	backupInformer := deps.InformerFactory.Pingcap().V1alpha1().CompactBackups()
	jobInformer := deps.KubeInformerFactory.Batch().V1().Jobs()
	backupInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.updateCompact,
		UpdateFunc: func(old, cur interface{}) {
			c.updateCompact(cur)
		},
		DeleteFunc: c.updateCompact,
	})
	jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteJob,
	})

	return c
}

// Name returns backup controller name.
func (c *Controller) Name() string {
	return "compactBackup"
}

// Run runs the backup controller.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting compact backup controller")
	defer klog.Info("Shutting down compact backup controller")

	for i := 0; i < workers; i++ {
		go wait.Until(c.worker, time.Second, stopCh)
	}

	<-stopCh
}

// worker runs a worker goroutine that invokes processNextWorkItem until the the controller's queue is closed
func (c *Controller) worker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) UpdateStatus(backup *v1alpha1.CompactBackup, newState string, message ...string) error {
	ns := backup.GetNamespace()
	backupName := backup.GetName()
	// try best effort to guarantee backup is updated.
	err := retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		// Always get the latest backup before update.
		if updated, err := c.deps.CompactBackupLister.CompactBackups(ns).Get(backupName); err == nil {
			// make a copy so we don't mutate the shared cache
			*backup = *(updated.DeepCopy())
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated backup %s/%s from lister: %v", ns, backupName, err))
			return err
		}
		if backup.Status.State != newState {
			backup.Status.State = newState
			if len(message) > 0 {
				backup.Status.Message = message[0]
			}
			_, updateErr := c.cli.PingcapV1alpha1().CompactBackups(ns).Update(context.TODO(), backup, metav1.UpdateOptions{})
			if updateErr == nil {
				klog.Infof("Backup: [%s/%s] updated successfully", ns, backupName)
				return nil
			}
			klog.Errorf("Failed to update backup [%s/%s], error: %v", ns, backupName, updateErr)
			return updateErr
		}
		return nil
	})
	return err
}

func (c *Controller) resolveCompactBackupFromJob(namespace string, job *batchv1.Job) *v1alpha1.CompactBackup {
	owner := metav1.GetControllerOf(job)
	if owner == nil {
		return nil
	}

	if owner.Kind != controller.CompactBackupControllerKind.Kind {
		return nil
	}

	backup, err := c.deps.CompactBackupLister.CompactBackups(namespace).Get(owner.Name)
	if err != nil {
		return nil
	}
	if owner.UID != backup.UID {
		return nil
	}
	return backup
}

func (c *Controller) deleteJob(obj interface{}) {
	job, ok := obj.(*batchv1.Job)
	if !ok {
		return
	}

	ns := job.GetNamespace()
	jobName := job.GetName()
	backup := c.resolveCompactBackupFromJob(ns, job)
	if backup == nil {
		return
	}
	klog.V(4).Infof("Job %s/%s deleted through %v.", ns, jobName, utilruntime.GetCaller())
	c.updateCompact(backup)
}

func (c *Controller) updateCompact(cur interface{}) {
	newBackup := cur.(*v1alpha1.CompactBackup)
	ns := newBackup.GetNamespace()
	name := newBackup.GetName()

	if newBackup.Status.State == string(v1alpha1.BackupFailed){
		klog.Errorf("Backup %s/%s is failed, skip", ns, name)
		return
	}

	if newBackup.Status.State == string(v1alpha1.BackupComplete){
		klog.Errorf("Backup %s/%s is complete, skip", ns, name)
		return
	}

	klog.Infof("backup object %s/%s enqueue", ns, name)
	c.enqueueBackup(newBackup)
}

// enqueueBackup enqueues the given backup in the work queue.
func (c *Controller) enqueueBackup(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("cound't get key for object %+v: %v", obj, err))
		return
	}
	c.queue.Add(key)
}

// processNextWorkItem dequeues items, processes them, and marks them done. It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (c *Controller) processNextWorkItem() bool {
	metrics.ActiveWorkers.WithLabelValues(c.Name()).Add(1)
	defer metrics.ActiveWorkers.WithLabelValues(c.Name()).Add(-1)

	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	if err := c.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("Backup: %v, still need sync: %v, requeuing", key.(string), err)
			c.queue.AddRateLimited(key)
		} else if perrors.Find(err, controller.IsIgnoreError) != nil {
			klog.Infof("Backup: %v, ignore err: %v", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("Backup: %v, sync failed, err: %v, requeuing", key.(string), err))
			c.queue.AddRateLimited(key)
		}
	} else {
		c.queue.Forget(key)
	}
	return true
}

func (c *Controller) sync(key string) (err error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.ReconcileTime.WithLabelValues(c.Name()).Observe(duration.Seconds())

		if err == nil {
			metrics.ReconcileTotal.WithLabelValues(c.Name(), metrics.LabelSuccess).Inc()
		} else if perrors.Find(err, controller.IsRequeueError) != nil {
			metrics.ReconcileTotal.WithLabelValues(c.Name(), metrics.LabelRequeue).Inc()
		} else {
			metrics.ReconcileTotal.WithLabelValues(c.Name(), metrics.LabelError).Inc()
			metrics.ReconcileErrors.WithLabelValues(c.Name()).Inc()
		}

		klog.V(4).Infof("Finished syncing Backup %q (%v)", key, duration)
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	klog.Infof("Backup: [%s/%s] start to sync", ns, name)
	compact, err := c.deps.CompactBackupLister.CompactBackups(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("Backup has been deleted %v", key)
			return nil
		}
		klog.Infof("Backup get failed %v", err)
		return err
	}

	running,err := c.isCompactJobAlreadyRunning(compact) 
	if err != nil {
		return err
	}
	if running {
		klog.Infof("Backup %s/%s is running, skip", ns, name)
		return nil
	}

	c.statusUpdater.OnSchedule(context.TODO(), compact)

	err = c.doCompact(compact.DeepCopy())
	klog.Errorf("Backup: [%s/%s] sync failed, error: %v", ns, name, err)

	c.statusUpdater.OnCreateJob(context.TODO(), compact, err)
	return err
}

func (c *Controller) doCompact(compact *v1alpha1.CompactBackup) error {
	ns := compact.GetNamespace()
	name := compact.GetName()
	backupJobName := compact.GetName()

	// make backup job
	var err error
	var job *batchv1.Job
	var reason string
	if job, reason, err = c.makeCompactJob(compact); err != nil {
		klog.Errorf("backup %s/%s create job %s failed, reason is %s, error %v.", ns, name, backupJobName, reason, err)
		return err
	}

	// create k8s job
	klog.Infof("backup %s/%s creating job %s.", ns, name, backupJobName)
	if err := c.deps.JobControl.CreateJob(compact, job); err != nil {
		errMsg := fmt.Errorf("create backup %s/%s job %s failed, err: %v", ns, name, backupJobName, err)
		return errMsg
	}
	return nil
}

func (c *Controller) makeCompactJob(compact *v1alpha1.CompactBackup) (*batchv1.Job, string, error) {
	ns := compact.GetNamespace()
	name := compact.GetName()
	// Do we need a unique name for the job?
	jobName := compact.GetName()

	var (
		envVars []corev1.EnvVar
		reason  string
		err     error
	)

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, compact.Spec.UseKMS, compact.Spec.StorageProvider, c.deps.SecretLister)
	if err != nil {
		return nil, reason, fmt.Errorf("backup %s/%s, %v", ns, name, err)
	}

	envVars = append(envVars, storageEnv...)

	// set env vars specified in backup.Spec.Env
	envVars = util.AppendOverwriteEnv(envVars, compact.Spec.Env)

	args := []string{
		"compact",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--resourceName=%s", name),
	}

	tc, err := c.deps.TiDBClusterLister.TidbClusters(compact.Spec.BR.ClusterNamespace).Get(compact.Spec.BR.Cluster)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch tidbcluster %s/%s", ns, compact.Spec.BR.Cluster), err
	}
	tikvImage := tc.TiKVImage()
	_, tikvVersion := backuputil.ParseImage(tikvImage)
	brImage := "pingcap/br:" + tikvVersion
	if compact.Spec.ToolImage != "" {
		toolImage := compact.Spec.ToolImage
		if !strings.ContainsRune(compact.Spec.ToolImage, ':') {
			toolImage = fmt.Sprintf("%s:%s", toolImage, tikvVersion)
		}

		brImage = toolImage
	}
	klog.Infof("backup %s/%s use br image %s and tikv image %s", ns, name, brImage, tikvImage)

	//TODO: (Ris)What is the instance here?
	jobLabels := util.CombineStringMap(label.NewBackup().Instance("Compact-Backup").BackupJob().Backup(name), compact.Labels)
	podLabels := jobLabels
	jobAnnotations := compact.Annotations
	podAnnotations := jobAnnotations

	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}

	volumes = append(volumes, corev1.Volume{
		Name: "tool-bin",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	volumeMounts = append(volumeMounts,
		corev1.VolumeMount{
			Name:      "tool-bin",
			MountPath: util.BRBinPath,
		},
		corev1.VolumeMount{
			Name:      "tool-bin",
			MountPath: util.KVCTLBinPath,
		},
	)

	if len(compact.Spec.AdditionalVolumes) > 0 {
		volumes = append(volumes, compact.Spec.AdditionalVolumes...)
	}
	if len(compact.Spec.AdditionalVolumeMounts) > 0 {
		volumeMounts = append(volumeMounts, compact.Spec.AdditionalVolumeMounts...)
	}

	// mount volumes if specified
	if compact.Spec.Local != nil {
		volumes = append(volumes, compact.Spec.Local.Volume)
		volumeMounts = append(volumeMounts, compact.Spec.Local.VolumeMount)
	}

	serviceAccount := constants.DefaultServiceAccountName
	if compact.Spec.ServiceAccount != "" {
		serviceAccount = compact.Spec.ServiceAccount
	}

	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      podLabels,
			Annotations: podAnnotations,
		},
		Spec: corev1.PodSpec{
			SecurityContext:    compact.Spec.PodSecurityContext,
			ServiceAccountName: serviceAccount,
			InitContainers: []corev1.Container{
				{
					Name:            "br",
					Image:           brImage,
					Command:         []string{"/bin/sh", "-c"},
					Args:            []string{fmt.Sprintf("cp /br %s/br; echo 'BR copy finished'", util.BRBinPath)},
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts:    volumeMounts,
					Resources:       compact.Spec.ResourceRequirements,
				},
				{
					Name:            "tikv-ctl",
					Image:           tikvImage,
					Command:         []string{"/bin/sh", "-c"},
					Args:            []string{fmt.Sprintf("cp /tikv-ctl %s/tikv-ctl; echo 'tikv-ctl copy finished'", util.KVCTLBinPath)},
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts:    volumeMounts,
					Resources:       compact.Spec.ResourceRequirements,
				},
			},
			Containers: []corev1.Container{
				{
					Name:            "backup-manager",
					Image:           c.deps.CLIConfig.TiDBBackupManagerImage,
					Args:            args,
					Env:             envVars,
					VolumeMounts:    volumeMounts,
					ImagePullPolicy: corev1.PullIfNotPresent,
				},
			},
			RestartPolicy:     corev1.RestartPolicyOnFailure,
			Tolerations:       compact.Spec.Tolerations,
			ImagePullSecrets:  compact.Spec.ImagePullSecrets,
			Affinity:          compact.Spec.Affinity,
			Volumes:           volumes,
			PriorityClassName: compact.Spec.PriorityClassName,
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   ns,
			Labels:      jobLabels,
			Annotations: jobAnnotations,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetCompactBackupOwnerRef(compact),
			},
		},
		Spec: batchv1.JobSpec{
			Template:     *podSpec,
			BackoffLimit: ptr.To(compact.Spec.MaxRetryTimes),
		},
	}

	return job, "", nil
}

func (c *Controller) isCompactJobAlreadyRunning(compact *v1alpha1.CompactBackup) (bool, error) {
	ns := compact.GetNamespace()
	name := compact.GetName()

	job, err := c.deps.KubeClientset.BatchV1().Jobs(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		klog.Errorf("Failed to get job %s for compact %s/%s, error %v", name, ns, name, err)
		return false, err
	}

	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			// Check events if job failed
			events, err := c.deps.KubeClientset.CoreV1().Events(ns).List(context.TODO(), metav1.ListOptions{
				FieldSelector: fmt.Sprintf("involvedObject.kind=Job,involvedObject.name=%s", name),
			})
			if err != nil {
				if errors.IsNotFound(err) {
					return true, nil // No events found, treat as "running"
				}
				klog.Errorf("Failed to get events for job %s/%s, error %v", ns, name, err)
				return true, err
			}

			for _, event := range events.Items {
				if event.Reason == "BackoffLimitExceeded" {
					klog.Warningf("Job %s has exceeded the backoff limit, no further retries will be attempted.", name)
					c.statusUpdater.OnJobFailed(context.TODO(), compact, event.Message)
				}
			}
			return true, nil
		}
	}

	return true, nil
}
