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

package controller

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/label"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	batchinformers "k8s.io/client-go/informers/batch/v1"
	"k8s.io/client-go/kubernetes"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

// JobControlInterface manages Jobs used in backup、restore and clean
type JobControlInterface interface {
	CreateJob(object runtime.Object, job *batchv1.Job) error
	DeleteJob(object runtime.Object, job *batchv1.Job) error
}

type realJobControl struct {
	kubeCli  kubernetes.Interface
	recorder record.EventRecorder
}

// NewRealJobControl creates a new JobControlInterface
func NewRealJobControl(
	kubeCli kubernetes.Interface,
	recorder record.EventRecorder,
) JobControlInterface {
	return &realJobControl{
		kubeCli:  kubeCli,
		recorder: recorder,
	}
}

func (rjc *realJobControl) CreateJob(object runtime.Object, job *batchv1.Job) error {
	ns := job.GetNamespace()
	jobName := job.GetName()
	instanceName := job.GetLabels()[label.InstanceLabelKey]
	kind := object.GetObjectKind().GroupVersionKind().Kind

	_, err := rjc.kubeCli.BatchV1().Jobs(ns).Create(job)
	if err != nil {
		klog.Errorf("failed to create %s job: [%s/%s], cluster: %s, err: %v", strings.ToLower(kind), ns, jobName, instanceName, err)
	} else {
		klog.V(4).Infof("create %s job: [%s/%s] successfully, cluster: %s", strings.ToLower(kind), ns, jobName, instanceName)
	}
	rjc.recordJobEvent("create", object, job, err)
	return err
}

func (rjc *realJobControl) DeleteJob(object runtime.Object, job *batchv1.Job) error {
	ns := job.GetNamespace()
	jobName := job.GetName()
	instanceName := job.GetLabels()[label.InstanceLabelKey]
	kind := object.GetObjectKind().GroupVersionKind().Kind

	propForeground := metav1.DeletePropagationForeground
	opts := &metav1.DeleteOptions{
		PropagationPolicy: &propForeground,
	}
	err := rjc.kubeCli.BatchV1().Jobs(ns).Delete(jobName, opts)
	if err != nil {
		klog.Errorf("failed to delete %s job: [%s/%s], cluster: %s, err: %v", strings.ToLower(kind), ns, jobName, instanceName, err)
	} else {
		klog.V(4).Infof("delete %s job: [%s/%s] successfully, cluster: %s", strings.ToLower(kind), ns, jobName, instanceName)
	}
	rjc.recordJobEvent("delete", object, job, err)
	return err
}

func (rjc *realJobControl) recordJobEvent(verb string, obj runtime.Object, job *batchv1.Job, err error) {
	jobName := job.GetName()
	ns := job.GetNamespace()
	instanceName := job.GetLabels()[label.InstanceLabelKey]
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s job %s/%s for cluster %s %s successful",
			strings.ToLower(verb), ns, jobName, instanceName, strings.ToLower(kind))
		rjc.recorder.Event(obj, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s job %s/%s for cluster %s %s failed error: %s",
			strings.ToLower(verb), ns, jobName, instanceName, strings.ToLower(kind), err)
		rjc.recorder.Event(obj, corev1.EventTypeWarning, reason, msg)
	}
}

var _ JobControlInterface = &realJobControl{}

// FakeJobControl is a fake JobControlInterface
type FakeJobControl struct {
	JobLister        batchlisters.JobLister
	JobIndexer       cache.Indexer
	createJobTracker RequestTracker
	deleteJobTracker RequestTracker
}

// NewFakeJobControl returns a FakeJobControl
func NewFakeJobControl(jobInformer batchinformers.JobInformer) *FakeJobControl {
	return &FakeJobControl{
		jobInformer.Lister(),
		jobInformer.Informer().GetIndexer(),
		RequestTracker{},
		RequestTracker{},
	}
}

// SetCreateJobError sets the error attributes of createJobTracker
func (fjc *FakeJobControl) SetCreateJobError(err error, after int) {
	fjc.createJobTracker.SetError(err).SetAfter(after)
}

// SetDeleteJobError sets the error attributes of deleteJobTracker
func (fjc *FakeJobControl) SetDeleteJobError(err error, after int) {
	fjc.deleteJobTracker.SetError(err).SetAfter(after)
}

// CreateJob adds the job to JobIndexer
func (fjc *FakeJobControl) CreateJob(_ runtime.Object, job *batchv1.Job) error {
	defer fjc.createJobTracker.Inc()
	if fjc.createJobTracker.ErrorReady() {
		defer fjc.createJobTracker.Reset()
		return fjc.createJobTracker.GetError()
	}

	return fjc.JobIndexer.Add(job)
}

// DeleteJob deletes the job
func (fjc *FakeJobControl) DeleteJob(_ runtime.Object, _ *batchv1.Job) error {
	defer fjc.deleteJobTracker.Inc()
	if fjc.deleteJobTracker.ErrorReady() {
		defer fjc.deleteJobTracker.Reset()
		return fjc.deleteJobTracker.GetError()
	}
	return nil
}

var _ JobControlInterface = &FakeJobControl{}
